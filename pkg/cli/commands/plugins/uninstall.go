package plugins

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/plugins"
)

type uninstallCommand struct{}

func newUninstallCommand(options core.BindOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall <plugin-name>",
		Short: "Uninstall a plugin",
		Args:  cobra.ExactArgs(1),
	}
	core.Bind(cmd, &uninstallCommand{}, options)
	return cmd
}

func (c *uninstallCommand) Execute(ctx core.CommandContext) error {
	pluginName := ctx.Args[0]

	pluginsDir := os.Getenv("SUPERPLANE_PLUGINS_DIR")
	if pluginsDir == "" {
		pluginsDir = "plugins"
	}

	pj, err := plugins.ReadPluginsJSON(pluginsDir)
	if err != nil {
		return fmt.Errorf("reading plugins.json: %w", err)
	}

	found := false
	filtered := make([]plugins.PluginRecord, 0, len(pj.Plugins))
	for _, p := range pj.Plugins {
		if p.Name == pluginName {
			found = true
			continue
		}
		filtered = append(filtered, p)
	}

	if !found {
		return fmt.Errorf("plugin %q is not installed", pluginName)
	}

	pluginDir := filepath.Join(pluginsDir, pluginName)
	if err := os.RemoveAll(pluginDir); err != nil {
		return fmt.Errorf("removing plugin directory: %w", err)
	}

	pj.Plugins = filtered
	if err := plugins.WritePluginsJSON(pluginsDir, pj); err != nil {
		return fmt.Errorf("writing plugins.json: %w", err)
	}

	fmt.Fprintf(ctx.Cmd.OutOrStdout(), "Uninstalled %s\n", pluginName)

	if err := reloadPluginsViaAPI(ctx); err != nil {
		return err
	}

	return nil
}
