package ssh

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output.json
var exampleOutputBytes []byte
var exampleOutput = utils.NewEmbeddedJSON(exampleOutputBytes)

func (c *SSHCommand) ExampleOutput() map[string]any {
	return exampleOutput.Value()
}
