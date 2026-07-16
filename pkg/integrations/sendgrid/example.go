package sendgrid

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_send_email.json
var exampleOutputSendEmailBytes []byte

//go:embed example_output_create_or_update_contact.json
var exampleOutputCreateOrUpdateContactBytes []byte

//go:embed example_data_on_email_event.json
var exampleDataOnEmailEventBytes []byte
var exampleOutputSendEmail = utils.NewEmbeddedJSON(exampleOutputSendEmailBytes)
var exampleOutputCreateOrUpdateContact = utils.NewEmbeddedJSON(exampleOutputCreateOrUpdateContactBytes)
var exampleDataOnEmailEvent = utils.NewEmbeddedJSON(exampleDataOnEmailEventBytes)

func (c *SendEmail) ExampleOutput() map[string]any {
	return exampleOutputSendEmail.Value()
}

func (c *CreateOrUpdateContact) ExampleOutput() map[string]any {
	return exampleOutputCreateOrUpdateContact.Value()
}

func (t *OnEmailEvent) ExampleData() map[string]any {
	return exampleDataOnEmailEvent.Value()
}
