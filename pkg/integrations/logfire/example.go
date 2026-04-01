package logfire

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_alert_received.json
var exampleDataOnAlertReceivedBytes []byte

//go:embed example_output_query_logfire.json
var exampleOutputQueryLogfireBytes []byte

var exampleDataOnAlertReceivedOnce sync.Once
var exampleDataOnAlertReceived map[string]any

var exampleOutputQueryLogfireOnce sync.Once
var exampleOutputQueryLogfire map[string]any

func (t *OnAlertReceived) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnAlertReceivedOnce, exampleDataOnAlertReceivedBytes, &exampleDataOnAlertReceived)
}

func (c *QueryLogfire) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputQueryLogfireOnce, exampleOutputQueryLogfireBytes, &exampleOutputQueryLogfire)
}
