package agent

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "agent [prompt...]",
		Short:   "Chat with the SuperPlane agent",
		Aliases: []string{"agents"},
		Args:    cobra.ArbitraryArgs,
	}

	var canvasID string
	root.PersistentFlags().StringVar(&canvasID, "canvas-id", "", "canvas id to use (defaults to the active canvas)")

	core.Bind(root, &NewChatCommand{CanvasID: &canvasID}, options)

	return root
}
