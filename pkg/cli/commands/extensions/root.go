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
	listVersionsCmd.Flags().StringVar(&extensionID, "extension-id", "", "extension ID")
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
	createCmd.Flags().StringVar(&name, "name", "", "extension name")
	createCmd.Flags().StringVar(&description, "description", "", "extension description")
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

	var watch bool
	var entryPoint string
	var versionName string
	createVersionCmd.Flags().StringVar(&extensionID, "extension-id", "", "extension ID")
	createVersionCmd.Flags().StringVar(&versionName, "version", "", "version name")
	createVersionCmd.Flags().StringVar(&entryPoint, "entry-point", "", "entry point")
	createVersionCmd.Flags().BoolVarP(&watch, "watch", "w", false, "watch for changes")
	createVersionCmd.MarkFlagRequired("extension-id")
	createVersionCmd.MarkFlagRequired("version")
	createVersionCmd.RunE = bindExtensionsCommand(options, func() core.Command {
		return &CreateVersionCommand{
			ExtensionID: extensionID,
			EntryPoint:  entryPoint,
			Version:     versionName,
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

	updateVersionCmd.Flags().StringVar(&extensionID, "extension-id", "", "extension ID")
	updateVersionCmd.Flags().StringVar(&versionName, "version", "", "version name")
	updateVersionCmd.Flags().StringVar(&entryPoint, "entrypoint", "", "entry point")
	updateVersionCmd.Flags().BoolVar(&watch, "watch", false, "watch for changes")
	updateVersionCmd.MarkFlagRequired("extension-id")
	updateVersionCmd.MarkFlagRequired("version")
	updateVersionCmd.RunE = bindExtensionsCommand(options, func() core.Command {
		return &UpdateVersionCommand{
			ExtensionID: extensionID,
			Version:     versionName,
			EntryPoint:  entryPoint,
			Watch:       watch,
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
	packageVersionCmd.Flags().StringVar(&destination, "destination", "./dist", "destination directory")
	packageVersionCmd.Flags().StringVar(&packageEntryPoint, "entry-point", "", "entry point")
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

	publishVersionCmd.Flags().StringVar(&extensionID, "extension-id", "", "extension ID")
	publishVersionCmd.Flags().StringVar(&versionName, "version", "", "version name")
	publishVersionCmd.MarkFlagRequired("extension-id")
	publishVersionCmd.MarkFlagRequired("version")

	publishVersionCmd.RunE = bindExtensionsCommand(options, func() core.Command {
		return &PublishVersionCommand{
			ExtensionID: extensionID,
			Version:     versionName,
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
