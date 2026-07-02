package commands

import (
	"github.com/spf13/cobra"

	"umbraco-cli/internal/api"
)

func RegisterPublishedCache(root *cobra.Command, deps Dependencies) {
	cache := &cobra.Command{Use: "published-cache", Short: "Published content cache operations"}
	cache.AddCommand(readOnlyEndpointWithFallback(deps, "status", "Get published cache rebuild status", "/published-cache/rebuild/status", "/published-cache/status"))
	cache.AddCommand(publishedCacheRebuild(deps))
	cache.AddCommand(publishedCacheReload(deps))
	root.AddCommand(cache)
}

func publishedCacheRebuild(deps Dependencies) *cobra.Command {
	var force bool
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "rebuild",
		Short: "Rebuild the published content cache from the database",
		Long:  "POST /published-cache/rebuild. Rebuilds the published content cache from the database — the standard fix for stale published content. Expensive on large sites; poll 'published-cache status' to see when the rebuild finishes.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireForceOrDryRun(cmd, "rebuilds the entire published cache and is expensive on large sites", force, dryRun); err != nil {
				return err
			}
			result, err := deps.Client.Post(cmd.Context(), "/published-cache/rebuild", nil, api.RequestOptions{DryRun: dryRun})
			if err != nil {
				return err
			}
			return printMutationResult(cmd, deps, "rebuilding", result, dryRun)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Confirm the rebuild")
	addDryRunFlag(cmd, &dryRun)
	return cmd
}

func publishedCacheReload(deps Dependencies) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "reload",
		Short: "Reload the in-memory published cache",
		Long:  "POST /published-cache/reload. Reloads the in-memory published cache from the cache store without a database rebuild; much cheaper than 'published-cache rebuild'.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := deps.Client.Post(cmd.Context(), "/published-cache/reload", nil, api.RequestOptions{DryRun: dryRun})
			if err != nil {
				return err
			}
			return printMutationResult(cmd, deps, "reloaded", result, dryRun)
		},
	}
	addDryRunFlag(cmd, &dryRun)
	return cmd
}
