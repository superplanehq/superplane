package logfire

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_alert_received.json
var exampleDataOnAlertReceivedBytes []byte

//go:embed example_output_query_logfire.json
var exampleOutputQueryLogfireBytes []byte
var exampleDataOnAlertReceived = utils.NewEmbeddedJSON(exampleDataOnAlertReceivedBytes)
var exampleOutputQueryLogfire = utils.NewEmbeddedJSON(exampleOutputQueryLogfireBytes)

func (t *OnAlertReceived) ExampleData() map[string]any {
	return exampleDataOnAlertReceived.Value()
}

func (c *QueryLogfire) ExampleOutput() map[string]any {
	return exampleOutputQueryLogfire.Value()
}
