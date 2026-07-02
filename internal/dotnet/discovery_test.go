package dotnet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testNormalize mirrors the caller-owned normalization contract: trim
// trailing slashes so duplicate candidates collapse.
func testNormalize(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

func writeFixture(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

const hostCsproj = `<Project Sdk="Microsoft.NET.Sdk.Web">
  <ItemGroup><PackageReference Include="Umbraco.Cms" Version="16.0.0" /></ItemGroup>
</Project>`

func TestDiscoverBaseURLFromLaunchSettings(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "Properties", "launchSettings.json"), `{
		"profiles": {
			"Web": {"applicationUrl": "https://localhost:44314;http://localhost:5000"}
		}
	}`)

	url, ok := DiscoverBaseURL(root, testNormalize)
	if !ok || url != "https://localhost:44314" {
		t.Fatalf("expected https launch URL preferred, got %q ok=%v", url, ok)
	}
}

func TestDiscoverBaseURLFromKestrelAppSettings(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "appsettings.json"), `{
		"Kestrel": {"Endpoints": {"Https": {"Url": "https://localhost:44391/"}}}
	}`)

	url, ok := DiscoverBaseURL(root, testNormalize)
	if !ok || url != "https://localhost:44391" {
		t.Fatalf("expected normalized Kestrel URL, got %q ok=%v", url, ok)
	}
}

func TestDiscoverBaseURLRefusesAmbiguousCandidates(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "Properties", "launchSettings.json"), `{
		"profiles": {
			"A": {"applicationUrl": "https://localhost:44314"},
			"B": {"applicationUrl": "https://localhost:44391"}
		}
	}`)

	if url, ok := DiscoverBaseURL(root, testNormalize); ok {
		t.Fatalf("expected no guess between two https candidates, got %q", url)
	}
}

func TestDiscoverBaseURLIgnoresMalformedFiles(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "Properties", "launchSettings.json"), `{not-json`)
	writeFixture(t, filepath.Join(root, "appsettings.json"), `also broken`)

	if url, ok := DiscoverBaseURL(root, testNormalize); ok {
		t.Fatalf("expected malformed files treated as absent, got %q", url)
	}
}

func TestDiscoverBaseURLFindsSingleSiblingHostProject(t *testing.T) {
	parent := t.TempDir()
	workingDir := filepath.Join(parent, "cli-workdir")
	host := filepath.Join(parent, "MySite")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(host, "MySite.csproj"), hostCsproj)
	writeFixture(t, filepath.Join(host, "Properties", "launchSettings.json"), `{
		"profiles": {"Web": {"applicationUrl": "https://localhost:44314"}}
	}`)

	url, ok := DiscoverBaseURL(workingDir, testNormalize)
	if !ok || url != "https://localhost:44314" {
		t.Fatalf("expected sibling host URL, got %q ok=%v", url, ok)
	}
}

func TestDiscoverBaseURLRefusesAmbiguousSiblingHosts(t *testing.T) {
	parent := t.TempDir()
	workingDir := filepath.Join(parent, "cli-workdir")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"SiteA", "SiteB"} {
		host := filepath.Join(parent, name)
		writeFixture(t, filepath.Join(host, name+".csproj"), hostCsproj)
		writeFixture(t, filepath.Join(host, "Properties", "launchSettings.json"),
			`{"profiles": {"Web": {"applicationUrl": "https://localhost:4431`+string(rune('0'+i))+`"}}}`)
	}

	if url, ok := DiscoverBaseURL(workingDir, testNormalize); ok {
		t.Fatalf("expected no guess between two sibling hosts, got %q", url)
	}
}

func TestDiscoverBaseURLIgnoresNonHostSiblings(t *testing.T) {
	parent := t.TempDir()
	workingDir := filepath.Join(parent, "cli-workdir")
	library := filepath.Join(parent, "SomeLibrary")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A class library referencing Umbraco but not a web host must not count.
	writeFixture(t, filepath.Join(library, "SomeLibrary.csproj"), `<Project Sdk="Microsoft.NET.Sdk">
  <ItemGroup><PackageReference Include="Umbraco.Cms.Core" Version="16.0.0" /></ItemGroup>
</Project>`)
	writeFixture(t, filepath.Join(library, "appsettings.json"), `{
		"Kestrel": {"Endpoints": {"Https": {"Url": "https://localhost:44391"}}}
	}`)

	if url, ok := DiscoverBaseURL(workingDir, testNormalize); ok {
		t.Fatalf("expected non-host sibling ignored, got %q", url)
	}
}

func TestDiscoverBaseURLEmptyWorkingDir(t *testing.T) {
	if url, ok := DiscoverBaseURL("", testNormalize); ok {
		t.Fatalf("expected no discovery without a working dir, got %q", url)
	}
}

func TestSplitURLCandidates(t *testing.T) {
	got := splitURLCandidates("https://localhost:44314; http://localhost:5000 ,ftp://nope, ")
	want := []string{"https://localhost:44314", "http://localhost:5000"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("splitURLCandidates = %v, want %v", got, want)
	}
}

func TestChoosePreferredURLPrefersSingleHTTPS(t *testing.T) {
	url, ok := choosePreferredURL([]string{"http://localhost:5000", "https://localhost:44314", "http://localhost:5001"})
	if !ok || url != "https://localhost:44314" {
		t.Fatalf("expected single https winner, got %q ok=%v", url, ok)
	}

	if _, ok := choosePreferredURL(nil); ok {
		t.Fatalf("expected no choice from empty candidates")
	}
	if _, ok := choosePreferredURL([]string{"https://a", "https://b"}); ok {
		t.Fatalf("expected no choice between two https candidates")
	}
	url, ok = choosePreferredURL([]string{"http://only", "http://only"})
	if !ok || url != "http://only" {
		t.Fatalf("expected deduped single candidate, got %q ok=%v", url, ok)
	}
}
