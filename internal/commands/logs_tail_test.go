package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
)

func buildLogsRoot(deps Dependencies) *cobra.Command {
	root := &cobra.Command{Use: "umbraco", SilenceErrors: true, SilenceUsage: true}
	root.SetErr(io.Discard)
	if deps.OutputFlag != nil {
		root.PersistentFlags().StringVarP(deps.OutputFlag, "output", "o", *deps.OutputFlag, "Output format: json, table, plain")
	}
	RegisterLogs(root, deps)
	return root
}

func logsTailDeps(handler func(poll int64, req *http.Request) (*http.Response, error)) Dependencies {
	var polls int64
	return endpointDeps(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/umbraco/management/api/v1/security/back-office/token" {
			return endpointJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		}
		return handler(atomic.AddInt64(&polls, 1), req)
	})
}

func TestLogsTailPrintsEachEntryOnceAsNDJSON(t *testing.T) {
	var lastStartDate atomic.Value
	deps := logsTailDeps(func(poll int64, req *http.Request) (*http.Response, error) {
		lastStartDate.Store(req.URL.Query().Get("startDate"))
		if poll == 1 {
			return endpointJSONResponse(http.StatusOK, `{"items":[
				{"timestamp":"2026-07-03T10:00:01Z","level":"Information","renderedMessage":"first"},
				{"timestamp":"2026-07-03T10:00:02Z","level":"Information","renderedMessage":"second"}
			],"total":2}`), nil
		}
		// Later polls replay the boundary entry plus one new one.
		return endpointJSONResponse(http.StatusOK, `{"items":[
			{"timestamp":"2026-07-03T10:00:02Z","level":"Information","renderedMessage":"second"},
			{"timestamp":"2026-07-03T10:00:03Z","level":"Information","renderedMessage":"third"}
		],"total":2}`), nil
	})

	out, err := execute(buildLogsRoot(deps), "logs", "tail", "--since", "2026-07-03T10:00:00Z", "--interval", "1ms", "--for", "40ms")
	if err != nil {
		t.Fatalf("logs tail failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected exactly 3 entries printed once each, got %d lines:\n%s", len(lines), out)
	}
	for i, expected := range []string{"first", "second", "third"} {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(lines[i]), &decoded); err != nil {
			t.Fatalf("line %d is not valid NDJSON: %v\n%s", i, err, lines[i])
		}
		if decoded["renderedMessage"] != expected {
			t.Fatalf("line %d: expected message %q, got %v", i, expected, decoded["renderedMessage"])
		}
	}
	if got := lastStartDate.Load().(string); got != "2026-07-03T10:00:03Z" {
		t.Fatalf("expected cursor to advance to the newest entry, got startDate %q", got)
	}
}

func TestLogsTailFiltersByLevelClientSide(t *testing.T) {
	deps := logsTailDeps(func(poll int64, req *http.Request) (*http.Response, error) {
		return endpointJSONResponse(http.StatusOK, `{"items":[
			{"timestamp":"2026-07-03T10:00:01Z","level":"Information","renderedMessage":"noise"},
			{"timestamp":"2026-07-03T10:00:02Z","level":"Error","renderedMessage":"boom"}
		],"total":2}`), nil
	})

	out, err := execute(buildLogsRoot(deps), "logs", "tail", "--since", "2026-07-03T10:00:00Z", "--level", "Error", "--interval", "1ms", "--for", "20ms")
	if err != nil {
		t.Fatalf("logs tail failed: %v", err)
	}
	if strings.Contains(out, "noise") {
		t.Fatalf("expected Information entry filtered out, got %s", out)
	}
	if strings.Count(out, "boom") != 1 {
		t.Fatalf("expected the Error entry exactly once, got %s", out)
	}
}

func TestLogsTailPlainOutputFormatsLines(t *testing.T) {
	deps := logsTailDeps(func(poll int64, req *http.Request) (*http.Response, error) {
		return endpointJSONResponse(http.StatusOK, `{"items":[
			{"timestamp":"2026-07-03T10:00:01Z","level":"Warning","renderedMessage":"careful"}
		],"total":1}`), nil
	})

	out, err := execute(buildLogsRoot(deps), "logs", "tail", "-o", "plain", "--since", "2026-07-03T10:00:00Z", "--interval", "1ms", "--for", "20ms")
	if err != nil {
		t.Fatalf("logs tail failed: %v", err)
	}
	if !strings.Contains(out, "2026-07-03T10:00:01Z [Warning] careful") {
		t.Fatalf("expected formatted plain line, got %s", out)
	}
}

func TestLogsTailRedactsSensitiveValues(t *testing.T) {
	deps := logsTailDeps(func(poll int64, req *http.Request) (*http.Response, error) {
		return endpointJSONResponse(http.StatusOK, `{"items":[
			{"timestamp":"2026-07-03T10:00:01Z","level":"Information","renderedMessage":"login by user@example.test"}
		],"total":1}`), nil
	})

	out, err := execute(buildLogsRoot(deps), "logs", "tail", "--redact-default", "--since", "2026-07-03T10:00:00Z", "--interval", "1ms", "--for", "20ms")
	if err != nil {
		t.Fatalf("logs tail failed: %v", err)
	}
	if strings.Contains(out, "user@example.test") {
		t.Fatalf("expected email redacted, got %s", out)
	}
	if !strings.Contains(out, "login by") {
		t.Fatalf("expected entry still printed, got %s", out)
	}
}

func TestLogsTailRejectsInvalidSince(t *testing.T) {
	deps := logsTailDeps(func(poll int64, req *http.Request) (*http.Response, error) {
		t.Fatalf("no HTTP request expected for invalid --since")
		return nil, nil
	})

	if _, err := execute(buildLogsRoot(deps), "logs", "tail", "--since", "not-a-time", "--for", "10ms"); err == nil || !strings.Contains(err.Error(), "invalid --since") {
		t.Fatalf("expected invalid --since error, got %v", err)
	}
}

func TestLogsTailRequestsAscendingOrderAndDrainsBursts(t *testing.T) {
	// A burst larger than one page: the first poll returns a full ascending
	// page; the cursor must advance only past fetched entries so the second
	// poll picks up the remainder instead of skipping it.
	burstPage := func(start, count int) string {
		var b strings.Builder
		b.WriteString(`{"items":[`)
		for i := 0; i < count; i++ {
			if i > 0 {
				b.WriteString(",")
			}
			fmt.Fprintf(&b, `{"timestamp":"2026-07-03T10:%02d:%02dZ","level":"Information","renderedMessage":"entry-%d"}`, (start+i)/60, (start+i)%60, start+i)
		}
		fmt.Fprintf(&b, `],"total":%d}`, count)
		return b.String()
	}

	var orderDirections []string
	var startDates []string
	deps := logsTailDeps(func(poll int64, req *http.Request) (*http.Response, error) {
		orderDirections = append(orderDirections, req.URL.Query().Get("orderDirection"))
		startDates = append(startDates, req.URL.Query().Get("startDate"))
		if poll == 1 {
			return endpointJSONResponse(http.StatusOK, burstPage(1, tailPageSize)), nil
		}
		return endpointJSONResponse(http.StatusOK, burstPage(tailPageSize, 3)), nil
	})

	out, err := execute(buildLogsRoot(deps), "logs", "tail", "--since", "2026-07-03T10:00:00Z", "--interval", "1h", "--for", "50ms")
	if err != nil {
		t.Fatalf("logs tail failed: %v", err)
	}
	for _, direction := range orderDirections {
		if direction != "Ascending" {
			t.Fatalf("expected every poll to request ascending order, got %v", orderDirections)
		}
	}
	if len(startDates) < 2 {
		t.Fatalf("expected the full page to trigger an immediate drain poll despite the 1h interval, got %d polls", len(startDates))
	}
	printed := strings.Count(out, "entry-")
	if printed != tailPageSize+2 {
		// entry-500 appears in both pages (boundary) and must print once.
		t.Fatalf("expected %d unique entries printed, got %d", tailPageSize+2, printed)
	}
}
