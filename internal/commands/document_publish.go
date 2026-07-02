package commands

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"umbraco-cli/internal/api"
)

// Publish and unpublish operations for the document command group,
// including the invariant-content race retry.

func documentPublish(deps Dependencies) *cobra.Command {
	var jsonPayload string
	var culture string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "publish <id>",
		Short: "Publish a document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := documentPublishBody(jsonPayload, culture)
			if err != nil {
				return err
			}
			result, err := deps.Client.Put(cmd.Context(), api.JoinPath("/document/%s/publish", args[0]), body, api.RequestOptions{DryRun: dryRun})
			if err != nil {
				return err
			}
			return printMutationResult(cmd, deps, "published", result, dryRun)
		},
	}
	cmd.Flags().StringVar(&jsonPayload, "json", "", "Publish payload as JSON")
	cmd.Flags().StringVar(&culture, "culture", "", "Culture shortcut")
	addDryRunFlag(cmd, &dryRun)
	return cmd
}

// invariantRaceMaxAttempts is the upper bound on retries for the spurious
// "culture for invariant content" 400 that the Management API throws under
// rapid back-to-back save-and-publish loops. The error is timing-dependent
// and clears on retry; 4 attempts with exponential-ish backoff matches what
// the bug report saw work in practice.
const invariantRaceMaxAttempts = 4

// publishWithInvariantRaceRetry PUTs the publish body and retries on the
// specific 400 "culture for invariant content" error that Umbraco intermittently
// returns under tight save-and-publish loops on invariant content. The
// payload is valid (verified via --dry-run in the bug report) — the same
// request succeeds on retry, so the retry is the right correctness-preserving
// workaround at the CLI layer. Other 400s are surfaced immediately.
func publishWithInvariantRaceRetry(ctx context.Context, client *api.Client, id string, body map[string]any, opts api.RequestOptions) (any, error) {
	path := api.JoinPath("/document/%s/publish", id)
	var lastErr error
	for attempt := 0; attempt < invariantRaceMaxAttempts; attempt++ {
		result, err := client.Put(ctx, path, body, opts)
		if err == nil {
			return result, nil
		}
		if opts.DryRun || !isInvariantContentRaceError(err) || attempt == invariantRaceMaxAttempts-1 {
			return nil, err
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(invariantRaceBackoffs[attempt]):
		}
	}
	return nil, lastErr
}

// isInvariantContentRaceError matches the spurious 400 the Management API
// returns under the save-and-publish race. The payload looks like
// {"detail":"One or more property values specify a culture for an [invariant content]"}.
// Substring-match on "invariant content" inside the rendered error is robust
// to message phrasing tweaks without false-positiving on unrelated 400s.
func isInvariantContentRaceError(err error) bool {
	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.StatusCode != 400 {
		return false
	}
	return strings.Contains(apiErr.Error(), "invariant content")
}

func documentPublishBody(jsonPayload string, culture string) (map[string]any, error) {
	if strings.TrimSpace(jsonPayload) != "" {
		return parsePayload(jsonPayload)
	}
	if strings.TrimSpace(culture) != "" {
		return map[string]any{"cultures": []any{culture}}, nil
	}
	return map[string]any{
		"publishSchedules": []any{
			map[string]any{"culture": nil},
		},
	}, nil
}

func documentUnpublish(deps Dependencies) *cobra.Command {
	var jsonPayload string
	var culture string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "unpublish <id>",
		Short: "Unpublish a document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var body map[string]any
			var err error
			if jsonPayload != "" {
				body, err = parsePayload(jsonPayload)
			} else if culture != "" {
				body = map[string]any{"cultures": []any{culture}}
			} else {
				body = map[string]any{}
			}
			if err != nil {
				return err
			}
			result, err := deps.Client.Put(cmd.Context(), api.JoinPath("/document/%s/unpublish", args[0]), body, api.RequestOptions{DryRun: dryRun})
			if err != nil {
				return err
			}
			return printMutationResult(cmd, deps, "unpublished", result, dryRun)
		},
	}
	cmd.Flags().StringVar(&jsonPayload, "json", "", "Unpublish payload as JSON")
	cmd.Flags().StringVar(&culture, "culture", "", "Culture shortcut")
	addDryRunFlag(cmd, &dryRun)
	return cmd
}
