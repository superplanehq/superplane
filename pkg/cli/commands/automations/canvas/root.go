package canvas

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func Commands(options core.BindOptions) []*cobra.Command {
	getCmd := &cobra.Command{
		Use:    "get [name-or-id]",
		Short:  "Get an automation definition",
		Hidden: true,
		Long: `Print the live automation definition. With -o yaml, prints the canonical
definition (apiVersion: v1, kind: Canvas).

The automation argument is optional. When omitted, the active automation
configured with "superplane automations active" is used.`,
		Args: cobra.MaximumNArgs(1),
	}
	core.Bind(getCmd, &getCommand{}, options)

	var updateFile string
	var updateMessage string
	var updateAutoLayout string
	var updateAutoLayoutScope string
	var updateAutoLayoutNodes []string
	updateCmd := &cobra.Command{
		Use:    "update [name-or-id]",
		Short:  "Update an automation from a YAML file",
		Hidden: true,
		Long: `Update the live automation from --file and commit immediately with --message.

The automation argument is optional. When omitted, the active automation configured with
"superplane automations active" is used. To stage changes without committing, use
"superplane automations staging update" and "staging commit".`,
		Args: cobra.MaximumNArgs(1),
	}
	updateCmd.Flags().StringVarP(&updateFile, "file", "f", "", "filename, directory, or URL to files to use to update the resource")
	_ = updateCmd.MarkFlagRequired("file")
	updateCmd.Flags().StringVarP(&updateMessage, "message", "m", "", "commit message")
	_ = updateCmd.MarkFlagRequired("message")
	updateCmd.Flags().StringVar(&updateAutoLayout, "auto-layout", "", "automatically arrange the automation (supported: horizontal, vertical, disable)")
	updateCmd.Flags().StringVar(&updateAutoLayoutScope, "auto-layout-scope", "", "scope for auto layout (full-canvas, connected-component)")
	updateCmd.Flags().StringArrayVar(&updateAutoLayoutNodes, "auto-layout-node", nil, "node id seed for auto layout (repeatable)")
	core.Bind(updateCmd, &updateCommand{
		file:            &updateFile,
		message:         &updateMessage,
		autoLayout:      &updateAutoLayout,
		autoLayoutScope: &updateAutoLayoutScope,
		autoLayoutNodes: &updateAutoLayoutNodes,
	}, options)

	var initTemplate string
	var initListTemplates bool
	var initOutputFile string
	initCmd := &cobra.Command{
		Use:    "init",
		Short:  "Generate a starter automation YAML definition",
		Hidden: true,
		Long: `Print a starter automation YAML definition to stdout.

By default, prints a blank automation. Use --template to start from a built-in
example (currently: health-check-monitor), or --list-templates to see options.

The file uses apiVersion: v1 and kind: Canvas.`,
		Args: cobra.NoArgs,
	}
	initCmd.Flags().StringVar(&initTemplate, "template", "", "start from a built-in template (e.g. health-check-monitor)")
	initCmd.Flags().BoolVar(&initListTemplates, "list-templates", false, "list available built-in templates")
	initCmd.Flags().StringVar(&initOutputFile, "output-file", "", "write to a file instead of stdout")
	core.Bind(initCmd, &initCommand{
		template:      &initTemplate,
		listTemplates: &initListTemplates,
		outputFile:    &initOutputFile,
	}, options)

	return []*cobra.Command{getCmd, initCmd, updateCmd}
}
