package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"umbraco-cli/internal/auth"
	"umbraco-cli/internal/config"
)

func TestDryRunReturnsPreview(t *testing.T) {
	cfg := config.Config{BaseURL: "https://example.test"}
	client := NewClient(cfg, http.DefaultClient, nil)

	result, err := client.Post(context.Background(), "/document/abc-123/publish", map[string]any{"cultures": []any{"en-US"}}, RequestOptions{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run should not fail: %v", err)
	}

	dryRun, ok := result.(DryRunResult)
	if !ok {
		t.Fatalf("expected DryRunResult, got %T", result)
	}

	if !dryRun.DryRun || !dryRun.Valid || dryRun.Method != http.MethodPost {
		t.Fatalf("unexpected dry-run metadata: %+v", dryRun)
	}
	if dryRun.Path != "/umbraco/management/api/v1/document/abc-123/publish" {
		t.Fatalf("unexpected dry-run path: %s", dryRun.Path)
	}
}

func TestRequestBuildsURLAndUsesToken(t *testing.T) {
	var observedRequestPath string
	var observedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token-123","expires_in":3600}`))
		case "/umbraco/management/api/v1/document/root":
			observedRequestPath = r.URL.String()
			observedAuth = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"id":"root"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := config.Config{BaseURL: server.URL, ClientID: "client-id", ClientSecret: "client-secret"}
	httpClient := server.Client()
	tokenProvider := auth.New(cfg, httpClient)
	client := NewClient(cfg, httpClient, tokenProvider)

	result, err := client.Get(context.Background(), "/document/root", RequestOptions{Fields: "id,name", Params: map[string]any{"skip": 0, "take": 10, "culture": "en-US"}})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if !strings.Contains(observedRequestPath, "fields=id%2Cname") || !strings.Contains(observedRequestPath, "skip=0") || !strings.Contains(observedRequestPath, "take=10") {
		t.Fatalf("unexpected query string: %s", observedRequestPath)
	}
	if observedAuth != "Bearer token-123" {
		t.Fatalf("unexpected auth header: %s", observedAuth)
	}

	payload, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %T", result)
	}
	if _, ok := payload["items"]; !ok {
		t.Fatalf("expected items in response")
	}
}

func TestRequestReturnsAPIErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"token-123","expires_in":3600}`))
		case "/umbraco/management/api/v1/document/root":
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"error":"invalid request"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := config.Config{BaseURL: server.URL, ClientID: "client-id", ClientSecret: "client-secret"}
	httpClient := server.Client()
	client := NewClient(cfg, httpClient, auth.New(cfg, httpClient))

	_, err := client.Get(context.Background(), "/document/root", RequestOptions{})
	if err == nil {
		t.Fatalf("expected API error")
	}
	if !strings.Contains(err.Error(), "API 400") {
		t.Fatalf("expected status in error, got: %v", err)
	}
}

func TestDryRunBodySerializesConsistently(t *testing.T) {
	cfg := config.Config{BaseURL: "https://example.test"}
	client := NewClient(cfg, http.DefaultClient, nil)

	result, err := client.Post(context.Background(), "/document/abc-123/publish", map[string]any{"cultures": []any{"da-DK"}}, RequestOptions{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(encoded), `"dryRun":true`) {
		t.Fatalf("unexpected dry-run JSON: %s", string(encoded))
	}
	if !strings.Contains(string(encoded), `"da-DK"`) {
		t.Fatalf("expected body culture in JSON: %s", string(encoded))
	}
	_ = fmt.Sprintf("%s", encoded)
}
