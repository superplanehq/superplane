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

	var getDraft bool
	getCmd := &cobra.Command{
		Use:   "get [app-name-or-id]",
		Short: "Get the console for an app",
		Long: `Print the console for an app. With -o yaml, prints the canonical
Console YAML (apiVersion: v1, kind: Console). Defaults to the live console;
use --draft to read your in-progress draft.

The app argument is optional. When omitted, the active app
configured with "superplane apps active" is used.`,
		Args: cobra.MaximumNArgs(1),
	}
	getCmd.Flags().BoolVar(&getDraft, "draft", false, "read the current user's draft console instead of the live console")
	core.Bind(getCmd, &getCommand{draft: &getDraft}, options)

	var setFile string
	var setDraftOnly bool
	setCmd := &cobra.Command{
		Use:   "set [app-name-or-id] [file]",
		Short: "Replace the console draft with YAML",
		Long: `Replace the console panels and layout in the current user's draft
version. The YAML must use apiVersion: v1 and kind: Console.

When change management is enabled for the app, an open change request
for the updated draft is created automatically so it shows up in the UI
for review. Pass --draft to skip change-request creation and leave the
edit as a draft only.

The app argument is optional. When omitted, the active app
configured with "superplane apps active" is used.

YAML source resolution order:
  1. --file <path>   (use "-" for stdin)
  2. positional file argument (only when an app argument is also given)
  3. piped stdin (when "-" is given)`,
		Args: cobra.MaximumNArgs(2),
	}
	setCmd.Flags().StringVarP(&setFile, "file", "f", "", `console YAML file path, or "-" for stdin`)
	setCmd.Flags().BoolVar(&setDraftOnly, "draft", false, "update the draft only; do not create a change request when change management is enabled")
	core.Bind(setCmd, &setCommand{file: &setFile, draftOnly: &setDraftOnly}, options)

	root.AddCommand(getCmd)
	root.AddCommand(setCmd)

	return root
}
