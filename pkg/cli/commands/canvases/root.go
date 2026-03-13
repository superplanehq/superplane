package canvases

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "canvases",
		Short:   "Manage canvases",
		Aliases: []string{"canvas"},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List canvases",
		Args:  cobra.NoArgs,
	}
	core.Bind(listCmd, &listCommand{}, options)

	getCmd := &cobra.Command{
		Use:   "get <name-or-id>",
		Short: "Get a canvas",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(getCmd, &getCommand{}, options)

	activeCmd := &cobra.Command{
		Use:   "active [canvas-id]",
		Short: "Set the active canvas",
		Long:  "Without arguments, prompts for a canvas selection. With a canvas ID, sets it directly.",
		Args:  cobra.MaximumNArgs(1),
	}
	core.Bind(activeCmd, &ActiveCommand{}, options)

	var createFile string
	createCmd := &cobra.Command{
		Use:   "create [canvas-name]",
		Short: "Create a canvas",
		Args:  cobra.MaximumNArgs(1),
	}
	createCmd.Flags().StringVarP(&createFile, "file", "f", "", "filename, directory, or URL to files to use to create the resource")
	core.Bind(createCmd, &createCommand{file: &createFile}, options)

	var updateFile string
	var updateDraft bool
	var updateAutoLayout string
	var updateAutoLayoutScope string
	var updateAutoLayoutNodes []string
	updateCmd := &cobra.Command{
		Use:   "update [name-or-id]",
		Short: "Update a canvas from a file",
		Args:  cobra.MaximumNArgs(1),
	}
	updateCmd.Flags().StringVarP(&updateFile, "file", "f", "", "filename, directory, or URL to files to use to update the resource")
	updateCmd.Flags().BoolVar(&updateDraft, "draft", false, "update your draft version (required when effective canvas versioning is enabled)")
	updateCmd.Flags().StringVar(&updateAutoLayout, "auto-layout", "", "automatically arrange the canvas (supported: horizontal)")
	updateCmd.Flags().StringVar(&updateAutoLayoutScope, "auto-layout-scope", "", "scope for auto layout (full-canvas, connected-component)")
	updateCmd.Flags().StringArrayVar(&updateAutoLayoutNodes, "auto-layout-node", nil, "node id seed for auto layout (repeatable)")
	core.Bind(updateCmd, &updateCommand{
		file:            &updateFile,
		draft:           &updateDraft,
		autoLayout:      &updateAutoLayout,
		autoLayoutScope: &updateAutoLayoutScope,
		autoLayoutNodes: &updateAutoLayoutNodes,
	}, options)

	var changeRequestsListStatusFilter string
	var changeRequestsListOnlyMine bool
	var changeRequestsListQuery string
	var changeRequestsListLimit int64
	var changeRequestsListBefore string

	changeRequestsCmd := &cobra.Command{
		Use:     "change-requests",
		Short:   "Manage canvas change requests",
		Aliases: []string{"cr"},
	}

	changeRequestsListCmd := &cobra.Command{
		Use:   "list [name-or-id]",
		Short: "List change requests for a canvas",
		Args:  cobra.MaximumNArgs(1),
	}
	changeRequestsListCmd.Flags().StringVar(&changeRequestsListStatusFilter, "status", "", "status filter: all, open, conflicted, rejected, published")
	changeRequestsListCmd.Flags().BoolVar(&changeRequestsListOnlyMine, "mine", false, "list only change requests created by the current user")
	changeRequestsListCmd.Flags().StringVar(&changeRequestsListQuery, "query", "", "search by title or description")
	changeRequestsListCmd.Flags().Int64Var(&changeRequestsListLimit, "limit", 50, "maximum number of change requests to return")
	changeRequestsListCmd.Flags().StringVar(&changeRequestsListBefore, "before", "", "return change requests created before an RFC3339 timestamp")
	core.Bind(changeRequestsListCmd, &changeRequestListCommand{
		statusFilter: &changeRequestsListStatusFilter,
		onlyMine:     &changeRequestsListOnlyMine,
		query:        &changeRequestsListQuery,
		limit:        &changeRequestsListLimit,
		before:       &changeRequestsListBefore,
	}, options)

	changeRequestsGetCmd := &cobra.Command{
		Use:   "get <change-request-id> [name-or-id]",
		Short: "Describe a change request",
		Args:  cobra.RangeArgs(1, 2),
	}
	core.Bind(changeRequestsGetCmd, &changeRequestGetCommand{}, options)

	var changeRequestsCreateVersionID string
	var changeRequestsCreateTitle string
	var changeRequestsCreateDescription string
	changeRequestsCreateCmd := &cobra.Command{
		Use:   "create [name-or-id]",
		Short: "Create a change request",
		Args:  cobra.MaximumNArgs(1),
	}
	changeRequestsCreateCmd.Flags().StringVar(&changeRequestsCreateVersionID, "version-id", "", "version id to use (defaults to current user draft)")
	changeRequestsCreateCmd.Flags().StringVar(&changeRequestsCreateTitle, "title", "", "change request title")
	changeRequestsCreateCmd.Flags().StringVar(&changeRequestsCreateDescription, "description", "", "change request description")
	core.Bind(changeRequestsCreateCmd, &changeRequestCreateCommand{
		versionID:   &changeRequestsCreateVersionID,
		title:       &changeRequestsCreateTitle,
		description: &changeRequestsCreateDescription,
	}, options)

	changeRequestsApproveCmd := &cobra.Command{
		Use:   "approve <change-request-id> [name-or-id]",
		Short: "Approve a change request",
		Args:  cobra.RangeArgs(1, 2),
	}
	core.Bind(changeRequestsApproveCmd, &changeRequestActionCommand{
		action: openapi_client.ACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_APPROVE,
	}, options)

	changeRequestsUnapproveCmd := &cobra.Command{
		Use:   "unapprove <change-request-id> [name-or-id]",
		Short: "Remove your active approval from a change request",
		Args:  cobra.RangeArgs(1, 2),
	}
	core.Bind(changeRequestsUnapproveCmd, &changeRequestActionCommand{
		action: openapi_client.ACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_UNAPPROVE,
	}, options)

	changeRequestsRejectCmd := &cobra.Command{
		Use:   "reject <change-request-id> [name-or-id]",
		Short: "Reject an open change request",
		Args:  cobra.RangeArgs(1, 2),
	}
	core.Bind(changeRequestsRejectCmd, &changeRequestActionCommand{
		action: openapi_client.ACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_REJECT,
	}, options)

	changeRequestsReopenCmd := &cobra.Command{
		Use:   "reopen <change-request-id> [name-or-id]",
		Short: "Reopen a rejected change request",
		Args:  cobra.RangeArgs(1, 2),
	}
	core.Bind(changeRequestsReopenCmd, &changeRequestActionCommand{
		action: openapi_client.ACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_REOPEN,
	}, options)

	changeRequestsPublishCmd := &cobra.Command{
		Use:   "publish <change-request-id> [name-or-id]",
		Short: "Publish an approved change request",
		Args:  cobra.RangeArgs(1, 2),
	}
	core.Bind(changeRequestsPublishCmd, &changeRequestActionCommand{
		action: openapi_client.ACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_PUBLISH,
	}, options)

	var changeRequestsResolveFile string
	var changeRequestsResolveAutoLayout string
	var changeRequestsResolveAutoLayoutScope string
	var changeRequestsResolveAutoLayoutNodes []string
	changeRequestsResolveCmd := &cobra.Command{
		Use:   "resolve <change-request-id> [name-or-id]",
		Short: "Resolve conflicts by updating the change request version",
		Args:  cobra.RangeArgs(1, 2),
	}
	changeRequestsResolveCmd.Flags().StringVarP(&changeRequestsResolveFile, "file", "f", "", "canvas file containing the conflict-resolved version")
	changeRequestsResolveCmd.Flags().StringVar(&changeRequestsResolveAutoLayout, "auto-layout", "", "automatically arrange the canvas (supported: horizontal)")
	changeRequestsResolveCmd.Flags().StringVar(&changeRequestsResolveAutoLayoutScope, "auto-layout-scope", "", "scope for auto layout (full-canvas, connected-component)")
	changeRequestsResolveCmd.Flags().StringArrayVar(&changeRequestsResolveAutoLayoutNodes, "auto-layout-node", nil, "node id seed for auto layout (repeatable)")
	core.Bind(changeRequestsResolveCmd, &changeRequestResolveCommand{
		file:            &changeRequestsResolveFile,
		autoLayout:      &changeRequestsResolveAutoLayout,
		autoLayoutScope: &changeRequestsResolveAutoLayoutScope,
		autoLayoutNodes: &changeRequestsResolveAutoLayoutNodes,
	}, options)

	changeRequestsCmd.AddCommand(changeRequestsListCmd)
	changeRequestsCmd.AddCommand(changeRequestsGetCmd)
	changeRequestsCmd.AddCommand(changeRequestsCreateCmd)
	changeRequestsCmd.AddCommand(changeRequestsApproveCmd)
	changeRequestsCmd.AddCommand(changeRequestsUnapproveCmd)
	changeRequestsCmd.AddCommand(changeRequestsRejectCmd)
	changeRequestsCmd.AddCommand(changeRequestsReopenCmd)
	changeRequestsCmd.AddCommand(changeRequestsPublishCmd)
	changeRequestsCmd.AddCommand(changeRequestsResolveCmd)

	root.AddCommand(listCmd)
	root.AddCommand(getCmd)
	root.AddCommand(activeCmd)
	root.AddCommand(createCmd)
	root.AddCommand(updateCmd)
	root.AddCommand(changeRequestsCmd)

	return root
}
