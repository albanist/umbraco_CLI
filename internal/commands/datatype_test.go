package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"umbraco-cli/internal/api"
	"umbraco-cli/internal/auth"
	"umbraco-cli/internal/config"
)

type datatypeRoundTripper func(*http.Request) (*http.Response, error)

func (fn datatypeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func datatypeJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func datatypeDeps(handler datatypeRoundTripper) Dependencies {
	cfg := config.Config{
		BaseURL:      "https://example.test",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	}
	httpClient := &http.Client{Transport: handler}
	output := "json"

	return Dependencies{
		Client:     api.NewClient(cfg, httpClient, auth.New(cfg, httpClient)),
		EnvOutput:  config.OutputJSON,
		OutputFlag: &output,
	}
}

func TestDatatypeListUsesFilterEndpointWithPagination(t *testing.T) {
	var observedPath string

	deps := datatypeDeps(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			return datatypeJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		case "/umbraco/management/api/v1/filter/data-type":
			observedPath = req.URL.String()
			return datatypeJSONResponse(http.StatusOK, `{"total":1,"items":[{"id":"dt-1","name":"Article Grid"}]}`), nil
		default:
			return datatypeJSONResponse(http.StatusNotFound, `null`), nil
		}
	})

	output, err := execute(buildRootWithCollections(t, deps), "datatype", "list", "--skip", "5", "--take", "20")
	if err != nil {
		t.Fatalf("datatype list failed: %v", err)
	}

	if !strings.Contains(observedPath, "skip=5") || !strings.Contains(observedPath, "take=20") {
		t.Fatalf("expected pagination params on filter endpoint, got %q", observedPath)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to decode datatype list payload: %v", err)
	}
	if payload["total"] != float64(1) {
		t.Fatalf("unexpected datatype list payload: %+v", payload)
	}
}

func TestDatatypeSearchFallsBackToFilterEndpointWhenItemSearchIsMissing(t *testing.T) {
	var itemSearchRequests int
	var observedFilterPath string

	deps := datatypeDeps(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			return datatypeJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		case "/umbraco/management/api/v1/item/data-type/search":
			itemSearchRequests++
			return datatypeJSONResponse(http.StatusNotFound, `null`), nil
		case "/umbraco/management/api/v1/filter/data-type":
			observedFilterPath = req.URL.String()
			return datatypeJSONResponse(http.StatusOK, `{"total":1,"items":[{"id":"dt-1","name":"Google Docs"}]}`), nil
		default:
			return datatypeJSONResponse(http.StatusNotFound, `null`), nil
		}
	})

	output, err := execute(buildRootWithCollections(t, deps), "datatype", "search", "--query", "google", "--skip", "2", "--take", "15")
	if err != nil {
		t.Fatalf("datatype search failed: %v", err)
	}

	if itemSearchRequests != 1 {
		t.Fatalf("expected one request to item search endpoint, got %d", itemSearchRequests)
	}
	if !strings.Contains(observedFilterPath, "filter=google") || !strings.Contains(observedFilterPath, "skip=2") || !strings.Contains(observedFilterPath, "take=15") {
		t.Fatalf("expected mapped fallback filter params, got %q", observedFilterPath)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to decode datatype search payload: %v", err)
	}
	if payload["total"] != float64(1) {
		t.Fatalf("unexpected datatype search payload: %+v", payload)
	}
}

func TestDatatypeRootUsesTreeRootEndpoint(t *testing.T) {
	var observedPath string

	deps := datatypeDeps(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			return datatypeJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		case "/umbraco/management/api/v1/tree/data-type/root":
			observedPath = req.URL.String()
			return datatypeJSONResponse(http.StatusOK, `{"total":1,"items":[{"id":"root-1","name":"Root"}]}`), nil
		default:
			return datatypeJSONResponse(http.StatusNotFound, `null`), nil
		}
	})

	output, err := execute(buildRootWithCollections(t, deps), "datatype", "root", "--skip", "1", "--take", "10")
	if err != nil {
		t.Fatalf("datatype root failed: %v", err)
	}

	if !strings.Contains(observedPath, "skip=1") || !strings.Contains(observedPath, "take=10") {
		t.Fatalf("expected pagination params on tree root endpoint, got %q", observedPath)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to decode datatype root payload: %v", err)
	}
	if payload["total"] != float64(1) {
		t.Fatalf("unexpected datatype root payload: %+v", payload)
	}
}
