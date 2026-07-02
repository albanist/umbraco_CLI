package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteUserConfigWithOptionsTargetsProfileFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := Config{BaseURL: "https://dev.example.test", ClientID: "id", ClientSecret: "secret", OutputFormat: OutputJSON}
	if err := WriteUserConfigWithOptions(LoadOptions{Profile: "dev"}, cfg); err != nil {
		t.Fatalf("WriteUserConfigWithOptions failed: %v", err)
	}

	path := filepath.Join(home, ".umbraco", "dev.config.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected profile config written: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config permissions = %v, want 0600", info.Mode().Perm())
	}

	loaded, ok, _, err := LoadUserConfigWithOptions(LoadOptions{Profile: "dev"})
	if err != nil || !ok {
		t.Fatalf("expected written profile to load, ok=%v err=%v", ok, err)
	}
	if loaded.BaseURL != "https://dev.example.test" || loaded.ClientID != "id" {
		t.Fatalf("unexpected loaded config %+v", loaded)
	}
}

func TestWriteUserConfigTightensExistingPermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".umbraco")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteUserConfig(Config{BaseURL: "https://example.test"}); err != nil {
		t.Fatalf("WriteUserConfig failed: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("pre-existing config not tightened to 0600, got %v", info.Mode().Perm())
	}
}

func TestClearUserAuthWithOptionsPreservesNonCredentialFields(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := Config{BaseURL: "https://dev.example.test", ClientID: "id", ClientSecret: "secret", OutputFormat: OutputTable}
	if err := WriteUserConfigWithOptions(LoadOptions{Profile: "dev"}, cfg); err != nil {
		t.Fatal(err)
	}

	if err := ClearUserAuthWithOptions(LoadOptions{Profile: "dev"}); err != nil {
		t.Fatalf("ClearUserAuthWithOptions failed: %v", err)
	}

	loaded, ok, _, err := LoadUserConfigWithOptions(LoadOptions{Profile: "dev"})
	if err != nil || !ok {
		t.Fatalf("expected cleared profile to load, ok=%v err=%v", ok, err)
	}
	if loaded.ClientID != "" || loaded.ClientSecret != "" {
		t.Fatalf("expected credentials cleared, got %+v", loaded)
	}
	if loaded.BaseURL != "https://dev.example.test" || loaded.OutputFormat != OutputTable {
		t.Fatalf("expected non-credential fields preserved, got %+v", loaded)
	}

	payload, err := os.ReadFile(filepath.Join(home, ".umbraco", "dev.config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "secret") {
		t.Fatalf("secret still present on disk: %s", payload)
	}
}

func TestClearUserAuthWithOptionsNoopWhenConfigMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := ClearUserAuthWithOptions(LoadOptions{}); err != nil {
		t.Fatalf("expected clearing a missing config to be a no-op, got %v", err)
	}
}
