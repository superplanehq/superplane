package smtp

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_send_email.json
var exampleOutputSendEmailBytes []byte

var exampleOutputOnce sync.Once
var exampleOutput map[string]any

func (c *SendEmail) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputOnce, exampleOutputSendEmailBytes, &exampleOutput)
}
