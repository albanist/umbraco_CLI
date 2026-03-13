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

func TestDatatypeUpdateMergeJSONFetchesCurrentAndSendsMergedPayload(t *testing.T) {
	var putPayload map[string]any
	var getRequests int

	deps := datatypeDeps(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			return datatypeJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		case "/umbraco/management/api/v1/data-type/dt-1":
			if req.Method == http.MethodGet {
				getRequests++
				return datatypeJSONResponse(http.StatusOK, `{
  "id":"dt-1",
  "name":"Rich Text",
  "editorAlias":"Umb.PropertyEditorUi.Tiptap",
  "configuration":{
    "maxChars":120,
    "toolbar":{"bold":true,"italic":true}
  }
}`), nil
			}
			if req.Method == http.MethodPut {
				if err := json.NewDecoder(req.Body).Decode(&putPayload); err != nil {
					t.Fatalf("failed to decode put payload: %v", err)
				}
				return datatypeJSONResponse(http.StatusOK, `{"updated":true}`), nil
			}
			return datatypeJSONResponse(http.StatusMethodNotAllowed, `{"error":"method not allowed"}`), nil
		default:
			return datatypeJSONResponse(http.StatusNotFound, `null`), nil
		}
	})

	output, err := execute(
		buildRootWithCollections(t, deps),
		"datatype", "update", "dt-1",
		"--merge-json", `{"configuration":{"toolbar":{"italic":false}}}`,
	)
	if err != nil {
		t.Fatalf("datatype merge update failed: %v", err)
	}

	if getRequests != 1 {
		t.Fatalf("expected one fetch of the current datatype, got %d", getRequests)
	}
	if putPayload["name"] != "Rich Text" || putPayload["editorAlias"] != "Umb.PropertyEditorUi.Tiptap" {
		t.Fatalf("expected required fields to be preserved, got %+v", putPayload)
	}

	configuration, ok := putPayload["configuration"].(map[string]any)
	if !ok {
		t.Fatalf("missing merged configuration payload: %+v", putPayload)
	}
	if configuration["maxChars"] != float64(120) {
		t.Fatalf("expected untouched config fields to be preserved, got %+v", configuration)
	}
	toolbar, ok := configuration["toolbar"].(map[string]any)
	if !ok {
		t.Fatalf("missing merged toolbar payload: %+v", configuration)
	}
	if toolbar["bold"] != true || toolbar["italic"] != false {
		t.Fatalf("expected nested merge to preserve bold and update italic, got %+v", toolbar)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to decode datatype merge update result: %v", err)
	}
	if result["updated"] != true {
		t.Fatalf("unexpected update result payload: %+v", result)
	}
}

func TestDatatypeUpdateMergeJSONSupportsDryRunForNestedObjectConfig(t *testing.T) {
	var getRequests int

	deps := datatypeDeps(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			return datatypeJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		case "/umbraco/management/api/v1/data-type/dt-1":
			if req.Method == http.MethodGet {
				getRequests++
				return datatypeJSONResponse(http.StatusOK, `{
  "id":"dt-1",
  "name":"Rich Text",
  "editorAlias":"Umb.PropertyEditorUi.Tiptap",
  "configuration":{"toolbar":{"bold":true,"italic":true}}
}`), nil
			}
			return datatypeJSONResponse(http.StatusMethodNotAllowed, `{"error":"unexpected write"}`), nil
		default:
			return datatypeJSONResponse(http.StatusNotFound, `null`), nil
		}
	})

	output, err := execute(
		buildRootWithCollections(t, deps),
		"datatype", "update", "dt-1",
		"--merge-json", `{"configuration":{"toolbar":{"italic":false}}}`,
		"--dry-run",
	)
	if err != nil {
		t.Fatalf("datatype merge update dry-run failed: %v", err)
	}

	if getRequests != 1 {
		t.Fatalf("expected dry-run merge update to fetch the current datatype once, got %d", getRequests)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to decode dry-run payload: %v", err)
	}
	if payload["dryRun"] != true {
		t.Fatalf("expected dryRun=true, got %+v", payload)
	}
	body, ok := payload["body"].(map[string]any)
	if !ok {
		t.Fatalf("missing dry-run body: %+v", payload)
	}
	configuration, ok := body["configuration"].(map[string]any)
	if !ok {
		t.Fatalf("missing dry-run merged configuration: %+v", body)
	}
	toolbar, ok := configuration["toolbar"].(map[string]any)
	if !ok || toolbar["bold"] != true || toolbar["italic"] != false {
		t.Fatalf("unexpected dry-run merged toolbar payload: %+v", configuration)
	}
}

func TestDatatypeUpdateRejectsJSONAndMergeJSONTogether(t *testing.T) {
	deps := makeDeps()
	root := buildRootWithCollections(t, deps)

	_, err := execute(
		root,
		"datatype", "update", "dt-1",
		"--json", `{"name":"Full"}`,
		"--merge-json", `{"configuration":{"toolbar":{"italic":false}}}`,
	)
	if err == nil {
		t.Fatalf("expected datatype update to reject simultaneous --json and --merge-json")
	}
	if !strings.Contains(err.Error(), "exactly one of --json or --merge-json") {
		t.Fatalf("unexpected merge-json validation error: %v", err)
	}
}

func TestDatatypeUpdateMergeJSONMergesValuesByAliasAndPreservesRequiredFields(t *testing.T) {
	var observedPutBody map[string]any

	deps := datatypeDeps(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			return datatypeJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		case "/umbraco/management/api/v1/data-type/dt-1":
			if req.Method == http.MethodGet {
				return datatypeJSONResponse(http.StatusOK, `{
  "id":"dt-1",
  "name":"Rich Text",
  "editorAlias":"Umb.PropertyEditorUi.Tiptap",
  "values":[
    {"alias":"extensions","value":["Existing.Extension"]},
    {"alias":"toolbar","value":["bold","italic"]}
  ]
}`), nil
			}
			if req.Method == http.MethodPut {
				if err := json.NewDecoder(req.Body).Decode(&observedPutBody); err != nil {
					t.Fatalf("failed to decode merged datatype payload: %v", err)
				}
				return datatypeJSONResponse(http.StatusOK, `{"ok":true}`), nil
			}
			return datatypeJSONResponse(http.StatusMethodNotAllowed, `{"error":"method not allowed"}`), nil
		default:
			return datatypeJSONResponse(http.StatusNotFound, `null`), nil
		}
	})

	_, err := execute(
		buildRootWithCollections(t, deps),
		"datatype", "update", "dt-1",
		"--merge-json", `{"values":[{"alias":"extensions","value":["Existing.Extension","New.Extension"]}]}`,
	)
	if err != nil {
		t.Fatalf("datatype merge update failed: %v", err)
	}

	if observedPutBody["name"] != "Rich Text" || observedPutBody["editorAlias"] != "Umb.PropertyEditorUi.Tiptap" {
		t.Fatalf("expected required fields to be preserved, got %+v", observedPutBody)
	}

	values, ok := observedPutBody["values"].([]any)
	if !ok || len(values) != 2 {
		t.Fatalf("expected merged values array, got %+v", observedPutBody["values"])
	}

	var extensionsValue []any
	var toolbarValue []any
	for _, item := range values {
		itemMap, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("expected value item to be an object, got %T", item)
		}
		switch itemMap["alias"] {
		case "extensions":
			extensionsValue, _ = itemMap["value"].([]any)
		case "toolbar":
			toolbarValue, _ = itemMap["value"].([]any)
		}
	}

	if len(extensionsValue) != 2 || extensionsValue[1] != "New.Extension" {
		t.Fatalf("expected extensions alias to be updated, got %+v", extensionsValue)
	}
	if len(toolbarValue) != 2 || toolbarValue[0] != "bold" {
		t.Fatalf("expected unrelated alias entries to be preserved, got %+v", toolbarValue)
	}
}

func TestDatatypeUpdateMergeJSONSupportsDryRunForAliasValues(t *testing.T) {
	deps := datatypeDeps(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			return datatypeJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		case "/umbraco/management/api/v1/data-type/dt-1":
			if req.Method == http.MethodGet {
				return datatypeJSONResponse(http.StatusOK, `{
  "id":"dt-1",
  "name":"Rich Text",
  "editorAlias":"Umb.PropertyEditorUi.Tiptap",
  "values":[{"alias":"extensions","value":["Existing.Extension"]}]
}`), nil
			}
			return datatypeJSONResponse(http.StatusMethodNotAllowed, `{"error":"unexpected method"}`), nil
		default:
			return datatypeJSONResponse(http.StatusNotFound, `null`), nil
		}
	})

	output, err := execute(
		buildRootWithCollections(t, deps),
		"datatype", "update", "dt-1",
		"--merge-json", `{"values":[{"alias":"extensions","value":["Existing.Extension","New.Extension"]}]}`,
		"--dry-run",
	)
	if err != nil {
		t.Fatalf("datatype merge dry-run failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to decode datatype merge dry-run payload: %v", err)
	}
	if payload["dryRun"] != true {
		t.Fatalf("expected dryRun=true, got %+v", payload)
	}

	body, ok := payload["body"].(map[string]any)
	if !ok {
		t.Fatalf("expected dry-run body to be an object, got %+v", payload["body"])
	}
	if body["name"] != "Rich Text" || body["editorAlias"] != "Umb.PropertyEditorUi.Tiptap" {
		t.Fatalf("expected merged dry-run payload to preserve required fields, got %+v", body)
	}
}
