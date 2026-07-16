package smtp

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_send_email.json
var exampleOutputSendEmailBytes []byte
var exampleOutput = utils.NewEmbeddedJSON(exampleOutputSendEmailBytes)

func (c *SendEmail) ExampleOutput() map[string]any {
	return exampleOutput.Value()
}
