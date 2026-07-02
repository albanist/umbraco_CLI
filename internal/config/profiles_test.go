package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeProfileName(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "", want: ""},
		{in: "  ", want: ""},
		{in: "default", want: "default"},
		{in: "dev", want: "dev"},
		{in: "Dev-2.local_x", want: "Dev-2.local_x"},
		{in: "../escape", wantErr: true},
		{in: "has space", wantErr: true},
		{in: "sla/sh", wantErr: true},
	}
	for _, tc := range cases {
		got, err := normalizeProfileName(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("normalizeProfileName(%q): expected error", tc.in)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Fatalf("normalizeProfileName(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
}

func TestProfileConfigPathResolution(t *testing.T) {
	path, err := profileConfigPath("/home/user", "dev")
	if err != nil || path != filepath.Join("/home/user", ".umbraco", "dev.config.json") {
		t.Fatalf("unexpected profile path %q, %v", path, err)
	}

	path, err = profileConfigPath("/home/user", "default")
	if err != nil || path != filepath.Join("/home/user", ".umbraco", "config.json") {
		t.Fatalf("default profile must map to config.json, got %q, %v", path, err)
	}

	if _, err := profileConfigPath("", "dev"); err == nil {
		t.Fatalf("expected error without home directory")
	}
	if _, err := profileConfigPath("/home/user", "../escape"); err == nil {
		t.Fatalf("expected error for invalid profile name")
	}
}

func TestExpandConfigPath(t *testing.T) {
	home := t.TempDir()
	work := t.TempDir()

	path, err := expandConfigPath("~/custom.json", work, home)
	if err != nil || path != filepath.Join(home, "custom.json") {
		t.Fatalf("expected tilde expansion, got %q, %v", path, err)
	}

	path, err = expandConfigPath("nested/config.json", work, home)
	if err != nil || path != filepath.Join(work, "nested", "config.json") {
		t.Fatalf("expected relative resolution against working dir, got %q, %v", path, err)
	}

	abs := filepath.Join(work, "abs.json")
	path, err = expandConfigPath(abs, work, home)
	if err != nil || path != abs {
		t.Fatalf("expected absolute path passthrough, got %q, %v", path, err)
	}

	if _, err := expandConfigPath("  ", work, home); err == nil {
		t.Fatalf("expected error for empty path")
	}
	if _, err := expandConfigPath("~/x.json", work, ""); err == nil {
		t.Fatalf("expected error expanding tilde without home directory")
	}
}

func TestActiveProfileRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if _, ok, err := ActiveProfile(); err != nil || ok {
		t.Fatalf("expected no active profile initially, got ok=%v err=%v", ok, err)
	}

	if err := SetActiveProfile("dev"); err != nil {
		t.Fatalf("SetActiveProfile failed: %v", err)
	}
	active, ok, err := ActiveProfile()
	if err != nil || !ok || active != "dev" {
		t.Fatalf("expected active profile dev, got %q ok=%v err=%v", active, ok, err)
	}

	info, err := os.Stat(filepath.Join(home, ".umbraco", "active-profile"))
	if err != nil {
		t.Fatalf("active-profile file missing: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("active-profile permissions = %v, want 0600", info.Mode().Perm())
	}

	// Selecting the default profile clears the marker.
	if err := SetActiveProfile("default"); err != nil {
		t.Fatalf("SetActiveProfile(default) failed: %v", err)
	}
	if _, ok, _ := ActiveProfile(); ok {
		t.Fatalf("expected no active profile after resetting to default")
	}

	if err := SetActiveProfile("../escape"); err == nil {
		t.Fatalf("expected error for invalid profile name")
	}
}

func TestReadActiveProfileToleratesWhitespaceAndMissingHome(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".umbraco"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".umbraco", "active-profile"), []byte("  dev \n"), 0o600); err != nil {
		t.Fatal(err)
	}

	profile, ok, err := readActiveProfile(home)
	if err != nil || !ok || profile != "dev" {
		t.Fatalf("expected trimmed profile dev, got %q ok=%v err=%v", profile, ok, err)
	}

	if _, ok, err := readActiveProfile(""); err != nil || ok {
		t.Fatalf("expected no profile without home dir, got ok=%v err=%v", ok, err)
	}
}

func TestListUserProfiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".umbraco")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeProfile := func(name, baseURL string) {
		payload := `{"baseUrl":"` + baseURL + `","clientId":"id","clientSecret":"secret"}`
		if err := os.WriteFile(filepath.Join(dir, name), []byte(payload), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeProfile("config.json", "https://live.example.test")
	writeProfile("dev.config.json", "https://dev.example.test")
	// Files that are not profile configs are skipped.
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	profiles, err := ListUserProfiles()
	if err != nil {
		t.Fatalf("ListUserProfiles failed: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d: %+v", len(profiles), profiles)
	}
	if !profiles[0].IsDefault || profiles[0].Name != "default" || !profiles[0].Active {
		t.Fatalf("expected default profile first and active without a marker, got %+v", profiles[0])
	}
	if profiles[1].Name != "dev" || profiles[1].Active {
		t.Fatalf("expected inactive dev profile second, got %+v", profiles[1])
	}
	if profiles[1].Config.BaseURL != "https://dev.example.test" {
		t.Fatalf("expected profile config loaded, got %+v", profiles[1].Config)
	}

	// Activating dev flips the Active flags.
	if err := SetActiveProfile("dev"); err != nil {
		t.Fatal(err)
	}
	profiles, err = ListUserProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if profiles[0].Active || !profiles[1].Active {
		t.Fatalf("expected dev active after SetActiveProfile, got %+v", profiles)
	}

	pubPath, err := ProfileConfigPath("dev")
	if err != nil || !strings.HasSuffix(pubPath, filepath.Join(".umbraco", "dev.config.json")) {
		t.Fatalf("unexpected ProfileConfigPath %q, %v", pubPath, err)
	}
}
