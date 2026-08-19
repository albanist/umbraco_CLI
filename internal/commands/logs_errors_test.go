package commands

import (
	"strings"
	"testing"
	"time"
)

func errorEntry(timestamp, level, template, exception, source string) map[string]any {
	return map[string]any{
		"timestamp":       timestamp,
		"level":           level,
		"messageTemplate": template,
		"exception":       exception,
		"renderedMessage": "rendered " + template,
		"properties":      []any{map[string]any{"name": "SourceContext", "value": source}},
	}
}

func TestGroupErrorClassesSeparatesByNormalizedExceptionHead(t *testing.T) {
	items := []any{
		errorEntry("2026-08-19T10:00:00Z", "Error", "An exception occurred", "SqlException (0x80131904): Violation of PRIMARY KEY constraint 'PK_umbracoAutomateWorkflowLock'.\nstack", "Ctx.A"),
		errorEntry("2026-08-19T10:00:10Z", "Error", "An exception occurred", "SqlException (0x80131904): Violation of PRIMARY KEY constraint 'PK_umbracoAutomateWorkflowLock'.\nother stack", "Ctx.A"),
		errorEntry("2026-08-19T10:01:00Z", "Fatal", "An exception occurred", "SqlException (0x80131904): Violation of PRIMARY KEY constraint 'PK_otherTable'.\nstack", "Ctx.B"),
	}

	classes, suppressed := groupErrorClasses(items, nil, nil)
	if suppressed != 0 {
		t.Fatalf("expected no suppression, got %d", suppressed)
	}
	if len(classes) != 2 {
		t.Fatalf("expected two classes (distinct constraint names), got %+v", classes)
	}
	// Newest-first-seen ordering: PK_otherTable (10:01) before the chronic one.
	if classes[0].Count != 1 || classes[1].Count != 2 {
		t.Fatalf("expected newest-first-seen ordering with counts 1,2, got %+v", classes)
	}
	if classes[0].Levels[0] != "Fatal" {
		t.Fatalf("expected level recorded, got %+v", classes[0].Levels)
	}
	if classes[1].FirstSeen != "2026-08-19T10:00:00Z" || classes[1].LastSeen != "2026-08-19T10:00:10Z" {
		t.Fatalf("expected first/last seen tracked, got %+v", classes[1])
	}
}

func TestGroupErrorClassesFingerprintStableAcrossVolatileValues(t *testing.T) {
	a := errorFingerprint("tpl", normalizeExceptionHead("SqlException (0x80131904): timeout after 31 ms, id 6a2b8c9d-1111-2222-3333-444455556666"))
	b := errorFingerprint("tpl", normalizeExceptionHead("SqlException (0xDEADBEEF): timeout after 9000 ms, id ffffffff-aaaa-bbbb-cccc-000011112222"))
	if a != b {
		t.Fatalf("expected hex/number/guid normalization to fingerprint identically, got %s vs %s", a, b)
	}
	c := errorFingerprint("tpl", normalizeExceptionHead("SqlException (0x1): Violation of PRIMARY KEY constraint 'PK_x'"))
	if a == c {
		t.Fatalf("expected different exception text to fingerprint differently")
	}
}

func TestGroupErrorClassesSuppression(t *testing.T) {
	items := []any{
		errorEntry("2026-08-19T10:00:00Z", "Error", "chronic template", "SqlException: Violation of PRIMARY KEY constraint 'PK_umbracoAutomateWorkflowLock'", "Ctx"),
		errorEntry("2026-08-19T10:02:00Z", "Error", "new template", "", "Ctx"),
	}

	classes, suppressed := groupErrorClasses(items, nil, []string{"PK_umbracoAutomateWorkflowLock"})
	if suppressed != 1 || len(classes) != 1 || classes[0].MessageTemplate != "new template" {
		t.Fatalf("expected chronic class suppressed by substring, got classes=%+v suppressed=%d", classes, suppressed)
	}

	fingerprint := errorFingerprint("new template", "")
	classes, suppressed = groupErrorClasses(items, []string{fingerprint}, nil)
	if suppressed != 1 || len(classes) != 1 || classes[0].MessageTemplate != "chronic template" {
		t.Fatalf("expected class suppressed by fingerprint, got classes=%+v suppressed=%d", classes, suppressed)
	}
}

func TestFilterEntriesToWindow(t *testing.T) {
	start := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 19, 11, 0, 0, 0, time.UTC)
	items := []any{
		errorEntry("2026-08-19T09:59:00Z", "Error", "before", "", "Ctx"),
		errorEntry("2026-08-19T10:30:00+00:00", "Error", "inside", "", "Ctx"),
		errorEntry("2026-08-19T12:30:00+02:00", "Error", "inside-offset", "", "Ctx"),
		errorEntry("2026-08-19T11:00:01Z", "Error", "after", "", "Ctx"),
		errorEntry("not-a-time", "Error", "kept-unparsable", "", "Ctx"),
	}
	filtered := filterEntriesToWindow(items, start, end)
	templates := make([]string, 0, len(filtered))
	for _, item := range filtered {
		templates = append(templates, item.(map[string]any)["messageTemplate"].(string))
	}
	want := "inside,inside-offset,kept-unparsable"
	if strings.Join(templates, ",") != want {
		t.Fatalf("expected %s, got %v", want, templates)
	}
}

func TestGroupErrorClassesNullFingerprintFieldsFallBackToRenderedMessage(t *testing.T) {
	a := map[string]any{"timestamp": "2026-08-19T10:00:00Z", "level": "Error", "renderedMessage": "boom in module A", "properties": []any{}}
	b := map[string]any{"timestamp": "2026-08-19T10:01:00Z", "level": "Error", "renderedMessage": "totally different failure B", "properties": []any{}}
	classes, _ := groupErrorClasses([]any{a, b}, nil, nil)
	if len(classes) != 2 {
		t.Fatalf("expected null template+exception entries to fingerprint by rendered message, got %d classes: %+v", len(classes), classes)
	}
}
