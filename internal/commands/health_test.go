package commands

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func buildHealthRoot(deps Dependencies) *cobra.Command {
	root := &cobra.Command{Use: "umbraco", SilenceErrors: true, SilenceUsage: true}
	root.SetErr(io.Discard)
	if deps.OutputFlag != nil {
		root.PersistentFlags().StringVarP(deps.OutputFlag, "output", "o", *deps.OutputFlag, "Output format: json, table, plain")
	}
	RegisterHealth(root, deps)
	return root
}

func TestHealthGroupsListsGroups(t *testing.T) {
	var requestedPath string
	deps := endpointDeps(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			return endpointJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		default:
			requestedPath = req.URL.Path
			return endpointJSONResponse(http.StatusOK, `{"items":[{"name":"Configuration"}],"total":1}`), nil
		}
	})

	out, err := execute(buildHealthRoot(deps), "health", "groups")
	if err != nil {
		t.Fatalf("health groups failed: %v", err)
	}
	if requestedPath != "/umbraco/management/api/v1/health-check-group" {
		t.Fatalf("unexpected path %q", requestedPath)
	}
	if !strings.Contains(out, "Configuration") {
		t.Fatalf("expected group in output, got %s", out)
	}
}

func TestHealthGroupEscapesNameSegment(t *testing.T) {
	var requestedURI string
	deps := endpointDeps(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			return endpointJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		default:
			requestedURI = req.URL.RequestURI()
			return endpointJSONResponse(http.StatusOK, `{"name":"Data Integrity","checks":[]}`), nil
		}
	})

	if _, err := execute(buildHealthRoot(deps), "health", "group", "Data Integrity"); err != nil {
		t.Fatalf("health group failed: %v", err)
	}
	if requestedURI != "/umbraco/management/api/v1/health-check-group/Data%20Integrity" {
		t.Fatalf("expected escaped group name in path, got %q", requestedURI)
	}
}

func TestHealthRunHitsGroupRunEndpoint(t *testing.T) {
	var requestedPath string
	deps := endpointDeps(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			return endpointJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		default:
			requestedPath = req.URL.Path
			return endpointJSONResponse(http.StatusOK, `{"checks":[{"name":"Macro errors","results":[]}]}`), nil
		}
	})

	if _, err := execute(buildHealthRoot(deps), "health", "run", "Configuration"); err != nil {
		t.Fatalf("health run failed: %v", err)
	}
	if requestedPath != "/umbraco/management/api/v1/health-check-group/Configuration/run" {
		t.Fatalf("unexpected path %q", requestedPath)
	}
}

func TestHealthActionPostsPayload(t *testing.T) {
	var requestedPath, requestedMethod, requestedBody string
	deps := endpointDeps(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			return endpointJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		default:
			requestedPath = req.URL.Path
			requestedMethod = req.Method
			body, _ := io.ReadAll(req.Body)
			requestedBody = string(body)
			return endpointJSONResponse(http.StatusOK, `{"success":true}`), nil
		}
	})

	if _, err := execute(buildHealthRoot(deps), "health", "action", "action-1", "--json", `{"alias":"fixConfig"}`); err != nil {
		t.Fatalf("health action failed: %v", err)
	}
	if requestedMethod != http.MethodPost || requestedPath != "/umbraco/management/api/v1/health-check/action-1" {
		t.Fatalf("unexpected request %s %s", requestedMethod, requestedPath)
	}
	if !strings.Contains(requestedBody, `"fixConfig"`) {
		t.Fatalf("expected payload forwarded, got %q", requestedBody)
	}
}

func TestHealthActionDryRunSkipsRequest(t *testing.T) {
	requests := 0
	deps := endpointDeps(func(req *http.Request) (*http.Response, error) {
		requests++
		return endpointJSONResponse(http.StatusOK, `{}`), nil
	})

	out, err := execute(buildHealthRoot(deps), "health", "action", "action-1", "--dry-run")
	if err != nil {
		t.Fatalf("health action dry-run failed: %v", err)
	}
	if requests != 0 {
		t.Fatalf("dry-run must not hit the API, saw %d requests", requests)
	}
	if !strings.Contains(out, `"dryRun": true`) && !strings.Contains(out, `"dryRun":true`) {
		t.Fatalf("expected dry-run preview in output, got %s", out)
	}
}

func TestHealthActionRejectsInvalidJSON(t *testing.T) {
	deps := endpointDeps(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("no HTTP request expected for invalid payload")
		return nil, nil
	})

	if _, err := execute(buildHealthRoot(deps), "health", "action", "action-1", "--json", "{not-json"); err == nil {
		t.Fatalf("expected error for invalid JSON payload")
	}
}
