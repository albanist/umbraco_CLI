package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"umbraco-cli/internal/api"
)

func RegisterDatatype(root *cobra.Command, deps Dependencies) {
	datatype := &cobra.Command{Use: "datatype", Short: "Data type operations"}
	datatype.AddCommand(datatypeGet(deps))
	datatype.AddCommand(datatypeList(deps))
	datatype.AddCommand(datatypeRoot(deps))
	datatype.AddCommand(datatypeSearch(deps))
	datatype.AddCommand(datatypeIsUsed(deps))
	datatype.AddCommand(datatypeCreate(deps))
	datatype.AddCommand(datatypeUpdate(deps))
	datatype.AddCommand(datatypeDelete(deps))
	root.AddCommand(datatype)
}

func datatypeGet(deps Dependencies) *cobra.Command {
	var fields string
	cmd := &cobra.Command{Use: "get <id>", Short: "Get data type by ID", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		result, err := deps.Client.Get(context.Background(), fmt.Sprintf("/data-type/%s", args[0]), api.RequestOptions{Fields: fields})
		if err != nil {
			return err
		}
		return printResult(cmd, deps, result)
	}}
	cmd.Flags().StringVar(&fields, "fields", "", "Limit response fields")
	return cmd
}

func datatypeList(deps Dependencies) *cobra.Command {
	var fields string
	cmd := &cobra.Command{Use: "list", Short: "List data types", RunE: func(cmd *cobra.Command, args []string) error {
		result, err := deps.Client.Get(context.Background(), "/data-type", api.RequestOptions{Fields: fields})
		if err != nil {
			return err
		}
		return printResult(cmd, deps, result)
	}}
	cmd.Flags().StringVar(&fields, "fields", "", "Limit response fields")
	return cmd
}

func datatypeRoot(deps Dependencies) *cobra.Command {
	return &cobra.Command{Use: "root", Short: "Get root data types", RunE: func(cmd *cobra.Command, args []string) error {
		result, err := deps.Client.Get(context.Background(), "/data-type/root", api.RequestOptions{})
		if err != nil {
			return err
		}
		return printResult(cmd, deps, result)
	}}
}

func datatypeSearch(deps Dependencies) *cobra.Command {
	var paramsRaw string
	var query string
	cmd := &cobra.Command{Use: "search", Short: "Search data types", RunE: func(cmd *cobra.Command, args []string) error {
		params, err := parseParams(paramsRaw)
		if err != nil {
			return err
		}
		if params == nil {
			if query == "" {
				return fmt.Errorf("datatype search requires either --params or --query")
			}
			params = map[string]any{"query": query}
		}
		result, err := deps.Client.Get(context.Background(), "/data-type/search", api.RequestOptions{Params: params})
		if err != nil {
			return err
		}
		return printResult(cmd, deps, result)
	}}
	cmd.Flags().StringVar(&paramsRaw, "params", "", "Query parameters as JSON")
	cmd.Flags().StringVar(&query, "query", "", "Search query")
	return cmd
}

func datatypeIsUsed(deps Dependencies) *cobra.Command {
	return &cobra.Command{Use: "is-used <id>", Short: "Check whether a data type is in use", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		result, err := deps.Client.Get(context.Background(), fmt.Sprintf("/data-type/%s/is-used", args[0]), api.RequestOptions{})
		if err != nil {
			return err
		}
		return printResult(cmd, deps, result)
	}}
}

func datatypeCreate(deps Dependencies) *cobra.Command {
	var jsonPayload string
	var dryRun bool
	cmd := &cobra.Command{Use: "create", Short: "Create data type", RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireValue("--json", jsonPayload); err != nil {
			return err
		}
		body, err := parsePayload(jsonPayload)
		if err != nil {
			return err
		}
		result, err := deps.Client.Post(context.Background(), "/data-type", body, api.RequestOptions{DryRun: dryRun})
		if err != nil {
			return err
		}
		return printResult(cmd, deps, result)
	}}
	cmd.Flags().StringVar(&jsonPayload, "json", "", "Create payload as JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate request without executing")
	return cmd
}

func datatypeUpdate(deps Dependencies) *cobra.Command {
	var jsonPayload string
	var dryRun bool
	cmd := &cobra.Command{Use: "update <id>", Short: "Update data type", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireValue("--json", jsonPayload); err != nil {
			return err
		}
		body, err := parsePayload(jsonPayload)
		if err != nil {
			return err
		}
		result, err := deps.Client.Put(context.Background(), fmt.Sprintf("/data-type/%s", args[0]), body, api.RequestOptions{DryRun: dryRun})
		if err != nil {
			return err
		}
		return printResult(cmd, deps, result)
	}}
	cmd.Flags().StringVar(&jsonPayload, "json", "", "Update payload as JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate request without executing")
	return cmd
}

func datatypeDelete(deps Dependencies) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{Use: "delete <id>", Short: "Delete data type", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		result, err := deps.Client.Delete(context.Background(), fmt.Sprintf("/data-type/%s", args[0]), api.RequestOptions{DryRun: dryRun})
		if err != nil {
			return err
		}
		return printResult(cmd, deps, result)
	}}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate request without executing")
	return cmd
}
