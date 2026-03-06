package commands

import (
	"context"

	"github.com/spf13/cobra"

	"umbraco-cli/internal/api"
)

func RegisterServer(root *cobra.Command, deps Dependencies) {
	server := &cobra.Command{Use: "server", Short: "Server information and diagnostics"}
	server.AddCommand(readOnlyEndpoint(deps, "status", "Get server status", "/server/status"))
	server.AddCommand(readOnlyEndpoint(deps, "info", "Get server info", "/server/info"))
	server.AddCommand(readOnlyEndpoint(deps, "config", "Get server config", "/server/config"))
	server.AddCommand(readOnlyEndpoint(deps, "troubleshoot", "Run troubleshooting checks", "/server/troubleshoot"))
	server.AddCommand(readOnlyEndpoint(deps, "upgrade-check", "Check upgrade readiness", "/server/upgrade-check"))
	root.AddCommand(server)
}

func readOnlyEndpoint(deps Dependencies, use string, short string, path string) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, RunE: func(cmd *cobra.Command, args []string) error {
		result, err := deps.Client.Get(context.Background(), path, api.RequestOptions{})
		if err != nil {
			return err
		}
		return printResult(cmd, deps, result)
	}}
}
