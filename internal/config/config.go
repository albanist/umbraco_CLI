package config

import (
	"fmt"
	"os"
	"strings"
)

type OutputFormat string

const (
	OutputJSON  OutputFormat = "json"
	OutputTable OutputFormat = "table"
	OutputPlain OutputFormat = "plain"
)

type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	OutputFormat OutputFormat
}

func Load() (Config, error) {
	cfg := Config{
		BaseURL:      getenvDefault("UMBRACO_BASE_URL", "https://localhost:44391"),
		ClientID:     strings.TrimSpace(os.Getenv("UMBRACO_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("UMBRACO_CLIENT_SECRET")),
	}

	if output := strings.TrimSpace(os.Getenv("UMBRACO_OUTPUT_FORMAT")); output != "" {
		format, err := ParseOutputFormat(output)
		if err != nil {
			return Config{}, err
		}
		cfg.OutputFormat = format
	}

	return cfg, nil
}

func getenvDefault(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func ParseOutputFormat(raw string) (OutputFormat, error) {
	switch OutputFormat(strings.ToLower(strings.TrimSpace(raw))) {
	case OutputJSON:
		return OutputJSON, nil
	case OutputTable:
		return OutputTable, nil
	case OutputPlain:
		return OutputPlain, nil
	default:
		return "", fmt.Errorf("invalid output format %q (expected json|table|plain)", raw)
	}
}

func (c Config) ValidateAuth() error {
	if c.ClientID == "" || c.ClientSecret == "" {
		return fmt.Errorf("missing UMBRACO_CLIENT_ID or UMBRACO_CLIENT_SECRET")
	}
	return nil
}
