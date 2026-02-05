package sendgrid

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_send_email.json
var exampleOutputSendEmailBytes []byte

//go:embed example_data_on_email_event.json
var exampleDataOnEmailEventBytes []byte

var exampleOutputSendEmailOnce sync.Once
var exampleOutputSendEmail map[string]any

var exampleDataOnEmailEventOnce sync.Once
var exampleDataOnEmailEvent map[string]any

func (c *SendEmail) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputSendEmailOnce, exampleOutputSendEmailBytes, &exampleOutputSendEmail)
}

func (t *OnEmailEvent) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnEmailEventOnce, exampleDataOnEmailEventBytes, &exampleDataOnEmailEvent)
}
