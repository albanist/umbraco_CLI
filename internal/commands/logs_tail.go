package commands

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"umbraco-cli/internal/api"
	"umbraco-cli/internal/config"
)

// tailPageSize bounds each poll; bursts larger than one page are picked up
// on the next poll because the cursor only advances past fetched entries.
const tailPageSize = 500

func logsTail(deps Dependencies) *cobra.Command {
	var flags logQueryFlags
	flags.skip = -1
	flags.take = -1
	var since string
	var interval time.Duration
	var forDuration time.Duration

	cmd := &cobra.Command{
		Use:   "tail",
		Short: "Follow new log entries as they arrive (client-side polling)",
		Long: `The Management API has no streaming endpoint, so tail polls the log-viewer
with a moving startDate cursor, deduplicates boundary entries, and prints
each new entry exactly once: NDJSON (one JSON object per line) for json
output, one formatted line per entry otherwise. Runs until interrupted or
--for elapses; exits 0 on both.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			baseParams, runtime, err := logParamsFromFlags("", flags)
			if err != nil {
				return err
			}
			format, err := resolveOutputFormat(deps)
			if err != nil {
				return err
			}

			cursor := time.Now().UTC()
			if strings.TrimSpace(since) != "" {
				parsed, err := parseLogTime(since)
				if err != nil {
					return fmt.Errorf("invalid --since: %w", err)
				}
				cursor = parsed.UTC()
			}

			out := cmd.OutOrStdout()
			printEntry := func(entry map[string]any) error {
				shaped := any(entry)
				if runtime.flat {
					shaped = flattenLogEntry(entry)
				}
				if runtime.redaction.enabled() {
					shaped = redactLogValue(shaped, runtime.redaction)
				}
				if format == config.OutputJSON {
					encoded, err := json.Marshal(shaped)
					if err != nil {
						return err
					}
					_, err = fmt.Fprintln(out, string(encoded))
					return err
				}
				message := firstLogString(entry["renderedMessage"], entry["messageTemplate"], entry["message"])
				if runtime.redaction.enabled() {
					message = redactLogString(message, runtime.redaction)
				}
				_, err := fmt.Fprintf(out, "%s [%s] %s\n", stringValue(entry["timestamp"]), stringValue(entry["level"]), message)
				return err
			}

			ctx := cmd.Context()
			seen := map[string]struct{}{}
			var deadline time.Time
			if forDuration > 0 {
				deadline = time.Now().Add(forDuration)
			}

			for {
				params := copyAnyMap(baseParams)
				params["startDate"] = cursor.Format(time.RFC3339)
				params["take"] = tailPageSize
				result, err := getWithFallback(ctx, deps.Client,
					getRequestCandidate{path: logViewerLogPath, opts: api.RequestOptions{Params: params}},
					getRequestCandidate{path: logViewerLegacyListPath, opts: api.RequestOptions{Params: params}},
				)
				if err != nil {
					// An interrupt mid-request is a clean stop, not a failure.
					if ctx.Err() != nil {
						return nil
					}
					return friendlyLogViewerError(err)
				}

				type stampedEntry struct {
					entry map[string]any
					ts    time.Time
				}
				fresh := make([]stampedEntry, 0)
				for _, item := range resultItems(result) {
					entry, ok := item.(map[string]any)
					if !ok {
						continue
					}
					ts, ok := logEntryTimestamp(entry)
					if !ok || ts.Before(cursor) {
						continue
					}
					fresh = append(fresh, stampedEntry{entry: entry, ts: ts})
				}
				sort.SliceStable(fresh, func(i, j int) bool { return fresh[i].ts.Before(fresh[j].ts) })

				nextCursor := cursor
				for _, stamped := range fresh {
					key := stableJSON(stamped.entry)
					if _, duplicate := seen[key]; !duplicate && logEntryMatches(stamped.entry, runtime) {
						if err := printEntry(stamped.entry); err != nil {
							return err
						}
					}
					if stamped.ts.After(nextCursor) {
						nextCursor = stamped.ts
					}
				}

				// The startDate cursor has second precision, so entries within
				// a second of the new cursor can reappear on the next poll;
				// remember their identities to print each entry exactly once.
				if len(fresh) > 0 {
					nextSeen := map[string]struct{}{}
					boundary := nextCursor.Add(-time.Second)
					for _, stamped := range fresh {
						if !stamped.ts.Before(boundary) {
							nextSeen[stableJSON(stamped.entry)] = struct{}{}
						}
					}
					seen = nextSeen
					cursor = nextCursor
				}

				if !deadline.IsZero() && time.Now().After(deadline) {
					return nil
				}
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(interval):
				}
			}
		},
	}

	cmd.Flags().StringVar(&flags.level, "level", "", "Only entries at this level (Verbose, Debug, Information, Warning, Error, Fatal)")
	cmd.Flags().StringVar(&flags.filterExpression, "filter-expression", "", "Serilog filter expression (server-side)")
	cmd.Flags().StringVar(&flags.sourceContext, "source-context", "", "Only entries whose SourceContext contains this value")
	cmd.Flags().StringVar(&flags.path, "path", "", "Only entries whose RequestPath contains this value")
	cmd.Flags().StringVar(&flags.contains, "contains", "", "Only entries containing this substring anywhere")
	cmd.Flags().StringVar(&flags.correlationID, "correlation-id", "", "Only entries with this correlation/request id")
	cmd.Flags().BoolVar(&flags.flat, "flat", false, "Flatten entries (timestamp, level, message, sourceContext, ...)")
	cmd.Flags().StringVar(&flags.redact, "redact", "", "Redact matching value kinds: emails,secrets,tokens (comma-separated)")
	cmd.Flags().BoolVar(&flags.redactDefault, "redact-default", false, "Apply the default redaction set (emails, secrets, tokens)")
	cmd.Flags().StringVar(&since, "since", "", "Start from this timestamp (RFC3339); default: now (only new entries)")
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "Poll interval")
	cmd.Flags().DurationVar(&forDuration, "for", 0, "Stop after this duration (0 = run until interrupted); exits 0")
	return cmd
}
