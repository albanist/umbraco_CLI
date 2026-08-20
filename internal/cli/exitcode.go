package cli

import "errors"

// exitCoder is implemented by error types that map to a documented exit
// code: schema-diff differences (2), auth failures (3), API error
// responses (4), deploy watch failed (5), deploy watch timeout —
// status unknown (6), deploy status drift/missing found (7).
type exitCoder interface {
	ExitCode() int
}

// ExitCode resolves the process exit code for a command error. Errors
// without a specific contract exit 1 (usage errors, invalid flags, local
// failures); nil exits 0.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var coder exitCoder
	if errors.As(err, &coder) {
		return coder.ExitCode()
	}
	return 1
}
