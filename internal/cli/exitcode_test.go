package cli

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"umbraco-cli/internal/api"
	"umbraco-cli/internal/auth"
)

func TestExitCodeContract(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{name: "success", err: nil, want: 0},
		{name: "generic error", err: errors.New("boom"), want: 1},
		{name: "auth failure", err: &auth.Error{Err: errors.New("401 invalid_client")}, want: 3},
		{name: "wrapped auth failure", err: fmt.Errorf("context: %w", &auth.Error{Err: errors.New("bad creds")}), want: 3},
		{name: "api error", err: &api.APIError{StatusCode: http.StatusNotFound, Method: "GET", Path: "/document/x"}, want: 4},
		{name: "wrapped api error", err: fmt.Errorf("fetch failed: %w", &api.APIError{StatusCode: 500}), want: 4},
	}
	for _, tc := range cases {
		if got := ExitCode(tc.err); got != tc.want {
			t.Fatalf("%s: ExitCode = %d, want %d", tc.name, got, tc.want)
		}
	}
}
