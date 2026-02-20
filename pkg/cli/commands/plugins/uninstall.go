package plugins

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/plugins"
)

func newUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall <plugin-name>",
		Short: "Uninstall a plugin",
		Args:  cobra.ExactArgs(1),
		RunE:  runUninstall,
	}
}

func runUninstall(cmd *cobra.Command, args []string) error {
	pluginName := args[0]

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

	fmt.Fprintf(cmd.OutOrStdout(), "Uninstalled %s\n", pluginName)

	signalServer(cmd)

	return nil
}
