package console

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "console",
		Short: "Manage an app console",
		Long: `Manage the per-app console: panels and grid layout configured for an app.

Reads return the live console. Writes commit immediately with --message, or you
can stage changes with "superplane apps staging update" and "staging commit".`,
	}

	getCmd := &cobra.Command{
		Use:   "get [app-name-or-id]",
		Short: "Get the console for an app",
		Long: `Print the live console for an app. With -o yaml, prints the canonical
Console YAML (apiVersion: v1, kind: Console).

The app argument is optional. When omitted, the active app
configured with "superplane apps active" is used.`,
		Args: cobra.MaximumNArgs(1),
	}
	core.Bind(getCmd, &getCommand{}, options)

	var setFile string
	var setMessage string
	setCmd := &cobra.Command{
		Use:   "set [app-name-or-id] [file]",
		Short: "Replace the live console with YAML",
		Long: `Replace the live console panels and layout from YAML and commit with --message.

The YAML must use apiVersion: v1 and kind: Console. To stage changes without
committing, use "superplane apps staging update" and "staging commit".

The app argument is optional. When omitted, the active app
configured with "superplane apps active" is used.

YAML source resolution order:
  1. --file <path>   (use "-" for stdin)
  2. positional file argument (only when an app argument is also given)
  3. piped stdin (when "-" is given)`,
		Args: cobra.MaximumNArgs(2),
	}
	setCmd.Flags().StringVarP(&setFile, "file", "f", "", `console YAML file path, or "-" for stdin`)
	setCmd.Flags().StringVarP(&setMessage, "message", "m", "", "commit message")
	_ = setCmd.MarkFlagRequired("message")
	core.Bind(setCmd, &setCommand{file: &setFile, message: &setMessage}, options)

	root.AddCommand(getCmd)
	root.AddCommand(setCmd)

	return root
}
