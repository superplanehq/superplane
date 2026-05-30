package canvas

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "canvas",
		Short: "Manage app canvas (workflow graph)",
		Long: `Manage the workflow canvas for an app: nodes, edges, triggers, and actions.

Canvas YAML uses apiVersion: v1 and kind: Canvas. For canonical shapes and wiring
rules, install skills:
- ` + core.SkillsInstallCommand("superplane-app-builder") + `
- ` + core.SkillsInstallCommand("superplane-cli"),
	}

	getCmd := &cobra.Command{
		Use:   "get <name-or-id>",
		Short: "Get a canvas",
		Args:  cobra.ExactArgs(1),
	}
	var getDraft bool
	getCmd.Flags().BoolVar(&getDraft, "draft", false, "get your draft version instead of the live version")
	core.Bind(getCmd, &getCommand{draft: &getDraft}, options)

	var updateFile string
	var updateDraft bool
	var updateAutoLayout string
	var updateAutoLayoutScope string
	var updateAutoLayoutNodes []string
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a canvas from a YAML file",
		Long:  "Updates the canvas using --file. The file must include metadata.id to identify the target canvas.",
		Args:  cobra.NoArgs,
	}
	updateCmd.Flags().StringVarP(&updateFile, "file", "f", "", "filename, directory, or URL to files to use to update the resource")
	_ = updateCmd.MarkFlagRequired("file")
	updateCmd.Flags().BoolVar(&updateDraft, "draft", false, "keep the update as a draft instead of auto-publishing (required when change management is enabled)")
	updateCmd.Flags().StringVar(&updateAutoLayout, "auto-layout", "", "automatically arrange the canvas (supported: horizontal, disable)")
	updateCmd.Flags().StringVar(&updateAutoLayoutScope, "auto-layout-scope", "", "scope for auto layout (full-canvas, connected-component)")
	updateCmd.Flags().StringArrayVar(&updateAutoLayoutNodes, "auto-layout-node", nil, "node id seed for auto layout (repeatable)")
	core.Bind(updateCmd, &updateCommand{
		file:            &updateFile,
		draft:           &updateDraft,
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
		Long:  "Print a starter canvas YAML definition to stdout. Use --template to start from an existing template, or --list-templates to see available options.",
		Args:  cobra.NoArgs,
	}
	initCmd.Flags().StringVar(&initTemplate, "template", "", "start from a named template (e.g. health-check-monitor)")
	initCmd.Flags().BoolVar(&initListTemplates, "list-templates", false, "list available template names")
	initCmd.Flags().StringVar(&initOutputFile, "output-file", "", "write to a file instead of stdout")
	core.Bind(initCmd, &initCommand{
		template:      &initTemplate,
		listTemplates: &initListTemplates,
		outputFile:    &initOutputFile,
	}, options)

	deleteCmd := &cobra.Command{
		Use:   "delete <name-or-id>",
		Short: "Delete a canvas",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(deleteCmd, &deleteCommand{}, options)

	root.AddCommand(getCmd)
	root.AddCommand(initCmd)
	root.AddCommand(updateCmd)
	root.AddCommand(deleteCmd)

	return root
}
