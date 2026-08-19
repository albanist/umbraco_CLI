package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"umbraco-cli/internal/api"
)

// deployDriftFoundError maps "the command ran cleanly and found drift" to
// exit code 2, mirroring schema diff's contract.
type deployDriftFoundError struct{ drifted, missing int }

func (e deployDriftFoundError) Error() string {
	return fmt.Sprintf("deploy status found %d drifted and %d missing artifacts", e.drifted, e.missing)
}
func (deployDriftFoundError) ExitCode() int { return 2 }

func deployStatus(deps Dependencies) *cobra.Command {
	var udaDir string
	var kinds []string
	var flagStepAliases []string
	var exitZero bool
	var concurrency int

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Compare local .uda deploy artifacts against the environment, read-only",
		Long: `Reads the Umbraco Deploy artifacts in --uda-dir (the site repo's umbraco/Deploy/Revision) and compares each against the target environment's database via the Management API, reporting in-sync vs drifted per entity. Strictly read-only: a pre-flight check that turns "will this deploy blow up or carry surprises?" into an answerable question — in-sync artifacts are skipped by Deploy's schema pass and are therefore safe; drifted ones are processed.

Comparison is per entity kind (data types, document/media/member types, templates, containers, member groups, relation types) over the fields the artifact carries; environment-only additions like migration markers are ignored. Automate artifacts degrade to status "unknown" where the Automate API is unreachable (Cloud basic auth blocks package APIs on non-live environments) — never a false in-sync — but their step aliases are still read locally, and --flag-step-alias marks automations carrying aliases you know your Deploy version cannot validate (configuration, not encoded knowledge: those landmines change as bugs are fixed).

Exit 2 when drift or missing entities are found (suppress with --exit-zero); parse failures and unreachable comparisons are reported per artifact, never silently dropped.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if concurrency < 1 {
				return fmt.Errorf("--concurrency must be at least 1")
			}
			artifacts, err := loadUdaArtifacts(udaDir, kinds)
			if err != nil {
				return err
			}
			if len(artifacts) == 0 {
				return fmt.Errorf("no .uda artifacts found in %s (pass --uda-dir pointing at the site repo's umbraco/Deploy/Revision)", udaDir)
			}

			results := compareArtifacts(cmd.Context(), deps, artifacts, flagStepAliases, concurrency)
			summary := map[string]int{}
			flagged := 0
			for _, result := range results {
				summary[result.Status]++
				if len(result.Flags) > 0 {
					flagged++
				}
			}
			payload := map[string]any{
				"udaDir":    udaDir,
				"artifacts": results,
				"summary": map[string]any{
					"total":         len(results),
					"inSync":        summary["in-sync"],
					"drifted":       summary["drifted"],
					"missingRemote": summary["missing-remote"],
					"unknown":       summary["unknown"],
					"errors":        summary["error"],
					"flagged":       flagged,
				},
			}
			if err := printResult(cmd, deps, payload); err != nil {
				return err
			}
			if !exitZero && (summary["drifted"] > 0 || summary["missing-remote"] > 0) {
				return deployDriftFoundError{drifted: summary["drifted"], missing: summary["missing-remote"]}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&udaDir, "uda-dir", filepath.Join("umbraco", "Deploy", "Revision"), "Directory holding the .uda artifacts")
	cmd.Flags().StringArrayVar(&kinds, "kind", nil, "Only compare these artifact kinds (Udi entity types, e.g. data-type, document-type; repeatable)")
	cmd.Flags().StringArrayVar(&flagStepAliases, "flag-step-alias", nil, "Flag automations whose steps carry this action-alias substring (repeatable; e.g. a control-flow alias your Deploy version fails to validate)")
	cmd.Flags().BoolVar(&exitZero, "exit-zero", false, "Exit 0 even when drift or missing entities are found")
	cmd.Flags().IntVar(&concurrency, "concurrency", 8, "Maximum concurrent environment lookups")
	return cmd
}

// udaArtifact is one parsed .uda file, discriminated by the Udi entity type
// (authoritative across every artifact kind, unlike the filename prefix or
// the assembly-qualified __type).
type udaArtifact struct {
	File string
	Kind string
	GUID string
	Body map[string]any
	Err  error
}

// udaStatusResult is one artifact's comparison outcome.
type udaStatusResult struct {
	File        string   `json:"file"`
	Kind        string   `json:"kind"`
	Udi         string   `json:"udi,omitempty"`
	Name        string   `json:"name,omitempty"`
	Status      string   `json:"status"`
	Diffs       []string `json:"diffs,omitempty"`
	Reason      string   `json:"reason,omitempty"`
	StepAliases []string `json:"stepAliases,omitempty"`
	Flags       []string `json:"flags,omitempty"`
}

func loadUdaArtifacts(dir string, kinds []string) ([]udaArtifact, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read --uda-dir %s: %w", dir, err)
	}
	wanted := map[string]struct{}{}
	for _, kind := range kinds {
		wanted[strings.TrimSpace(kind)] = struct{}{}
	}

	artifacts := make([]udaArtifact, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".uda") {
			continue
		}
		artifact := parseUdaFile(filepath.Join(dir, entry.Name()))
		if len(wanted) > 0 {
			if _, ok := wanted[artifact.Kind]; !ok && artifact.Err == nil {
				continue
			}
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

func parseUdaFile(path string) udaArtifact {
	artifact := udaArtifact{File: filepath.Base(path)}
	raw, err := os.ReadFile(path)
	if err != nil {
		artifact.Err = err
		return artifact
	}
	// Deploy writes the files UTF-8 with BOM; json.Unmarshal rejects it.
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	body := map[string]any{}
	if err := json.Unmarshal(raw, &body); err != nil {
		artifact.Err = fmt.Errorf("parse: %w", err)
		return artifact
	}
	artifact.Body = body
	udi, _ := body["Udi"].(string)
	artifact.Kind, artifact.GUID = parseUdi(udi)
	if artifact.Kind == "" {
		artifact.Err = fmt.Errorf("no usable Udi in artifact")
	}
	return artifact
}

// parseUdi splits umb://<entity-type>/<guid-without-dashes> into the entity
// type and a dashed GUID the Management API accepts.
func parseUdi(udi string) (string, string) {
	rest, ok := strings.CutPrefix(udi, "umb://")
	if !ok {
		return "", ""
	}
	kind, hex, ok := strings.Cut(rest, "/")
	if !ok || len(hex) != 32 {
		return kind, ""
	}
	return kind, strings.Join([]string{hex[0:8], hex[8:12], hex[12:16], hex[16:20], hex[20:32]}, "-")
}

func compareArtifacts(ctx context.Context, deps Dependencies, artifacts []udaArtifact, flagStepAliases []string, concurrency int) []udaStatusResult {
	// Probe Automate availability once up front: on an environment without
	// the package (or with the API blocked by Cloud basic auth) every
	// per-entity lookup 404s, which must read as "unknown — API
	// unavailable", never as the entity missing remotely.
	var automateErr error
	automateProbed := false
	for _, artifact := range artifacts {
		if strings.HasPrefix(artifact.Kind, "umbraco-automate-") && artifact.Err == nil {
			_, automateErr = deps.Client.Get(ctx, "/automations", api.RequestOptions{APIPrefix: automateAPIPrefix, Params: map[string]any{"skip": 0, "take": 1}})
			automateProbed = true
			break
		}
	}
	_ = automateProbed

	results := make([]udaStatusResult, len(artifacts))
	var wg sync.WaitGroup
	slots := make(chan struct{}, concurrency)
	for i := range artifacts {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			results[index] = compareArtifact(ctx, deps, artifacts[index], flagStepAliases, automateErr)
		}(i)
	}
	wg.Wait()

	statusRank := map[string]int{"error": 0, "drifted": 1, "missing-remote": 2, "unknown": 3, "in-sync": 4}
	sort.SliceStable(results, func(i, j int) bool {
		if statusRank[results[i].Status] != statusRank[results[j].Status] {
			return statusRank[results[i].Status] < statusRank[results[j].Status]
		}
		return results[i].File < results[j].File
	})
	return results
}

func compareArtifact(ctx context.Context, deps Dependencies, artifact udaArtifact, flagStepAliases []string, automateErr error) udaStatusResult {
	result := udaStatusResult{File: artifact.File, Kind: artifact.Kind}
	if artifact.Err != nil {
		result.Status = "error"
		result.Reason = artifact.Err.Error()
		return result
	}
	result.Udi, _ = artifact.Body["Udi"].(string)
	result.Name, _ = artifact.Body["Name"].(string)

	if strings.HasPrefix(artifact.Kind, "umbraco-automate-") {
		return compareAutomateArtifact(ctx, deps, artifact, result, flagStepAliases, automateErr)
	}

	fetchPath, comparer := udaComparer(artifact.Kind)
	if comparer == nil {
		result.Status = "unknown"
		result.Reason = fmt.Sprintf("no comparison implemented for kind %s", artifact.Kind)
		return result
	}

	remote, err := fetchObject(ctx, deps.Client, api.JoinPath(fetchPath, artifact.GUID), api.RequestOptions{})
	if err != nil {
		if isAPIStatus(err, http.StatusNotFound) {
			result.Status = "missing-remote"
			result.Reason = "entity does not exist on the target environment"
			return result
		}
		// Never report a false in-sync when the comparison could not run.
		result.Status = "unknown"
		result.Reason = err.Error()
		return result
	}

	diffs := comparer(artifact.Body, remote)
	if len(diffs) > 0 {
		sort.Strings(diffs)
		result.Status = "drifted"
		result.Diffs = diffs
		return result
	}
	result.Status = "in-sync"
	return result
}

// compareAutomateArtifact handles the Automate package artifacts. The
// Automate API is blocked by Cloud basic auth on non-live environments and
// is absent entirely without the package, so unreachable comparisons
// degrade to "unknown" — but step aliases are always read locally, and
// --flag-step-alias marks automations regardless of API reachability.
func compareAutomateArtifact(ctx context.Context, deps Dependencies, artifact udaArtifact, result udaStatusResult, flagStepAliases []string, automateErr error) udaStatusResult {
	if artifact.Kind == "umbraco-automate-automation" {
		result.StepAliases = automationStepAliases(artifact.Body)
		for _, needle := range flagStepAliases {
			needle = strings.TrimSpace(needle)
			if needle == "" {
				continue
			}
			for _, alias := range result.StepAliases {
				if strings.Contains(alias, needle) {
					result.Flags = append(result.Flags, fmt.Sprintf("step alias %q matches --flag-step-alias %q", alias, needle))
				}
			}
		}
	}

	if automateErr != nil {
		result.Status = "unknown"
		result.Reason = "Automate API unavailable on this environment: " + automateErr.Error()
		return result
	}
	fetchPath := map[string]string{
		"umbraco-automate-automation": "/automations/%s",
		"umbraco-automate-workspace":  "/workspaces/%s",
		"umbraco-automate-connection": "/connections/%s",
	}[artifact.Kind]
	if fetchPath == "" {
		result.Status = "unknown"
		result.Reason = fmt.Sprintf("no comparison implemented for kind %s", artifact.Kind)
		return result
	}
	remote, err := deps.Client.Get(ctx, api.JoinPath(fetchPath, artifact.GUID), api.RequestOptions{APIPrefix: automateAPIPrefix})
	if err != nil {
		if isAPIStatus(err, http.StatusNotFound) {
			result.Status = "missing-remote"
			result.Reason = "entity does not exist on the target environment"
			return result
		}
		result.Status = "unknown"
		result.Reason = "Automate API unreachable: " + err.Error()
		return result
	}
	remoteObject, _ := remote.(map[string]any)
	diffs := diffFields(nil,
		fieldDiff("name", artifact.Body["Name"], remoteObject["name"]),
		fieldDiff("alias", artifact.Body["Alias"], remoteObject["alias"]),
	)
	if len(diffs) > 0 {
		result.Status = "drifted"
		result.Diffs = diffs
		return result
	}
	result.Status = "in-sync"
	return result
}

func automationStepAliases(body map[string]any) []string {
	steps, _ := body["Steps"].([]any)
	seen := map[string]struct{}{}
	for _, item := range steps {
		step, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if alias, _ := step["ActionAlias"].(string); alias != "" {
			seen[alias] = struct{}{}
		}
	}
	return sortedKeys(seen)
}

// udaComparer maps a Udi entity type to its Management API fetch path and a
// field comparer. Comparisons cover the fields the artifact carries;
// environment-only additions (e.g. migration markers in data-type values)
// are not drift.
func udaComparer(kind string) (string, func(artifact map[string]any, remote map[string]any) []string) {
	switch kind {
	case "data-type":
		return "/data-type/%s", compareDataTypeArtifact
	case "document-type":
		return "/document-type/%s", compareContentTypeArtifact
	case "media-type":
		return "/media-type/%s", compareContentTypeArtifact
	case "member-type":
		return "/member-type/%s", compareContentTypeArtifact
	case "template":
		return "/template/%s", compareTemplateArtifact
	case "document-type-container":
		return "/document-type/folder/%s", compareNameOnlyArtifact
	case "data-type-container":
		return "/data-type/folder/%s", compareNameOnlyArtifact
	case "media-type-container":
		return "/media-type/folder/%s", compareNameOnlyArtifact
	case "member-group":
		return "/member-group/%s", compareNameOnlyArtifact
	case "relation-type":
		return "/relation-type/%s", compareNameOnlyArtifact
	}
	return "", nil
}

func compareNameOnlyArtifact(artifact map[string]any, remote map[string]any) []string {
	return diffFields(nil, fieldDiff("name", artifact["Name"], remote["name"]))
}

func compareDataTypeArtifact(artifact map[string]any, remote map[string]any) []string {
	diffs := diffFields(nil,
		fieldDiff("name", artifact["Name"], remote["name"]),
		fieldDiff("editorAlias", artifact["EditorAlias"], remote["editorAlias"]),
		fieldDiff("editorUiAlias", artifact["EditorUiAlias"], remote["editorUiAlias"]),
	)
	configuration, _ := artifact["Configuration"].(map[string]any)
	remoteValues := map[string]any{}
	if values, ok := remote["values"].([]any); ok {
		for _, item := range values {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if alias, _ := entry["alias"].(string); alias != "" {
				remoteValues[alias] = entry["value"]
			}
		}
	}
	for key, artifactValue := range configuration {
		remoteValue, exists := remoteValues[key]
		if !exists || !jsonValueEqual(artifactValue, remoteValue) {
			diffs = append(diffs, "configuration."+key)
		}
	}
	return diffs
}

func compareTemplateArtifact(artifact map[string]any, remote map[string]any) []string {
	diffs := diffFields(nil,
		fieldDiff("name", artifact["Name"], remote["name"]),
		fieldDiff("alias", artifact["Alias"], remote["alias"]),
	)
	if artifactContent, ok := artifact["Content"].(string); ok {
		remoteContent, _ := remote["content"].(string)
		if normalizeTemplateContent(artifactContent) != normalizeTemplateContent(remoteContent) {
			diffs = append(diffs, "content")
		}
	}
	return diffs
}

func normalizeTemplateContent(content string) string {
	return strings.TrimSpace(strings.ReplaceAll(content, "\r\n", "\n"))
}

func compareContentTypeArtifact(artifact map[string]any, remote map[string]any) []string {
	diffs := diffFields(nil,
		fieldDiff("name", artifact["Name"], remote["name"]),
		fieldDiff("alias", artifact["Alias"], remote["alias"]),
		fieldDiff("icon", artifact["Icon"], remote["icon"]),
	)
	if permissions, ok := artifact["Permissions"].(map[string]any); ok {
		if isElement, ok := permissions["IsElementType"].(bool); ok {
			remoteElement, _ := remote["isElement"].(bool)
			if isElement != remoteElement {
				diffs = append(diffs, "isElement")
			}
		}
	}

	artifactProperties := artifactPropertyIndex(artifact)
	remoteProperties := map[string]map[string]any{}
	if properties, ok := remote["properties"].([]any); ok {
		for _, item := range properties {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if alias, _ := entry["alias"].(string); alias != "" {
				remoteProperties[alias] = entry
			}
		}
	}
	for alias, artifactProperty := range artifactProperties {
		remoteProperty, exists := remoteProperties[alias]
		if !exists {
			diffs = append(diffs, "property "+alias+" (missing remotely)")
			continue
		}
		if name, _ := artifactProperty["Name"].(string); name != udaStringField(remoteProperty, "name") {
			diffs = append(diffs, "property "+alias+".name")
		}
		if _, dataTypeGUID := parseUdi(udaStringField(artifactProperty, "DataType")); dataTypeGUID != "" {
			remoteDataType := ""
			if reference, ok := remoteProperty["dataType"].(map[string]any); ok {
				remoteDataType, _ = reference["id"].(string)
			}
			if !strings.EqualFold(dataTypeGUID, remoteDataType) {
				diffs = append(diffs, "property "+alias+".dataType")
			}
		}
		if sortOrder, ok := artifactProperty["SortOrder"].(float64); ok {
			if remoteSort, ok := remoteProperty["sortOrder"].(float64); ok && sortOrder != remoteSort {
				diffs = append(diffs, "property "+alias+".sortOrder")
			}
		}
	}
	for alias := range remoteProperties {
		if _, exists := artifactProperties[alias]; !exists {
			diffs = append(diffs, "property "+alias+" (missing in artifact)")
		}
	}
	return diffs
}

// artifactPropertyIndex reads BOTH property collections a content-type
// artifact carries: PropertyGroups[].PropertyTypes (grouped, the normal
// case) and the top-level PropertyTypes[] (ungrouped). Reading only the
// grouped collection silently ignores ungrouped properties.
func artifactPropertyIndex(artifact map[string]any) map[string]map[string]any {
	index := map[string]map[string]any{}
	collect := func(items []any) {
		for _, item := range items {
			property, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if alias, _ := property["Alias"].(string); alias != "" {
				index[alias] = property
			}
		}
	}
	if groups, ok := artifact["PropertyGroups"].([]any); ok {
		for _, item := range groups {
			group, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if properties, ok := group["PropertyTypes"].([]any); ok {
				collect(properties)
			}
		}
	}
	if properties, ok := artifact["PropertyTypes"].([]any); ok {
		collect(properties)
	}
	return index
}

func udaStringField(object map[string]any, key string) string {
	value, _ := object[key].(string)
	return value
}

func fieldDiff(field string, artifactValue any, remoteValue any) string {
	if artifactValue == nil {
		return ""
	}
	if jsonValueEqual(artifactValue, remoteValue) {
		return ""
	}
	return field
}

func diffFields(diffs []string, candidates ...string) []string {
	for _, candidate := range candidates {
		if candidate != "" {
			diffs = append(diffs, candidate)
		}
	}
	return diffs
}

func jsonValueEqual(a any, b any) bool {
	return reflect.DeepEqual(a, b)
}
