package extensions

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:     "extensions",
		Short:   "Manage extensions",
		Aliases: []string{"extension"},
	}

	//
	// superplane extensions list
	//
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List extensions",
		Args:  cobra.NoArgs,
	}
	core.Bind(listCmd, &ListExtensionsCommand{}, options)

	//
	// superplane extensions list-versions
	//
	listVersionsCmd := &cobra.Command{
		Use:   "list-versions",
		Short: "List versions of an extension",
		Args:  cobra.NoArgs,
	}

	var extensionID string
	listVersionsCmd.Flags().StringVarP(&extensionID, "extension-id", "e", "", "extension ID")
	listVersionsCmd.MarkFlagRequired("extension-id")
	listVersionsCmd.RunE = bindExtensionsCommand(options, func() core.Command {
		return &ListVersionsCommand{
			ExtensionID: extensionID,
		}
	})

	//
	// superplane extensions create
	//
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an extension",
		Args:  cobra.NoArgs,
	}

	var name string
	var description string
	createCmd.Flags().StringVarP(&name, "name", "n", "", "extension name")
	createCmd.Flags().StringVarP(&description, "description", "d", "", "extension description")
	createCmd.MarkFlagRequired("name")
	createCmd.RunE = bindExtensionsCommand(options, func() core.Command {
		return &CreateCommand{
			Name:        name,
			Description: description,
		}
	})

	//
	// superplane extensions create-version
	//
	createVersionCmd := &cobra.Command{
		Use:   "create-version",
		Short: "Create new version for an extension",
		Args:  cobra.NoArgs,
	}

	var entryPoint string
	var watch bool
	createVersionCmd.Flags().StringVarP(&extensionID, "extension-id", "e", "", "extension ID")
	createVersionCmd.Flags().StringVarP(&entryPoint, "entry-point", "t", "", "entry point")
	createVersionCmd.Flags().BoolVarP(&watch, "watch", "w", false, "watch for changes")
	createVersionCmd.MarkFlagRequired("extension-id")
	createVersionCmd.RunE = bindExtensionsCommand(options, func() core.Command {
		return &CreateVersionCommand{
			ExtensionID: extensionID,
			EntryPoint:  entryPoint,
			Watch:       watch,
		}
	})

	//
	// superplane extensions update-version
	//
	updateVersionCmd := &cobra.Command{
		Use:   "update-version",
		Short: "Update version for an extension",
		Args:  cobra.NoArgs,
	}

	var updateVersionID string
	var updateEntryPoint string
	var updateWatch bool
	updateVersionCmd.Flags().StringVarP(&extensionID, "extension-id", "e", "", "extension ID")
	updateVersionCmd.Flags().StringVarP(&updateVersionID, "version-id", "v", "", "version ID")
	updateVersionCmd.Flags().StringVarP(&updateEntryPoint, "entry-point", "t", "", "entry point")
	updateVersionCmd.Flags().BoolVarP(&updateWatch, "watch", "w", false, "watch for changes")
	updateVersionCmd.MarkFlagRequired("extension-id")
	updateVersionCmd.MarkFlagRequired("version-id")
	updateVersionCmd.RunE = bindExtensionsCommand(options, func() core.Command {
		return &UpdateVersionCommand{
			ExtensionID: extensionID,
			VersionID:   updateVersionID,
			EntryPoint:  updateEntryPoint,
			Watch:       updateWatch,
		}
	})

	//
	// superplane extensions package-version
	//
	packageVersionCmd := &cobra.Command{
		Use:   "package-version",
		Short: "Package a version of an extension locally",
		Args:  cobra.NoArgs,
	}

	var destination string
	var packageEntryPoint string
	packageVersionCmd.Flags().StringVarP(&destination, "destination", "d", "./dist", "destination directory")
	packageVersionCmd.Flags().StringVarP(&packageEntryPoint, "entry-point", "t", "", "entry point")
	packageVersionCmd.RunE = bindExtensionsCommand(options, func() core.Command {
		return &PackageVersionCommand{
			Destination: destination,
			EntryPoint:  packageEntryPoint,
		}
	})

	//
	// superplane extensions publish-version
	//
	publishVersionCmd := &cobra.Command{
		Use:   "publish-version",
		Short: "Publish version for an extension",
		Args:  cobra.NoArgs,
	}

	var versionID string
	var version string
	publishVersionCmd.Flags().StringVarP(&extensionID, "extension-id", "e", "", "extension ID")
	publishVersionCmd.Flags().StringVarP(&versionID, "version-id", "v", "", "version ID")
	publishVersionCmd.Flags().StringVarP(&version, "version", "", "", "version")
	publishVersionCmd.MarkFlagRequired("extension-id")
	publishVersionCmd.MarkFlagRequired("version-id")
	publishVersionCmd.MarkFlagRequired("version")

	publishVersionCmd.RunE = bindExtensionsCommand(options, func() core.Command {
		return &PublishVersionCommand{
			ExtensionID: extensionID,
			VersionID:   versionID,
			Version:     version,
		}
	})

	// Attach commands to root command
	//
	root.AddCommand(listCmd)
	root.AddCommand(listVersionsCmd)
	root.AddCommand(createCmd)
	root.AddCommand(createVersionCmd)
	root.AddCommand(updateVersionCmd)
	root.AddCommand(packageVersionCmd)
	root.AddCommand(publishVersionCmd)

	return root
}

func bindExtensionsCommand(options core.BindOptions, factory func() core.Command) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx, err := core.NewCommandContext(cmd, args, options)
		if err != nil {
			return err
		}

		return factory().Execute(ctx)
	}
}
