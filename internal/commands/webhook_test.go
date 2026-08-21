package commands

import (
	"strings"
	"testing"
)

func TestNormalizeWebhookEventsMapsObjectsToAliases(t *testing.T) {
	body := map[string]any{"events": []any{
		map[string]any{"eventName": "Content Deleted", "eventType": "Content", "alias": "Umbraco.ContentDelete"},
		"Umbraco.ContentPublish",
	}}
	if err := normalizeWebhookEvents(body); err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	events := body["events"].([]any)
	if events[0] != "Umbraco.ContentDelete" || events[1] != "Umbraco.ContentPublish" {
		t.Fatalf("expected alias strings, got %+v", events)
	}
	// Idempotent: a second pass must not change anything.
	if err := normalizeWebhookEvents(body); err != nil {
		t.Fatalf("second normalize failed: %v", err)
	}
	if body["events"].([]any)[0] != "Umbraco.ContentDelete" {
		t.Fatalf("expected idempotent normalization, got %+v", body["events"])
	}
}

func TestNormalizeWebhookEventsRejectsAliaslessObjects(t *testing.T) {
	body := map[string]any{"events": []any{map[string]any{"eventName": "X"}}}
	if err := normalizeWebhookEvents(body); err == nil || !strings.Contains(err.Error(), "without an alias") {
		t.Fatalf("expected aliasless rejection, got %v", err)
	}
}

func TestNormalizeWebhookEventsLeavesAbsentFieldAlone(t *testing.T) {
	body := map[string]any{"enabled": true}
	if err := normalizeWebhookEvents(body); err != nil {
		t.Fatalf("expected no-op on absent events, got %v", err)
	}
}
