// Package dotnet discovers the base URL of a local Umbraco host project by
// scanning .NET configuration files (Properties/launchSettings.json and
// appsettings*.json) in the working tree and sibling projects.
package dotnet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DiscoverBaseURL walks the working tree upward and then sibling project
// directories for a single unambiguous Umbraco host URL. normalize is applied
// to every candidate before deduplication (the caller owns URL normalization
// rules, e.g. trimming a trailing /umbraco). Discovery is best-effort: an
// unreadable or malformed file must never make the caller unusable.
func DiscoverBaseURL(workingDir string, normalize func(string) string) (string, bool) {
	if chosen, ok := discoverBaseURLFromSearchRoots(searchRootsForCurrentTree(workingDir), normalize); ok {
		return chosen, true
	}

	if chosen, ok := discoverBaseURLFromSearchRoots(searchRootsForSiblingProjects(workingDir), normalize); ok {
		return chosen, true
	}

	return "", false
}

func searchRootsForCurrentTree(workingDir string) []string {
	roots := make([]string, 0)
	if strings.TrimSpace(workingDir) == "" {
		return roots
	}

	dir := workingDir
	for {
		roots = append(roots, dir)

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return roots
}

func searchRootsForSiblingProjects(workingDir string) []string {
	if strings.TrimSpace(workingDir) == "" {
		return nil
	}

	parent := filepath.Dir(workingDir)
	if parent == workingDir || strings.TrimSpace(parent) == "" {
		return nil
	}

	entries, err := os.ReadDir(parent)
	if err != nil {
		return nil
	}

	currentBase := filepath.Clean(workingDir)
	roots := make([]string, 0)
	seen := map[string]struct{}{}

	appendRoot := func(path string) {
		cleaned := filepath.Clean(path)
		if cleaned == currentBase {
			return
		}
		if !isLikelyUmbracoHostProject(cleaned) {
			return
		}
		if _, exists := seen[cleaned]; exists {
			return
		}
		seen[cleaned] = struct{}{}
		roots = append(roots, cleaned)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		siblingRoot := filepath.Join(parent, entry.Name())
		appendRoot(siblingRoot)

		nestedEntries, err := os.ReadDir(siblingRoot)
		if err != nil {
			continue
		}
		for _, nested := range nestedEntries {
			if nested.IsDir() {
				appendRoot(filepath.Join(siblingRoot, nested.Name()))
			}
		}
	}

	return roots
}

func isLikelyUmbracoHostProject(root string) bool {
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}

	hostProjects := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".csproj" {
			continue
		}

		payload, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			continue
		}

		contents := string(payload)
		if strings.Contains(contents, "Microsoft.NET.Sdk.Web") && strings.Contains(contents, "Umbraco.Cms") {
			hostProjects++
		}
	}

	return hostProjects == 1
}

func discoverBaseURLFromSearchRoots(roots []string, normalize func(string) string) (string, bool) {
	urls := make([]string, 0)

	for _, root := range roots {
		urls = append(urls, collectBaseURLsFromRoot(root, normalize)...)
	}

	return choosePreferredURL(urls)
}

func collectBaseURLsFromRoot(root string, normalize func(string) string) []string {
	if strings.TrimSpace(root) == "" {
		return nil
	}

	urls := make([]string, 0)
	for _, candidate := range []string{
		filepath.Join(root, "Properties", "launchSettings.json"),
		filepath.Join(root, "appsettings.Development.json"),
		filepath.Join(root, "appsettings.json"),
	} {
		urls = append(urls, collectJSONConfigURLs(candidate, normalize)...)
	}

	return urls
}

func collectJSONConfigURLs(path string, normalize func(string) string) []string {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var urls []string
	switch filepath.Base(path) {
	case "launchSettings.json":
		urls, err = collectLaunchSettingsURLs(payload, normalize)
	case "appsettings.Development.json", "appsettings.json":
		urls, err = collectAppSettingsURLs(payload, normalize)
	}
	if err != nil {
		return nil
	}
	return urls
}

func collectLaunchSettingsURLs(payload []byte, normalize func(string) string) ([]string, error) {
	var decoded struct {
		Profiles map[string]struct {
			ApplicationURL string `json:"applicationUrl"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}

	results := make([]string, 0)
	for _, profile := range decoded.Profiles {
		for _, candidate := range splitURLCandidates(profile.ApplicationURL) {
			if normalized := normalize(candidate); normalized != "" {
				results = append(results, normalized)
			}
		}
	}
	return results, nil
}

func collectAppSettingsURLs(payload []byte, normalize func(string) string) ([]string, error) {
	var decoded struct {
		Kestrel struct {
			Endpoints map[string]struct {
				URL string `json:"Url"`
			} `json:"Endpoints"`
		} `json:"Kestrel"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}

	results := make([]string, 0)
	for _, endpoint := range decoded.Kestrel.Endpoints {
		for _, candidate := range splitURLCandidates(endpoint.URL) {
			if normalized := normalize(candidate); normalized != "" {
				results = append(results, normalized)
			}
		}
	}
	return results, nil
}

func splitURLCandidates(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ';' || r == ','
	})

	results := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
			results = append(results, trimmed)
		}
	}
	return results
}

func choosePreferredURL(candidates []string) (string, bool) {
	if len(candidates) == 0 {
		return "", false
	}

	unique := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		unique = append(unique, candidate)
	}

	if len(unique) == 1 {
		return unique[0], true
	}

	httpsOnly := make([]string, 0, len(unique))
	for _, candidate := range unique {
		if strings.HasPrefix(strings.ToLower(candidate), "https://") {
			httpsOnly = append(httpsOnly, candidate)
		}
	}
	if len(httpsOnly) == 1 {
		return httpsOnly[0], true
	}

	return "", false
}
