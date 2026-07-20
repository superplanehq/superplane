package canvas

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "canvas",
		Short: "Manage app canvas",
		Long: `Manage the canvas for an app: nodes, edges, triggers, and actions.

Canvas YAML uses apiVersion: v1 and kind: Canvas. For canonical shapes and wiring
rules, install skills:
- ` + core.SkillsInstallCommand("superplane-app-builder") + `
- ` + core.SkillsInstallCommand("superplane-cli"),
	}

	getCmd := &cobra.Command{
		Use:   "get [name-or-id]",
		Short: "Get a canvas",
		Long: `Print the live canvas for an app. With -o yaml, prints the canonical
Canvas YAML (apiVersion: v1, kind: Canvas).

The app argument is optional. When omitted, the active app
configured with "superplane apps active" is used.`,
		Args: cobra.MaximumNArgs(1),
	}
	core.Bind(getCmd, &getCommand{}, options)

	var updateFile string
	var updateMessage string
	var updateAutoLayout string
	var updateAutoLayoutScope string
	var updateAutoLayoutNodes []string
	updateCmd := &cobra.Command{
		Use:   "update [name-or-id]",
		Short: "Update a canvas from a YAML file",
		Long: `Update the live canvas from --file and commit immediately with --message.

The app argument is optional. When omitted, the active app configured with
"superplane apps active" is used. To stage changes without committing, use
"superplane apps staging update" and "staging commit".`,
		Args: cobra.MaximumNArgs(1),
	}
	updateCmd.Flags().StringVarP(&updateFile, "file", "f", "", "filename, directory, or URL to files to use to update the resource")
	_ = updateCmd.MarkFlagRequired("file")
	updateCmd.Flags().StringVarP(&updateMessage, "message", "m", "", "commit message")
	_ = updateCmd.MarkFlagRequired("message")
	updateCmd.Flags().StringVar(&updateAutoLayout, "auto-layout", "", "automatically arrange the canvas (supported: horizontal, disable)")
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
		Use:   "init",
		Short: "Generate a starter canvas YAML definition",
		Long: `Print a starter canvas YAML definition to stdout.

By default, prints a blank canvas. Use --template to start from a built-in
example (currently: health-check-monitor), or --list-templates to see options.`,
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

	root.AddCommand(getCmd)
	root.AddCommand(initCmd)
	root.AddCommand(updateCmd)

	return root
}
