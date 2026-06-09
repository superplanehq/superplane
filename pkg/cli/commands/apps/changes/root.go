package changes

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	var listStatusFilter string
	var listOnlyMine bool
	var listQuery string
	var listLimit int64
	var listBefore string

	root := &cobra.Command{
		Use:     "change-requests",
		Short:   "Manage app change requests",
		Aliases: []string{"cr"},
	}

	listCmd := &cobra.Command{
		Use:   "list [name-or-id]",
		Short: "List change requests for an app",
		Args:  cobra.MaximumNArgs(1),
	}

	listCmd.Flags().StringVar(&listStatusFilter, "status", "", "status filter: all, open, conflicted, rejected, published")
	listCmd.Flags().BoolVar(&listOnlyMine, "mine", false, "list only change requests created by the current user")
	listCmd.Flags().StringVar(&listQuery, "query", "", "search by title or description")
	listCmd.Flags().Int64Var(&listLimit, "limit", 50, "maximum number of change requests to return")
	listCmd.Flags().StringVar(&listBefore, "before", "", "return change requests created before an RFC3339 timestamp")

	core.Bind(listCmd, &ListCommand{
		statusFilter: &listStatusFilter,
		onlyMine:     &listOnlyMine,
		query:        &listQuery,
		limit:        &listLimit,
		before:       &listBefore,
	}, options)

	getCmd := &cobra.Command{
		Use:   "get <change-request-id> [name-or-id]",
		Short: "Describe a change request",
		Args:  cobra.RangeArgs(1, 2),
	}

	core.Bind(getCmd, &GetCommand{}, options)

	var createVersionID string
	var createDraftID string
	var createTitle string
	var createDescription string

	createCmd := &cobra.Command{
		Use:   "create [name-or-id]",
		Short: "Create a change request",
		Args:  cobra.MaximumNArgs(1),
	}

	createCmd.Flags().StringVar(&createVersionID, "version-id", "", "version id to use (defaults to current user draft)")
	createCmd.Flags().StringVar(&createDraftID, "draft-id", "", "alias for --version-id")
	createCmd.Flags().StringVar(&createTitle, "title", "", "change request title")
	createCmd.Flags().StringVar(&createDescription, "description", "", "change request description")

	core.Bind(createCmd, &CreateCommand{
		versionID:   &createVersionID,
		draftID:     &createDraftID,
		title:       &createTitle,
		description: &createDescription,
	}, options)

	approveCmd := &cobra.Command{
		Use:   "approve <change-request-id> [name-or-id]",
		Short: "Approve a change request",
		Args:  cobra.RangeArgs(1, 2),
	}
	core.Bind(approveCmd, &ActionCommand{
		action: openapi_client.CANVASESACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_APPROVE,
	}, options)

	unapproveCmd := &cobra.Command{
		Use:   "unapprove <change-request-id> [name-or-id]",
		Short: "Remove your active approval from a change request",
		Args:  cobra.RangeArgs(1, 2),
	}

	core.Bind(unapproveCmd, &ActionCommand{
		action: openapi_client.CANVASESACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_UNAPPROVE,
	}, options)

	rejectCmd := &cobra.Command{
		Use:   "reject <change-request-id> [name-or-id]",
		Short: "Reject an open change request",
		Args:  cobra.RangeArgs(1, 2),
	}
	core.Bind(rejectCmd, &ActionCommand{
		action: openapi_client.CANVASESACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_REJECT,
	}, options)

	reopenCmd := &cobra.Command{
		Use:   "reopen <change-request-id> [name-or-id]",
		Short: "Reopen a rejected change request",
		Args:  cobra.RangeArgs(1, 2),
	}
	core.Bind(reopenCmd, &ActionCommand{
		action: openapi_client.CANVASESACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_REOPEN,
	}, options)

	publishCmd := &cobra.Command{
		Use:   "publish <change-request-id> [name-or-id]",
		Short: "Publish an approved change request",
		Args:  cobra.RangeArgs(1, 2),
	}
	core.Bind(publishCmd, &ActionCommand{
		action: openapi_client.CANVASESACTONCANVASCHANGEREQUESTREQUESTACTION_ACTION_PUBLISH,
	}, options)

	var resolveFile string
	var resolveAutoLayout string
	var resolveAutoLayoutScope string
	var resolveAutoLayoutNodes []string
	resolveCmd := &cobra.Command{
		Use:   "resolve <change-request-id> [name-or-id]",
		Short: "Resolve conflicts by updating the change request version",
		Args:  cobra.RangeArgs(1, 2),
	}
	resolveCmd.Flags().StringVarP(&resolveFile, "file", "f", "", "canvas file containing the conflict-resolved version")
	resolveCmd.Flags().StringVar(&resolveAutoLayout, "auto-layout", "", "automatically arrange the canvas (supported: horizontal, disable)")
	resolveCmd.Flags().StringVar(&resolveAutoLayoutScope, "auto-layout-scope", "", "scope for auto layout (full-canvas, connected-component)")
	resolveCmd.Flags().StringArrayVar(&resolveAutoLayoutNodes, "auto-layout-node", nil, "node id seed for auto layout (repeatable)")
	core.Bind(resolveCmd, &ResolveCommand{
		file:            &resolveFile,
		autoLayout:      &resolveAutoLayout,
		autoLayoutScope: &resolveAutoLayoutScope,
		autoLayoutNodes: &resolveAutoLayoutNodes,
	}, options)

	root.AddCommand(listCmd)
	root.AddCommand(getCmd)
	root.AddCommand(createCmd)
	root.AddCommand(approveCmd)
	root.AddCommand(unapproveCmd)
	root.AddCommand(rejectCmd)
	root.AddCommand(reopenCmd)
	root.AddCommand(publishCmd)
	root.AddCommand(resolveCmd)

	return root
}
