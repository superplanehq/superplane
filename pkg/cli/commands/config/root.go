package config

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type getCommand struct{}

func (c *getCommand) Execute(ctx core.CommandContext) error {
	key := ctx.Args[0]
	if !viper.IsSet(key) {
		return fmt.Errorf("configuration key %q not found", key)
	}

	value := viper.Get(key)
	if ctx.Renderer.IsText() {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, _ = fmt.Fprintln(stdout, value)
			return nil
		})
	}

	return ctx.Renderer.Render(map[string]any{
		key: value,
	})
}

type setCommand struct{}

func (c *setCommand) Execute(ctx core.CommandContext) error {
	key := ctx.Args[0]
	value := ctx.Args[1]

	viper.Set(key, value)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write configuration: %w", err)
	}

	return nil
}

type viewCommand struct{}

func (c *viewCommand) Execute(ctx core.CommandContext) error {
	allSettings := viper.AllSettings()
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(allSettings)
	}

	if len(allSettings) == 0 {
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			_, _ = fmt.Fprintln(stdout, "No configuration values set")
			return nil
		})
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintln(stdout, "Current configuration:")
		for key, value := range allSettings {
			_, _ = fmt.Fprintf(stdout, "  %s: %v\n", key, value)
		}
		return nil
	})
}

func NewCommand(options core.BindOptions) *cobra.Command {
	root := &cobra.Command{
		Use:   "config",
		Short: "Get and set configuration options",
		Long:  "Get and set CLI configuration options like API URL and authentication token.",
	}

	getCmd := &cobra.Command{
		Use:   "get [KEY]",
		Short: "Display a configuration value",
		Long:  "Display the value of a specific configuration key.",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(getCmd, &getCommand{}, options)

	setCmd := &cobra.Command{
		Use:   "set [KEY] [VALUE]",
		Short: "Set a configuration value",
		Long:  "Set the value of a specific configuration key.",
		Args:  cobra.ExactArgs(2),
	}
	core.Bind(setCmd, &setCommand{}, options)

	viewCmd := &cobra.Command{
		Use:   "view",
		Short: "View all configuration values",
		Long:  "Display all configuration values currently set.",
	}
	core.Bind(viewCmd, &viewCommand{}, options)

	root.AddCommand(getCmd)
	root.AddCommand(setCmd)
	root.AddCommand(viewCmd)

	return root
}
