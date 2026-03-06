package commands

import (
	"context"

	"github.com/spf13/cobra"

	"umbraco-cli/internal/api"
)

func RegisterLogs(root *cobra.Command, deps Dependencies) {
	logs := &cobra.Command{Use: "logs", Short: "Log and diagnostics operations"}
	logs.AddCommand(logsList(deps))
	logs.AddCommand(logsLevels(deps))
	logs.AddCommand(logsLevelCount(deps))
	logs.AddCommand(logsTemplates(deps))
	logs.AddCommand(logsSearch(deps))
	root.AddCommand(logs)
}

func logsList(deps Dependencies) *cobra.Command {
	var paramsRaw string
	var level string
	var from string
	var to string
	var skip int
	var take int

	cmd := &cobra.Command{Use: "list", Short: "List log entries", RunE: func(cmd *cobra.Command, args []string) error {
		params, err := parseParams(paramsRaw)
		if err != nil {
			return err
		}
		if params == nil {
			params = map[string]any{}
			if level != "" {
				params["level"] = level
			}
			if from != "" {
				params["startDate"] = from
			}
			if to != "" {
				params["endDate"] = to
			}
			if skip >= 0 {
				params["skip"] = skip
			}
			if take >= 0 {
				params["take"] = take
			}
		}
		result, err := deps.Client.Get(context.Background(), "/log-viewer", api.RequestOptions{Params: params})
		if err != nil {
			return err
		}
		return printResult(cmd, deps, result)
	}}

	cmd.Flags().StringVar(&paramsRaw, "params", "", "Filter params as JSON")
	cmd.Flags().StringVar(&level, "level", "", "Log level")
	cmd.Flags().StringVar(&from, "from", "", "Start date (ISO)")
	cmd.Flags().StringVar(&to, "to", "", "End date (ISO)")
	cmd.Flags().IntVar(&skip, "skip", -1, "Skip count")
	cmd.Flags().IntVar(&take, "take", -1, "Take count")
	return cmd
}

func logsLevels(deps Dependencies) *cobra.Command {
	return &cobra.Command{Use: "levels", Short: "List log levels", RunE: func(cmd *cobra.Command, args []string) error {
		result, err := deps.Client.Get(context.Background(), "/log-viewer/levels", api.RequestOptions{})
		if err != nil {
			return err
		}
		return printResult(cmd, deps, result)
	}}
}

func logsLevelCount(deps Dependencies) *cobra.Command {
	var paramsRaw string
	var from string
	var to string
	cmd := &cobra.Command{Use: "level-count", Short: "Get count per level", RunE: func(cmd *cobra.Command, args []string) error {
		params, err := parseParams(paramsRaw)
		if err != nil {
			return err
		}
		if params == nil {
			params = map[string]any{}
			if from != "" {
				params["startDate"] = from
			}
			if to != "" {
				params["endDate"] = to
			}
		}
		result, err := deps.Client.Get(context.Background(), "/log-viewer/level-count", api.RequestOptions{Params: params})
		if err != nil {
			return err
		}
		return printResult(cmd, deps, result)
	}}
	cmd.Flags().StringVar(&paramsRaw, "params", "", "Filter params as JSON")
	cmd.Flags().StringVar(&from, "from", "", "Start date (ISO)")
	cmd.Flags().StringVar(&to, "to", "", "End date (ISO)")
	return cmd
}

func logsTemplates(deps Dependencies) *cobra.Command {
	return &cobra.Command{Use: "templates", Short: "List log templates", RunE: func(cmd *cobra.Command, args []string) error {
		result, err := deps.Client.Get(context.Background(), "/log-viewer/templates", api.RequestOptions{})
		if err != nil {
			return err
		}
		return printResult(cmd, deps, result)
	}}
}

func logsSearch(deps Dependencies) *cobra.Command {
	var paramsRaw string
	var filterExpression string
	var skip int
	var take int
	cmd := &cobra.Command{Use: "search", Short: "Search logs", RunE: func(cmd *cobra.Command, args []string) error {
		params, err := parseParams(paramsRaw)
		if err != nil {
			return err
		}
		if params == nil {
			params = map[string]any{}
			if filterExpression != "" {
				params["filterExpression"] = filterExpression
			}
			if skip >= 0 {
				params["skip"] = skip
			}
			if take >= 0 {
				params["take"] = take
			}
		}
		result, err := deps.Client.Get(context.Background(), "/log-viewer/search", api.RequestOptions{Params: params})
		if err != nil {
			return err
		}
		return printResult(cmd, deps, result)
	}}
	cmd.Flags().StringVar(&paramsRaw, "params", "", "Search params as JSON")
	cmd.Flags().StringVar(&filterExpression, "filter-expression", "", "Filter expression")
	cmd.Flags().IntVar(&skip, "skip", -1, "Skip count")
	cmd.Flags().IntVar(&take, "take", -1, "Take count")
	return cmd
}
