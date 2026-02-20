package plugins

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/plugins"
)

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed plugins",
		Args:  cobra.NoArgs,
		RunE:  runList,
	}
}

func runList(cmd *cobra.Command, args []string) error {
	pluginsDir := os.Getenv("SUPERPLANE_PLUGINS_DIR")
	if pluginsDir == "" {
		pluginsDir = "plugins"
	}

	pj, err := plugins.ReadPluginsJSON(pluginsDir)
	if err != nil {
		return fmt.Errorf("reading plugins.json: %w", err)
	}

	if len(pj.Plugins) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No plugins installed")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tINSTALLED")
	for _, p := range pj.Plugins {
		fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, p.Version, p.InstalledAt.Format("2006-01-02 15:04:05"))
	}
	return w.Flush()
}
