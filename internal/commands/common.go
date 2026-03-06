package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"umbraco-cli/internal/api"
	"umbraco-cli/internal/config"
	"umbraco-cli/internal/output"
)

type Dependencies struct {
	Client     *api.Client
	EnvOutput  config.OutputFormat
	OutputFlag *string
}

func (d Dependencies) requestedOutput() string {
	if d.OutputFlag == nil {
		return ""
	}
	return *d.OutputFlag
}

func printResult(cmd *cobra.Command, deps Dependencies, data any) error {
	return output.Print(data, deps.requestedOutput(), deps.EnvOutput, cmd.OutOrStdout())
}

func parseJSONObject(raw string, label string) (map[string]any, error) {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("invalid %s JSON: %w", label, err)
	}
	obj, ok := payload.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a JSON object", label)
	}
	return obj, nil
}

func parseParams(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	return parseJSONObject(raw, "--params")
}

func parsePayload(raw string) (map[string]any, error) {
	return parseJSONObject(raw, "--json")
}

func requireValue(name string, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("missing required option: %s", name)
	}
	return nil
}

func optionalBody(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	return parsePayload(raw)
}
