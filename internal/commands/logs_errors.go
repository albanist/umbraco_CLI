package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"umbraco-cli/internal/api"
)

// logsErrors answers "is anything broken, and is any of it NEW?" after an
// incident or deployment. Without --distinct it is a flat Error+Fatal list;
// with --distinct entries collapse into fingerprint-grouped error classes so
// one chronic error repeating every few seconds reads as one class, not a
// wall of entries. Known-chronic classes are suppressed per invocation via
// --suppress/--suppress-contains — deliberately configuration, not code,
// because chronic noise differs per site and changes as bugs are fixed.
func logsErrors(deps Dependencies) *cobra.Command {
	var since string
	var until string
	var distinct bool
	var suppress []string
	var suppressContains []string
	var maxEntries int

	cmd := &cobra.Command{
		Use:   "errors",
		Short: "List Error/Fatal log entries, optionally grouped into distinct error classes",
		Long:  "GET /log-viewer/log filtered to Error and Fatal levels. --distinct groups entries into error classes by fingerprint (message template + normalized exception head, so two SQL violations with different constraint names are different classes) and reports count, first/last seen, and an example per class, sorted newest-first-seen so new breakage tops the list. --suppress drops known-chronic classes by fingerprint; --suppress-contains drops classes whose template or exception matches a substring. Suppressed classes are counted in the summary so they never vanish silently.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			startDate := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
			if strings.TrimSpace(since) != "" {
				parsed, err := parseLogTime(since)
				if err != nil {
					return fmt.Errorf("invalid --since: %w", err)
				}
				startDate = parsed.Format(time.RFC3339Nano)
			}
			params := map[string]any{
				"logLevel":  []any{"Error", "Fatal"},
				"startDate": startDate,
			}
			if strings.TrimSpace(until) != "" {
				parsed, err := parseLogTime(until)
				if err != nil {
					return fmt.Errorf("invalid --until: %w", err)
				}
				params["endDate"] = parsed.Format(time.RFC3339Nano)
			}

			result, err := getAllPagesWithFallback(
				cmd.Context(),
				deps.Client,
				0, 0, maxEntries,
				getRequestCandidate{path: logViewerLogPath, opts: api.RequestOptions{Params: params}},
				getRequestCandidate{path: logViewerLegacyListPath, opts: api.RequestOptions{Params: params}},
			)
			if err != nil {
				return friendlyLogViewerError(err)
			}
			envelope, ok := result.(map[string]any)
			if !ok {
				return printResult(cmd, deps, result)
			}
			items, _ := envelope["items"].([]any)
			if maxEntries > 0 && len(items) >= maxEntries {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: hit --max-entries cap of %d; older entries in the window were not scanned\n", maxEntries)
			}

			if !distinct {
				if len(suppress) > 0 || len(suppressContains) > 0 {
					return fmt.Errorf("--suppress and --suppress-contains require --distinct")
				}
				return printResult(cmd, deps, envelope)
			}

			groups, suppressed := groupErrorClasses(items, suppress, suppressContains)
			return printResult(cmd, deps, map[string]any{
				"classes":          groups,
				"totalEntries":     len(items),
				"suppressedGroups": suppressed,
				"since":            startDate,
			})
		},
	}

	cmd.Flags().StringVar(&since, "since", "", "Start of the window (ISO/RFC3339); default: 24 hours ago")
	cmd.Flags().StringVar(&until, "until", "", "End of the window (ISO/RFC3339); default: now")
	cmd.Flags().BoolVar(&distinct, "distinct", false, "Group entries into fingerprinted error classes")
	cmd.Flags().StringArrayVar(&suppress, "suppress", nil, "Fingerprint of a known-chronic error class to drop (repeatable)")
	cmd.Flags().StringArrayVar(&suppressContains, "suppress-contains", nil, "Drop classes whose template or exception contains this substring (repeatable)")
	cmd.Flags().IntVar(&maxEntries, "max-entries", 10000, "Maximum entries to scan in the window")
	return cmd
}

// errorClass is one fingerprinted group of Error/Fatal entries.
type errorClass struct {
	Fingerprint     string   `json:"fingerprint"`
	Count           int      `json:"count"`
	Levels          []string `json:"levels"`
	FirstSeen       string   `json:"firstSeen"`
	LastSeen        string   `json:"lastSeen"`
	MessageTemplate string   `json:"messageTemplate"`
	ExceptionHead   string   `json:"exceptionHead,omitempty"`
	ExampleMessage  string   `json:"exampleMessage,omitempty"`
	SourceContexts  []string `json:"sourceContexts,omitempty"`
}

func groupErrorClasses(items []any, suppress []string, suppressContains []string) ([]errorClass, int) {
	suppressed := map[string]struct{}{}
	for _, fp := range suppress {
		suppressed[strings.TrimSpace(fp)] = struct{}{}
	}

	byFingerprint := map[string]*errorClass{}
	sourceSets := map[string]map[string]struct{}{}
	levelSets := map[string]map[string]struct{}{}
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		template, _ := entry["messageTemplate"].(string)
		exception, _ := entry["exception"].(string)
		head := normalizeExceptionHead(exception)
		fingerprint := errorFingerprint(template, head)

		class, exists := byFingerprint[fingerprint]
		if !exists {
			class = &errorClass{
				Fingerprint:     fingerprint,
				MessageTemplate: template,
				ExceptionHead:   head,
			}
			if rendered, _ := entry["renderedMessage"].(string); rendered != "" {
				class.ExampleMessage = truncateForDisplay(rendered, 300)
			}
			byFingerprint[fingerprint] = class
			sourceSets[fingerprint] = map[string]struct{}{}
			levelSets[fingerprint] = map[string]struct{}{}
		}
		class.Count++
		timestamp, _ := entry["timestamp"].(string)
		if class.FirstSeen == "" || timestamp < class.FirstSeen {
			class.FirstSeen = timestamp
		}
		if timestamp > class.LastSeen {
			class.LastSeen = timestamp
		}
		if level, _ := entry["level"].(string); level != "" {
			levelSets[fingerprint][level] = struct{}{}
		}
		if properties, ok := entry["properties"].([]any); ok {
			for _, p := range properties {
				prop, ok := p.(map[string]any)
				if !ok {
					continue
				}
				if name, _ := prop["name"].(string); name == "SourceContext" {
					if value, _ := prop["value"].(string); value != "" {
						sourceSets[fingerprint][value] = struct{}{}
					}
				}
			}
		}
	}

	suppressedCount := 0
	classes := make([]errorClass, 0, len(byFingerprint))
	for fingerprint, class := range byFingerprint {
		if isSuppressedClass(class, suppressed, suppressContains) {
			suppressedCount++
			continue
		}
		class.Levels = sortedKeys(levelSets[fingerprint])
		class.SourceContexts = sortedKeys(sourceSets[fingerprint])
		classes = append(classes, *class)
	}
	// Newest-first-seen so brand-new breakage tops the list; chronic classes
	// that predate the window sink to the bottom.
	sort.Slice(classes, func(i, j int) bool {
		if classes[i].FirstSeen != classes[j].FirstSeen {
			return classes[i].FirstSeen > classes[j].FirstSeen
		}
		return classes[i].Fingerprint < classes[j].Fingerprint
	})
	return classes, suppressedCount
}

func isSuppressedClass(class *errorClass, suppressed map[string]struct{}, suppressContains []string) bool {
	if _, ok := suppressed[class.Fingerprint]; ok {
		return true
	}
	for _, needle := range suppressContains {
		needle = strings.TrimSpace(needle)
		if needle == "" {
			continue
		}
		if strings.Contains(class.MessageTemplate, needle) || strings.Contains(class.ExceptionHead, needle) || strings.Contains(class.ExampleMessage, needle) {
			return true
		}
	}
	return false
}

var (
	errorHexPattern  = regexp.MustCompile(`0x[0-9a-fA-F]+`)
	errorGUIDPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	errorNumPattern  = regexp.MustCompile(`\d+`)
)

// normalizeExceptionHead reduces an exception blob to its stable first line:
// hex codes, GUIDs, and numbers become placeholders so occurrences
// fingerprint identically, while quoted identifiers (constraint names,
// aliases) are kept — they are what distinguishes one SQL violation class
// from another.
func normalizeExceptionHead(exception string) string {
	head := strings.TrimSpace(exception)
	if head == "" {
		return ""
	}
	if index := strings.IndexAny(head, "\r\n"); index >= 0 {
		head = head[:index]
	}
	head = errorGUIDPattern.ReplaceAllString(head, "<guid>")
	head = errorHexPattern.ReplaceAllString(head, "0x#")
	head = errorNumPattern.ReplaceAllString(head, "#")
	return truncateForDisplay(head, 300)
}

func errorFingerprint(template string, exceptionHead string) string {
	sum := sha256.Sum256([]byte(template + "\n" + exceptionHead))
	return hex.EncodeToString(sum[:])[:12]
}

func truncateForDisplay(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "…"
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
