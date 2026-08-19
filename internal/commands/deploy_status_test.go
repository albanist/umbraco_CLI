package commands

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func buildDeployRoot(deps Dependencies) *cobra.Command {
	root := &cobra.Command{Use: "umbraco", SilenceErrors: true, SilenceUsage: true}
	root.SetErr(os.NewFile(0, os.DevNull))
	if deps.OutputFlag != nil {
		root.PersistentFlags().StringVarP(deps.OutputFlag, "output", "o", *deps.OutputFlag, "Output format: json, table, plain")
	}
	RegisterDeploy(root, deps)
	return root
}

// writeUda writes a fixture artifact UTF-8 with BOM, exactly as Umbraco
// Deploy writes them — json parsing must strip it.
func writeUda(t *testing.T, dir string, name string, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), append([]byte{0xEF, 0xBB, 0xBF}, []byte(body)...), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func TestParseUdi(t *testing.T) {
	kind, guid := parseUdi("umb://data-type/c6bac0dd4ab945b18e30e4b619ee5da3")
	if kind != "data-type" || guid != "c6bac0dd-4ab9-45b1-8e30-e4b619ee5da3" {
		t.Fatalf("unexpected parse: %q %q", kind, guid)
	}
	if kind, guid := parseUdi("not-a-udi"); kind != "" || guid != "" {
		t.Fatalf("expected empty parse, got %q %q", kind, guid)
	}
}

const statusDataTypeUda = `{
  "Name": "Example Text",
  "EditorAlias": "Umbraco.TextArea",
  "EditorUiAlias": "Umb.PropertyEditorUi.TextArea",
  "Configuration": {"maxChars": 500},
  "Udi": "umb://data-type/aaaaaaaa111122223333444444444444",
  "Dependencies": [],
  "__type": "Umbraco.Deploy.Infrastructure,Umbraco.Deploy.Infrastructure.Artifacts.DataTypeArtifact",
  "__version": "18.0.1"
}`

const statusDoctypeUda = `{
  "Name": "Example Block",
  "Alias": "exampleBlock",
  "Icon": "icon-box",
  "Permissions": {"IsElementType": true, "AllowedChildContentTypes": []},
  "PropertyGroups": [
    {"Key": "11111111-1111-1111-1111-111111111111", "Name": "Content", "Alias": "content", "PropertyTypes": [
      {"Key": "22222222-2222-2222-2222-222222222222", "Alias": "heading", "DataType": "umb://data-type/aaaaaaaa111122223333444444444444", "ValueType": "System.String", "Name": "Heading", "SortOrder": 0}
    ]}
  ],
  "PropertyTypes": [
    {"Key": "33333333-3333-3333-3333-333333333333", "Alias": "ungrouped", "DataType": "umb://data-type/aaaaaaaa111122223333444444444444", "ValueType": "System.String", "Name": "Ungrouped", "SortOrder": 5}
  ],
  "Udi": "umb://document-type/bbbbbbbb111122223333444444444444",
  "Dependencies": [{"Udi": "umb://data-type/aaaaaaaa111122223333444444444444", "Ordering": true}],
  "__type": "Umbraco.Deploy.Infrastructure,Umbraco.Deploy.Infrastructure.Artifacts.ContentType.DocumentTypeArtifact",
  "__version": "18.0.1"
}`

// statusAutomationUda is a synthetic automation shaped like the handover's
// redacted sample — deliberately not derived from a real artifact.
const statusAutomationUda = `{
  "Name": "Example Automation",
  "Alias": "exampleAutomation",
  "WorkspaceUdi": "umb://umbraco-automate-workspace/cccccccc111122223333444444444444",
  "Trigger": {"TriggerAlias": "example.trigger", "Settings": {}},
  "Steps": [
    {"Id": "s1", "ActionAlias": "umbracoAutomate.forEach", "Name": "Loop", "Alias": "loop", "ConnectionId": null, "Settings": {}, "InputMappings": {}},
    {"Id": "s2", "ActionAlias": "example.sendMail", "Name": "Send", "Alias": "send", "ConnectionId": null, "Settings": {}, "InputMappings": {}}
  ],
  "Connections": [{"SourceStepId": "s1", "TargetStepId": "s2"}],
  "Udi": "umb://umbraco-automate-automation/dddddddd111122223333444444444444",
  "Dependencies": [{"Udi": "umb://umbraco-automate-workspace/cccccccc111122223333444444444444"}],
  "__type": "Umbraco.Deploy.Automate,Umbraco.Deploy.Automate.Artifacts.AutomateAutomationArtifact",
  "__version": "18.0.1"
}`

const statusRemoteDataType = `{
  "id": "aaaaaaaa-1111-2222-3333-444444444444",
  "name": "Example Text",
  "editorAlias": "Umbraco.TextArea",
  "editorUiAlias": "Umb.PropertyEditorUi.TextArea",
  "values": [
    {"alias": "maxChars", "value": 500},
    {"alias": "umbMigrationV14", "value": "2025-10-07T11:11:35Z"}
  ]
}`

const statusRemoteDoctype = `{
  "id": "bbbbbbbb-1111-2222-3333-444444444444",
  "name": "Example Block",
  "alias": "exampleBlock",
  "icon": "icon-box",
  "isElement": true,
  "properties": [
    {"alias": "heading", "name": "Heading", "sortOrder": 0, "dataType": {"id": "aaaaaaaa-1111-2222-3333-444444444444"}},
    {"alias": "ungrouped", "name": "Ungrouped", "sortOrder": 5, "dataType": {"id": "aaaaaaaa-1111-2222-3333-444444444444"}}
  ]
}`

func deployStatusDeps(t *testing.T, remoteDataType string, remoteDoctype string, automateAvailable bool) Dependencies {
	t.Helper()
	return endpointDeps(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Path == "/umbraco/management/api/v1/security/back-office/token":
			return endpointJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		case req.URL.Path == "/umbraco/management/api/v1/data-type/aaaaaaaa-1111-2222-3333-444444444444":
			return endpointJSONResponse(http.StatusOK, remoteDataType), nil
		case req.URL.Path == "/umbraco/management/api/v1/document-type/bbbbbbbb-1111-2222-3333-444444444444":
			return endpointJSONResponse(http.StatusOK, remoteDoctype), nil
		case strings.HasPrefix(req.URL.Path, "/umbraco/automate/management/api/v1/"):
			if !automateAvailable {
				return endpointJSONResponse(http.StatusNotFound, `null`), nil
			}
			if strings.HasSuffix(req.URL.Path, "/automations") {
				return endpointJSONResponse(http.StatusOK, `{"items":[],"total":0}`), nil
			}
			return endpointJSONResponse(http.StatusOK, `{"name":"Example Automation","alias":"exampleAutomation"}`), nil
		default:
			return endpointJSONResponse(http.StatusNotFound, `null`), nil
		}
	})
}

func runDeployStatus(t *testing.T, deps Dependencies, dir string, extra ...string) map[string]any {
	t.Helper()
	args := append([]string{"deploy", "status", "--uda-dir", dir, "--exit-zero"}, extra...)
	output, err := execute(buildDeployRoot(deps), args...)
	if err != nil {
		t.Fatalf("deploy status failed: %v", err)
	}
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	return payload
}

func statusByFile(t *testing.T, payload map[string]any) map[string]map[string]any {
	t.Helper()
	byFile := map[string]map[string]any{}
	for _, item := range payload["artifacts"].([]any) {
		entry := item.(map[string]any)
		byFile[entry["file"].(string)] = entry
	}
	return byFile
}

func TestDeployStatusInSyncIgnoresEnvironmentOnlyValues(t *testing.T) {
	dir := t.TempDir()
	writeUda(t, dir, "data-type__a.uda", statusDataTypeUda)
	writeUda(t, dir, "document-type__b.uda", statusDoctypeUda)

	payload := runDeployStatus(t, deployStatusDeps(t, statusRemoteDataType, statusRemoteDoctype, false), dir)
	byFile := statusByFile(t, payload)
	if byFile["data-type__a.uda"]["status"] != "in-sync" {
		t.Fatalf("expected data type in-sync despite env-only umbMigrationV14 value, got %+v", byFile["data-type__a.uda"])
	}
	if byFile["document-type__b.uda"]["status"] != "in-sync" {
		t.Fatalf("expected doctype in-sync (both property collections read), got %+v", byFile["document-type__b.uda"])
	}
}

func TestDeployStatusDetectsConfigurationAndPropertyDrift(t *testing.T) {
	dir := t.TempDir()
	writeUda(t, dir, "data-type__a.uda", statusDataTypeUda)
	writeUda(t, dir, "document-type__b.uda", statusDoctypeUda)

	driftedDataType := strings.Replace(statusRemoteDataType, `"value": 500`, `"value": 200`, 1)
	driftedDoctype := strings.Replace(statusRemoteDoctype, `{"alias": "ungrouped", "name": "Ungrouped", "sortOrder": 5, "dataType": {"id": "aaaaaaaa-1111-2222-3333-444444444444"}}`, "", 1)
	driftedDoctype = strings.Replace(driftedDoctype, `}},`, `}}`, 1)
	payload := runDeployStatus(t, deployStatusDeps(t, driftedDataType, driftedDoctype, false), dir)
	byFile := statusByFile(t, payload)

	dataType := byFile["data-type__a.uda"]
	if dataType["status"] != "drifted" || !strings.Contains(jsonString(dataType["diffs"]), "configuration.maxChars") {
		t.Fatalf("expected configuration drift, got %+v", dataType)
	}
	doctype := byFile["document-type__b.uda"]
	if doctype["status"] != "drifted" || !strings.Contains(jsonString(doctype["diffs"]), "ungrouped (missing remotely)") {
		t.Fatalf("expected top-level PropertyTypes to be read and missing remotely, got %+v", doctype)
	}
}

func jsonString(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func TestDeployStatusMissingRemoteAndExitCode(t *testing.T) {
	dir := t.TempDir()
	missing := strings.Replace(statusDataTypeUda, "aaaaaaaa111122223333444444444444", "eeeeeeee111122223333444444444444", 1)
	writeUda(t, dir, "data-type__missing.uda", missing)

	deps := deployStatusDeps(t, statusRemoteDataType, statusRemoteDoctype, false)
	payload := runDeployStatus(t, deps, dir)
	byFile := statusByFile(t, payload)
	if byFile["data-type__missing.uda"]["status"] != "missing-remote" {
		t.Fatalf("expected missing-remote, got %+v", byFile["data-type__missing.uda"])
	}

	_, err := execute(buildDeployRoot(deps), "deploy", "status", "--uda-dir", dir)
	var drift deployDriftFoundError
	if err == nil || !strings.Contains(err.Error(), "missing") || drift.ExitCode() != 2 {
		t.Fatalf("expected exit-2 drift error without --exit-zero, got %v", err)
	}
}

func TestDeployStatusAutomateUnavailableIsUnknownNeverMissing(t *testing.T) {
	dir := t.TempDir()
	writeUda(t, dir, "umbraco-automate-automation__d.uda", statusAutomationUda)

	payload := runDeployStatus(t, deployStatusDeps(t, statusRemoteDataType, statusRemoteDoctype, false), dir, "--flag-step-alias", "umbracoAutomate.forEach")
	byFile := statusByFile(t, payload)
	automation := byFile["umbraco-automate-automation__d.uda"]
	if automation["status"] != "unknown" || !strings.Contains(automation["reason"].(string), "Automate API unavailable") {
		t.Fatalf("expected unknown (never missing-remote) without Automate, got %+v", automation)
	}
	if !strings.Contains(jsonString(automation["stepAliases"]), "umbracoAutomate.forEach") {
		t.Fatalf("expected step aliases read locally despite unavailable API, got %+v", automation)
	}
	if !strings.Contains(jsonString(automation["flags"]), "matches --flag-step-alias") {
		t.Fatalf("expected control-flow flag, got %+v", automation)
	}
	summary := payload["summary"].(map[string]any)
	if summary["flagged"].(float64) != 1 {
		t.Fatalf("expected flagged count 1, got %+v", summary)
	}
}

func TestDeployStatusAutomateAvailableComparesAndFinds(t *testing.T) {
	dir := t.TempDir()
	writeUda(t, dir, "umbraco-automate-automation__d.uda", statusAutomationUda)

	payload := runDeployStatus(t, deployStatusDeps(t, statusRemoteDataType, statusRemoteDoctype, true), dir)
	byFile := statusByFile(t, payload)
	if byFile["umbraco-automate-automation__d.uda"]["status"] != "in-sync" {
		t.Fatalf("expected in-sync automation with reachable Automate API, got %+v", byFile["umbraco-automate-automation__d.uda"])
	}
}

func TestDeployStatusParseFailureIsReportedNotDropped(t *testing.T) {
	dir := t.TempDir()
	writeUda(t, dir, "data-type__broken.uda", "{not json")

	payload := runDeployStatus(t, deployStatusDeps(t, statusRemoteDataType, statusRemoteDoctype, false), dir)
	byFile := statusByFile(t, payload)
	if byFile["data-type__broken.uda"]["status"] != "error" {
		t.Fatalf("expected parse error surfaced, got %+v", byFile["data-type__broken.uda"])
	}
}
