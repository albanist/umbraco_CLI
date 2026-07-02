// Package version exposes the published umbraco-cli release identifier.
//
// VERSION is the canonical source of truth for the release identifier:
// the Go code embeds it via go:embed and surfaces it through Current().
package version

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var raw string

// Current returns the published umbraco-cli release identifier (trimmed of whitespace).
func Current() string {
	return strings.TrimSpace(raw)
}
