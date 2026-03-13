package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadResolvedConfigAppliesProjectAndEnvPrecedence(t *testing.T) {
	workingDir := t.TempDir()
	homeDir := t.TempDir()

	userConfigDir := filepath.Join(homeDir, ".umbraco")
	if err := os.MkdirAll(userConfigDir, 0o755); err != nil {
		t.Fatalf("failed to create user config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(userConfigDir, "config.json"), []byte(`{
  "baseUrl": "https://localhost:44391",
  "clientId": "user-client",
  "clientSecret": "user-secret",
  "outputFormat": "table"
}`), 0o644); err != nil {
		t.Fatalf("failed to write user config: %v", err)
	}

	if err := os.WriteFile(filepath.Join(workingDir, ".env"), []byte(`
UMBRACO_BASE_URL="https://localhost:44314/umbraco"
UMBRACO_CLIENT_ID=dotenv-client
IGNORED_VALUE=should-not-load
`), 0o644); err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}

	if err := os.WriteFile(filepath.Join(workingDir, ".umbracorc.json"), []byte(`{
  "clientSecret": "project-secret",
  "outputFormat": "plain"
}`), 0o644); err != nil {
		t.Fatalf("failed to write project config: %v", err)
	}

	cfg, err := loadResolvedConfig(workingDir, homeDir, map[string]string{
		"UMBRACO_CLIENT_ID":     "env-client",
		"UMBRACO_OUTPUT_FORMAT": "json",
	})
	if err != nil {
		t.Fatalf("loadResolvedConfig failed: %v", err)
	}

	if cfg.BaseURL != "https://localhost:44314" {
		t.Fatalf("expected normalized base URL from project .env, got %q", cfg.BaseURL)
	}
	if cfg.ClientID != "env-client" {
		t.Fatalf("expected env client ID to win, got %q", cfg.ClientID)
	}
	if cfg.ClientSecret != "project-secret" {
		t.Fatalf("expected project config client secret to win, got %q", cfg.ClientSecret)
	}
	if cfg.OutputFormat != OutputJSON {
		t.Fatalf("expected env output format to win, got %q", cfg.OutputFormat)
	}
}

func TestLoadResolvedConfigDiscoversBaseURLFromLaunchSettings(t *testing.T) {
	workingDir := t.TempDir()
	launchSettingsDir := filepath.Join(workingDir, "Properties")
	if err := os.MkdirAll(launchSettingsDir, 0o755); err != nil {
		t.Fatalf("failed to create Properties dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(launchSettingsDir, "launchSettings.json"), []byte(`{
  "profiles": {
    "https": {
      "applicationUrl": "https://localhost:44314;http://localhost:5000"
    }
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write launchSettings.json: %v", err)
	}

	cfg, err := loadResolvedConfig(workingDir, "", map[string]string{})
	if err != nil {
		t.Fatalf("loadResolvedConfig failed: %v", err)
	}

	if cfg.BaseURL != "https://localhost:44314" {
		t.Fatalf("expected HTTPS launchSettings URL, got %q", cfg.BaseURL)
	}
}

func TestLoadResolvedConfigDiscoversBaseURLFromAppSettings(t *testing.T) {
	workingDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workingDir, "appsettings.Development.json"), []byte(`{
  "Kestrel": {
    "Endpoints": {
      "Https": {
        "Url": "https://localhost:44314"
      },
      "Http": {
        "Url": "http://localhost:5000"
      }
    }
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write appsettings.Development.json: %v", err)
	}

	cfg, err := loadResolvedConfig(workingDir, "", map[string]string{})
	if err != nil {
		t.Fatalf("loadResolvedConfig failed: %v", err)
	}

	if cfg.BaseURL != "https://localhost:44314" {
		t.Fatalf("expected HTTPS appsettings URL, got %q", cfg.BaseURL)
	}
}

func TestLoadResolvedConfigEnvOverridesDiscoveredBaseURL(t *testing.T) {
	workingDir := t.TempDir()
	launchSettingsDir := filepath.Join(workingDir, "Properties")
	if err := os.MkdirAll(launchSettingsDir, 0o755); err != nil {
		t.Fatalf("failed to create Properties dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(launchSettingsDir, "launchSettings.json"), []byte(`{
  "profiles": {
    "https": {
      "applicationUrl": "https://localhost:44314"
    }
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write launchSettings.json: %v", err)
	}

	cfg, err := loadResolvedConfig(workingDir, "", map[string]string{
		"UMBRACO_BASE_URL": "https://localhost:44399/umbraco",
	})
	if err != nil {
		t.Fatalf("loadResolvedConfig failed: %v", err)
	}

	if cfg.BaseURL != "https://localhost:44399" {
		t.Fatalf("expected env base URL to override discovery, got %q", cfg.BaseURL)
	}
}

func TestLoadDotEnvConfigIgnoresNonUmbracoVariables(t *testing.T) {
	workingDir := t.TempDir()
	path := filepath.Join(workingDir, ".env")
	if err := os.WriteFile(path, []byte(`
export UMBRACO_CLIENT_ID="dotenv-client"
UMBRACO_CLIENT_SECRET='dotenv-secret'
UNRELATED_KEY=ignored
`), 0o644); err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}

	cfg, err := loadDotEnvConfig(path)
	if err != nil {
		t.Fatalf("loadDotEnvConfig failed: %v", err)
	}

	if cfg.ClientID != "dotenv-client" || cfg.ClientSecret != "dotenv-secret" {
		t.Fatalf("unexpected dotenv config: %+v", cfg)
	}
	if cfg.BaseURL != "" || cfg.OutputFormat != "" {
		t.Fatalf("expected unrelated values to stay empty, got %+v", cfg)
	}
}

func TestLoadResolvedConfigRejectsInvalidOutputFormat(t *testing.T) {
	workingDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workingDir, ".umbracorc.json"), []byte(`{
  "outputFormat": "xml"
}`), 0o644); err != nil {
		t.Fatalf("failed to write project config: %v", err)
	}

	_, err := loadResolvedConfig(workingDir, "", map[string]string{})
	if err == nil {
		t.Fatalf("expected invalid output format to fail")
	}
}
