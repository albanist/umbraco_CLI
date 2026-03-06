package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/spf13/cobra"

	"umbraco-cli/internal/api"
	"umbraco-cli/internal/config"
)

func makeDeps() Dependencies {
	cfg := config.Config{BaseURL: "https://example.test"}
	client := api.NewClient(cfg, http.DefaultClient, nil)
	output := "json"
	return Dependencies{Client: client, EnvOutput: config.OutputJSON, OutputFlag: &output}
}

func buildRootWithCollections(t *testing.T, deps Dependencies) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "umbraco", SilenceErrors: true, SilenceUsage: true}
	root.SetErr(io.Discard)
	RegisterDocument(root, deps)
	RegisterDictionary(root, deps)
	RegisterMedia(root, deps)
	RegisterDoctype(root, deps)
	RegisterDatatype(root, deps)
	RegisterTemplate(root, deps)
	RegisterLogs(root, deps)
	RegisterServer(root, deps)
	RegisterHealth(root, deps)
	RegisterSchema(root, deps)
	return root
}

func execute(root *cobra.Command, args ...string) (string, error) {
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(io.Discard)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestCommandCountsMatchMVP(t *testing.T) {
	deps := makeDeps()
	root := buildRootWithCollections(t, deps)

	total := 0
	for collection, expected := range ExpectedCollectionCommandCounts {
		var found *cobra.Command
		for _, command := range root.Commands() {
			if command.Name() == collection {
				found = command
				break
			}
		}
		if found == nil {
			t.Fatalf("missing collection %s", collection)
		}
		if len(found.Commands()) != expected {
			t.Fatalf("collection %s expected %d commands, got %d", collection, expected, len(found.Commands()))
		}
		total += len(found.Commands())
	}

	if total != 69 {
		t.Fatalf("expected 69 collection commands, got %d", total)
	}
}

func TestSchemaCommandListAndCollectionLookup(t *testing.T) {
	deps := makeDeps()
	output, err := execute(buildRootWithCollections(t, deps), "schema", "--list")
	if err != nil {
		t.Fatalf("schema --list failed: %v", err)
	}
	var listPayload map[string]any
	if err := json.Unmarshal([]byte(output), &listPayload); err != nil {
		t.Fatalf("failed to parse list payload: %v", err)
	}
	endpoints, ok := listPayload["endpoints"].([]any)
	if !ok || len(endpoints) == 0 {
		t.Fatalf("expected non-empty endpoints list")
	}

	output, err = execute(buildRootWithCollections(t, deps), "schema", "document")
	if err != nil {
		t.Fatalf("schema collection lookup failed: %v", err)
	}
	var collectionPayload map[string]any
	if err := json.Unmarshal([]byte(output), &collectionPayload); err != nil {
		t.Fatalf("failed to parse collection payload: %v", err)
	}
	if collectionPayload["collection"] != "document" {
		t.Fatalf("unexpected collection payload: %+v", collectionPayload)
	}
}

func TestDocumentPublishPrefersJSONOverCultureInDryRun(t *testing.T) {
	deps := makeDeps()
	root := buildRootWithCollections(t, deps)

	output, err := execute(root,
		"document", "publish", "abc-123",
		"--json", `{"cultures":["da-DK"]}`,
		"--culture", "en-US",
		"--dry-run",
	)
	if err != nil {
		t.Fatalf("document publish failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse dry-run payload: %v", err)
	}
	body, ok := payload["body"].(map[string]any)
	if !ok {
		t.Fatalf("missing body in dry-run payload: %+v", payload)
	}
	cultures, ok := body["cultures"].([]any)
	if !ok || len(cultures) != 1 || cultures[0] != "da-DK" {
		t.Fatalf("expected --json cultures to take precedence, got: %+v", body)
	}
}
