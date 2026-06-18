package console

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "console",
		Short: "Manage an app console",
		Long: `Manage the per-app console: panels and grid layout configured for an
app. The console is stored on app versions, so reads default to the
live console and writes always target your draft version.`,
	}

	var getDraftID string
	getCmd := &cobra.Command{
		Use:   "get [app-name-or-id]",
		Short: "Get the console for an app",
		Long: `Print the console for an app. With -o yaml, prints the canonical
Console YAML (apiVersion: v1, kind: Console). Defaults to the live console;
pass --draft-id to read a specific draft.

The app argument is optional. When omitted, the active app
configured with "superplane apps active" is used.`,
		Args: cobra.MaximumNArgs(1),
	}
	getCmd.Flags().StringVar(&getDraftID, "draft-id", "", "target a specific draft by id (see `superplane apps drafts list`)")
	core.Bind(getCmd, &getCommand{draftID: &getDraftID}, options)

	var setFile string
	var setDraftID string
	setCmd := &cobra.Command{
		Use:   "set [app-name-or-id] [file]",
		Short: "Replace the console draft with YAML",
		Long: `Replace the console panels and layout in the current user's draft
version. The YAML must use apiVersion: v1 and kind: Console.

Pass --draft-id to target a specific draft and leave the edit as a draft only.

The app argument is optional. When omitted, the active app
configured with "superplane apps active" is used.

YAML source resolution order:
  1. --file <path>   (use "-" for stdin)
  2. positional file argument (only when an app argument is also given)
  3. piped stdin (when "-" is given)`,
		Args: cobra.MaximumNArgs(2),
	}
	setCmd.Flags().StringVarP(&setFile, "file", "f", "", `console YAML file path, or "-" for stdin`)
	setCmd.Flags().StringVar(&setDraftID, "draft-id", "", "target a specific draft by id; skips change-request creation (see `superplane apps drafts list`)")
	core.Bind(setCmd, &setCommand{file: &setFile, draftID: &setDraftID}, options)

	root.AddCommand(getCmd)
	root.AddCommand(setCmd)

	return root
}
