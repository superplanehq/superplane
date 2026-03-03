package canvases

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func buildVersionsCommandGroup(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "versions",
		Short: "Manage canvas versions",
	}

	listCmd := &cobra.Command{
		Use:   "list [canvas-id-or-name]",
		Short: "List visible versions for a canvas",
		Args:  cobra.MaximumNArgs(1),
	}
	core.Bind(listCmd, &versionsListCommand{}, options)

	createCmd := &cobra.Command{
		Use:   "create [canvas-id-or-name]",
		Short: "Create a working version from live",
		Args:  cobra.MaximumNArgs(1),
	}
	core.Bind(createCmd, &versionsCreateCommand{}, options)

	var useCanvas string
	useCmd := &cobra.Command{
		Use:   "use <version-id|live>",
		Short: "Switch active version for local CLI context",
		Args:  cobra.ExactArgs(1),
	}
	useCmd.Flags().StringVar(&useCanvas, "canvas", "", "canvas id or name (defaults to active canvas)")
	core.Bind(useCmd, &versionsUseCommand{canvas: &useCanvas}, options)

	var updateCanvas string
	var updateFile string
	var updateAutoLayout string
	var updateAutoLayoutScope string
	var updateAutoLayoutNodes []string
	updateCmd := &cobra.Command{
		Use:   "update [version-id]",
		Short: "Update a working version from a canvas file",
		Args:  cobra.MaximumNArgs(1),
	}
	updateCmd.Flags().StringVar(&updateCanvas, "canvas", "", "canvas id or name (defaults to active canvas)")
	updateCmd.Flags().StringVarP(&updateFile, "file", "f", "", "canvas yaml file to use as version content")
	updateCmd.Flags().StringVar(&updateAutoLayout, "auto-layout", "", "automatically arrange the canvas (supported: horizontal)")
	updateCmd.Flags().StringVar(&updateAutoLayoutScope, "auto-layout-scope", "", "scope for auto layout (full-canvas, connected-component, exact-set)")
	updateCmd.Flags().StringArrayVar(&updateAutoLayoutNodes, "auto-layout-node", nil, "node id seed for auto layout (repeatable)")
	core.Bind(updateCmd, &versionsUpdateCommand{
		canvas:          &updateCanvas,
		file:            &updateFile,
		autoLayout:      &updateAutoLayout,
		autoLayoutScope: &updateAutoLayoutScope,
		autoLayoutNodes: &updateAutoLayoutNodes,
	}, options)

	var publishCanvas string
	var publishExpectedLive string
	publishCmd := &cobra.Command{
		Use:   "publish [version-id]",
		Short: "Publish a working version",
		Args:  cobra.MaximumNArgs(1),
	}
	publishCmd.Flags().StringVar(&publishCanvas, "canvas", "", "canvas id or name (defaults to active canvas)")
	publishCmd.Flags().StringVar(
		&publishExpectedLive,
		"expect-live-version-id",
		"",
		"expected live version id (set to \"auto\" to use current live version)",
	)
	core.Bind(publishCmd, &versionsPublishCommand{
		canvas:              &publishCanvas,
		expectedLiveVersion: &publishExpectedLive,
	}, options)

	var discardCanvas string
	discardCmd := &cobra.Command{
		Use:   "discard [version-id]",
		Short: "Discard a working version",
		Args:  cobra.MaximumNArgs(1),
	}
	discardCmd.Flags().StringVar(&discardCanvas, "canvas", "", "canvas id or name (defaults to active canvas)")
	core.Bind(discardCmd, &versionsDiscardCommand{canvas: &discardCanvas}, options)

	root.AddCommand(listCmd)
	root.AddCommand(createCmd)
	root.AddCommand(useCmd)
	root.AddCommand(updateCmd)
	root.AddCommand(publishCmd)
	root.AddCommand(discardCmd)

	return root
}
