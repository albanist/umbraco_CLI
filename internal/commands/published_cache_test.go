package commands

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func buildPublishedCacheRoot(deps Dependencies) *cobra.Command {
	root := &cobra.Command{Use: "umbraco", SilenceErrors: true, SilenceUsage: true}
	root.SetErr(io.Discard)
	if deps.OutputFlag != nil {
		root.PersistentFlags().StringVarP(deps.OutputFlag, "output", "o", *deps.OutputFlag, "Output format: json, table, plain")
	}
	RegisterPublishedCache(root, deps)
	return root
}

func TestPublishedCacheStatusFallsBackToLegacyRoute(t *testing.T) {
	var requests []string
	deps := endpointDeps(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			return endpointJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		case "/umbraco/management/api/v1/published-cache/rebuild/status":
			requests = append(requests, req.URL.Path)
			return endpointJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
		case "/umbraco/management/api/v1/published-cache/status":
			requests = append(requests, req.URL.Path)
			return endpointJSONResponse(http.StatusOK, `"cache is ok"`), nil
		default:
			return endpointJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
		}
	})

	out, err := execute(buildPublishedCacheRoot(deps), "published-cache", "status")
	if err != nil {
		t.Fatalf("published-cache status failed: %v", err)
	}
	if len(requests) != 2 || requests[0] != "/umbraco/management/api/v1/published-cache/rebuild/status" {
		t.Fatalf("expected modern-first fallback, got %v", requests)
	}
	if !strings.Contains(out, "cache is ok") {
		t.Fatalf("expected status payload, got %s", out)
	}
}

func TestPublishedCacheRebuildRequiresForceOrDryRun(t *testing.T) {
	deps := endpointDeps(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("no HTTP request expected without --force or --dry-run")
		return nil, nil
	})

	_, err := execute(buildPublishedCacheRoot(deps), "published-cache", "rebuild")
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("expected force/dry-run gate, got %v", err)
	}
}

func TestPublishedCacheRebuildPostsWithForce(t *testing.T) {
	var requestedPath, requestedMethod string
	deps := endpointDeps(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			return endpointJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		default:
			requestedPath = req.URL.Path
			requestedMethod = req.Method
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
	})

	out, err := execute(buildPublishedCacheRoot(deps), "published-cache", "rebuild", "--force")
	if err != nil {
		t.Fatalf("rebuild failed: %v", err)
	}
	if requestedMethod != http.MethodPost || requestedPath != "/umbraco/management/api/v1/published-cache/rebuild" {
		t.Fatalf("unexpected request %s %s", requestedMethod, requestedPath)
	}
	if !strings.Contains(out, `"rebuilding": true`) && !strings.Contains(out, `"rebuilding":true`) {
		t.Fatalf("expected empty 200 reported as rebuilding:true, got %s", out)
	}
}

func TestPublishedCacheRebuildDryRunSkipsRequest(t *testing.T) {
	requests := 0
	deps := endpointDeps(func(req *http.Request) (*http.Response, error) {
		requests++
		return endpointJSONResponse(http.StatusOK, `{}`), nil
	})

	out, err := execute(buildPublishedCacheRoot(deps), "published-cache", "rebuild", "--dry-run")
	if err != nil {
		t.Fatalf("rebuild dry-run failed: %v", err)
	}
	if requests != 0 {
		t.Fatalf("dry-run must not hit the API, saw %d requests", requests)
	}
	if !strings.Contains(out, `"dryRun": true`) && !strings.Contains(out, `"dryRun":true`) {
		t.Fatalf("expected dry-run preview, got %s", out)
	}
}

func TestPublishedCacheReloadPosts(t *testing.T) {
	var requestedPath, requestedMethod string
	deps := endpointDeps(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			return endpointJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		default:
			requestedPath = req.URL.Path
			requestedMethod = req.Method
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
	})

	out, err := execute(buildPublishedCacheRoot(deps), "published-cache", "reload")
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if requestedMethod != http.MethodPost || requestedPath != "/umbraco/management/api/v1/published-cache/reload" {
		t.Fatalf("unexpected request %s %s", requestedMethod, requestedPath)
	}
	if !strings.Contains(out, `"reloaded": true`) && !strings.Contains(out, `"reloaded":true`) {
		t.Fatalf("expected empty 200 reported as reloaded:true, got %s", out)
	}
}

func TestPublishedCacheRebuildWithWaitPollsUntilDone(t *testing.T) {
	var rebuilds, statusPolls int
	deps := endpointDeps(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			return endpointJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		case "/umbraco/management/api/v1/published-cache/rebuild":
			rebuilds++
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
		case "/umbraco/management/api/v1/published-cache/rebuild/status":
			statusPolls++
			if statusPolls == 1 {
				return endpointJSONResponse(http.StatusOK, `{"isRebuilding":true}`), nil
			}
			return endpointJSONResponse(http.StatusOK, `{"isRebuilding":false}`), nil
		default:
			return endpointJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
		}
	})

	out, err := execute(buildPublishedCacheRoot(deps), "published-cache", "rebuild", "--force", "--wait", "--poll-interval", "1ms", "--timeout", "5s")
	if err != nil {
		t.Fatalf("rebuild --wait failed: %v", err)
	}
	if rebuilds != 1 || statusPolls != 2 {
		t.Fatalf("expected 1 rebuild + 2 status polls, got %d + %d", rebuilds, statusPolls)
	}
	if !strings.Contains(out, `"rebuilt": true`) && !strings.Contains(out, `"rebuilt":true`) {
		t.Fatalf("expected rebuilt:true after polling, got %s", out)
	}
}

func TestPublishedCacheRebuildWaitTimesOut(t *testing.T) {
	deps := endpointDeps(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/umbraco/management/api/v1/security/back-office/token":
			return endpointJSONResponse(http.StatusOK, `{"access_token":"token-123","expires_in":3600}`), nil
		case "/umbraco/management/api/v1/published-cache/rebuild":
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
		default:
			return endpointJSONResponse(http.StatusOK, `{"isRebuilding":true}`), nil
		}
	})

	_, err := execute(buildPublishedCacheRoot(deps), "published-cache", "rebuild", "--force", "--wait", "--poll-interval", "1ms", "--timeout", "10ms")
	if err == nil || !strings.Contains(err.Error(), "still rebuilding") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestPublishedCacheRebuildRejectsDryRunWithWait(t *testing.T) {
	deps := endpointDeps(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("no HTTP request expected for flag validation error")
		return nil, nil
	})

	_, err := execute(buildPublishedCacheRoot(deps), "published-cache", "rebuild", "--dry-run", "--wait")
	if err == nil || !strings.Contains(err.Error(), "--wait has nothing to poll for") {
		t.Fatalf("expected dry-run/wait conflict error, got %v", err)
	}
}
