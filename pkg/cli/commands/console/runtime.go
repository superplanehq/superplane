package console

import (
	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

// addRuntimeCommands wires the Console runtime subcommands (data, trigger).
// The actual command implementations live in data.go and trigger.go so that
// each runtime entry point is self-contained.
func addRuntimeCommands(root *cobra.Command, options core.BindOptions) {
	addDataCommand(root, options)
	addTriggerCommand(root, options)
}
