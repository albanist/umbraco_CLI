package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"umbraco-cli/internal/api"
)

func RegisterDocument(root *cobra.Command, deps Dependencies) {
	document := &cobra.Command{
		Use:     "document",
		Aliases: []string{"doc"},
		Short:   "Document and content management operations",
	}

	document.AddCommand(documentGet(deps))
	document.AddCommand(documentRoot(deps))
	document.AddCommand(documentChildren(deps))
	document.AddCommand(documentAncestors(deps))
	document.AddCommand(documentSearch(deps))
	document.AddCommand(documentCreate(deps))
	document.AddCommand(documentUpdate(deps))
	document.AddCommand(documentUpdateProperties(deps))
	document.AddCommand(documentPublish(deps))
	document.AddCommand(documentUnpublish(deps))
	document.AddCommand(documentCopy(deps))
	document.AddCommand(documentMove(deps))
	document.AddCommand(documentDelete(deps))
	document.AddCommand(documentTrash(deps))
	document.AddCommand(documentRestore(deps))

	root.AddCommand(document)
}

func documentGet(deps Dependencies) *cobra.Command {
	var fields string
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a document by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := deps.Client.Get(context.Background(), fmt.Sprintf("/document/%s", args[0]), api.RequestOptions{Fields: fields})
			if err != nil {
				return err
			}
			return printResult(cmd, deps, result)
		},
	}
	cmd.Flags().StringVar(&fields, "fields", "", "Limit response fields")
	return cmd
}

func documentRoot(deps Dependencies) *cobra.Command {
	var fields string
	var paramsRaw string
	cmd := &cobra.Command{
		Use:   "root",
		Short: "Get root documents",
		RunE: func(cmd *cobra.Command, args []string) error {
			params, err := parseParams(paramsRaw)
			if err != nil {
				return err
			}
			result, err := deps.Client.Get(context.Background(), "/document/root", api.RequestOptions{Fields: fields, Params: params})
			if err != nil {
				return err
			}
			return printResult(cmd, deps, result)
		},
	}
	cmd.Flags().StringVar(&fields, "fields", "", "Limit response fields")
	cmd.Flags().StringVar(&paramsRaw, "params", "", "Query parameters as JSON")
	return cmd
}

func documentChildren(deps Dependencies) *cobra.Command {
	var fields string
	cmd := &cobra.Command{
		Use:   "children <id>",
		Short: "Get child documents",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := deps.Client.Get(context.Background(), fmt.Sprintf("/document/%s/children", args[0]), api.RequestOptions{Fields: fields})
			if err != nil {
				return err
			}
			return printResult(cmd, deps, result)
		},
	}
	cmd.Flags().StringVar(&fields, "fields", "", "Limit response fields")
	return cmd
}

func documentAncestors(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ancestors <id>",
		Short: "Get ancestor documents",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := deps.Client.Get(context.Background(), fmt.Sprintf("/document/%s/ancestors", args[0]), api.RequestOptions{})
			if err != nil {
				return err
			}
			return printResult(cmd, deps, result)
		},
	}
	return cmd
}

func documentSearch(deps Dependencies) *cobra.Command {
	var paramsRaw string
	var query string
	var skip int
	var take int

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search documents",
		RunE: func(cmd *cobra.Command, args []string) error {
			params, err := parseParams(paramsRaw)
			if err != nil {
				return err
			}
			if params == nil {
				params = map[string]any{}
				if query != "" {
					params["query"] = query
				}
				if skip >= 0 {
					params["skip"] = skip
				}
				if take >= 0 {
					params["take"] = take
				}
			}
			if len(params) == 0 {
				return fmt.Errorf("document search requires either --params or --query")
			}

			result, err := deps.Client.Get(context.Background(), "/document/search", api.RequestOptions{Params: params})
			if err != nil {
				return err
			}
			return printResult(cmd, deps, result)
		},
	}

	cmd.Flags().StringVar(&paramsRaw, "params", "", "Search parameters as JSON")
	cmd.Flags().StringVar(&query, "query", "", "Search query (convenience)")
	cmd.Flags().IntVar(&skip, "skip", -1, "Skip count")
	cmd.Flags().IntVar(&take, "take", -1, "Take count")
	return cmd
}

func documentCreate(deps Dependencies) *cobra.Command {
	var jsonPayload string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a document",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireValue("--json", jsonPayload); err != nil {
				return err
			}
			body, err := parsePayload(jsonPayload)
			if err != nil {
				return err
			}
			result, err := deps.Client.Post(context.Background(), "/document", body, api.RequestOptions{DryRun: dryRun})
			if err != nil {
				return err
			}
			return printResult(cmd, deps, result)
		},
	}
	cmd.Flags().StringVar(&jsonPayload, "json", "", "Full JSON payload")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate request without executing")
	return cmd
}

func documentUpdate(deps Dependencies) *cobra.Command {
	var jsonPayload string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireValue("--json", jsonPayload); err != nil {
				return err
			}
			body, err := parsePayload(jsonPayload)
			if err != nil {
				return err
			}
			result, err := deps.Client.Put(context.Background(), fmt.Sprintf("/document/%s", args[0]), body, api.RequestOptions{DryRun: dryRun})
			if err != nil {
				return err
			}
			return printResult(cmd, deps, result)
		},
	}
	cmd.Flags().StringVar(&jsonPayload, "json", "", "Update payload as JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate request without executing")
	return cmd
}

func documentUpdateProperties(deps Dependencies) *cobra.Command {
	var jsonPayload string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "update-properties <id>",
		Short: "Update document properties",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireValue("--json", jsonPayload); err != nil {
				return err
			}
			body, err := parsePayload(jsonPayload)
			if err != nil {
				return err
			}
			result, err := deps.Client.Put(context.Background(), fmt.Sprintf("/document/%s/properties", args[0]), body, api.RequestOptions{DryRun: dryRun})
			if err != nil {
				return err
			}
			return printResult(cmd, deps, result)
		},
	}
	cmd.Flags().StringVar(&jsonPayload, "json", "", "Properties payload as JSON")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate request without executing")
	return cmd
}

func documentPublish(deps Dependencies) *cobra.Command {
	var jsonPayload string
	var culture string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "publish <id>",
		Short: "Publish a document",
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
			result, err := deps.Client.Post(context.Background(), fmt.Sprintf("/document/%s/publish", args[0]), body, api.RequestOptions{DryRun: dryRun})
			if err != nil {
				return err
			}
			return printResult(cmd, deps, result)
		},
	}
	cmd.Flags().StringVar(&jsonPayload, "json", "", "Publish payload as JSON")
	cmd.Flags().StringVar(&culture, "culture", "", "Culture shortcut")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate request without executing")
	return cmd
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
			result, err := deps.Client.Post(context.Background(), fmt.Sprintf("/document/%s/unpublish", args[0]), body, api.RequestOptions{DryRun: dryRun})
			if err != nil {
				return err
			}
			return printResult(cmd, deps, result)
		},
	}
	cmd.Flags().StringVar(&jsonPayload, "json", "", "Unpublish payload as JSON")
	cmd.Flags().StringVar(&culture, "culture", "", "Culture shortcut")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate request without executing")
	return cmd
}

func documentCopy(deps Dependencies) *cobra.Command {
	var jsonPayload string
	var to string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "copy <id>",
		Short: "Copy a document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var body map[string]any
			var err error
			if jsonPayload != "" {
				body, err = parsePayload(jsonPayload)
			} else {
				if err := requireValue("--to", to); err != nil {
					return err
				}
				body = map[string]any{"target": map[string]any{"id": to}}
			}
			if err != nil {
				return err
			}
			result, err := deps.Client.Post(context.Background(), fmt.Sprintf("/document/%s/copy", args[0]), body, api.RequestOptions{DryRun: dryRun})
			if err != nil {
				return err
			}
			return printResult(cmd, deps, result)
		},
	}
	cmd.Flags().StringVar(&jsonPayload, "json", "", "Copy payload as JSON")
	cmd.Flags().StringVar(&to, "to", "", "Target parent ID shortcut")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate request without executing")
	return cmd
}

func documentMove(deps Dependencies) *cobra.Command {
	var jsonPayload string
	var to string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "move <id>",
		Short: "Move a document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var body map[string]any
			var err error
			if jsonPayload != "" {
				body, err = parsePayload(jsonPayload)
			} else {
				if err := requireValue("--to", to); err != nil {
					return err
				}
				body = map[string]any{"target": map[string]any{"id": to}}
			}
			if err != nil {
				return err
			}
			result, err := deps.Client.Post(context.Background(), fmt.Sprintf("/document/%s/move", args[0]), body, api.RequestOptions{DryRun: dryRun})
			if err != nil {
				return err
			}
			return printResult(cmd, deps, result)
		},
	}
	cmd.Flags().StringVar(&jsonPayload, "json", "", "Move payload as JSON")
	cmd.Flags().StringVar(&to, "to", "", "Target parent ID shortcut")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate request without executing")
	return cmd
}

func documentDelete(deps Dependencies) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := deps.Client.Delete(context.Background(), fmt.Sprintf("/document/%s", args[0]), api.RequestOptions{DryRun: dryRun})
			if err != nil {
				return err
			}
			return printResult(cmd, deps, result)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate request without executing")
	return cmd
}

func documentTrash(deps Dependencies) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "trash <id>",
		Short: "Move a document to recycle bin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := deps.Client.Post(context.Background(), fmt.Sprintf("/document/%s/move-to-recycle-bin", args[0]), map[string]any{}, api.RequestOptions{DryRun: dryRun})
			if err != nil {
				return err
			}
			return printResult(cmd, deps, result)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate request without executing")
	return cmd
}

func documentRestore(deps Dependencies) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "restore <id>",
		Short: "Restore a document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := deps.Client.Post(context.Background(), fmt.Sprintf("/document/%s/restore", args[0]), map[string]any{}, api.RequestOptions{DryRun: dryRun})
			if err != nil {
				return err
			}
			return printResult(cmd, deps, result)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate request without executing")
	return cmd
}
