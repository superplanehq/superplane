package slack

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_send_text_message.json
var exampleOutputSendTextMessageBytes []byte

//go:embed example_output_send_and_wait_for_response.json
var exampleOutputSendAndWaitForResponseBytes []byte

//go:embed example_data_on_app_mention.json
var exampleDataOnAppMentionBytes []byte

var exampleOutputOnce sync.Once
var exampleOutput map[string]any

var exampleOutputSendAndWaitForResponseOnce sync.Once
var exampleOutputSendAndWaitForResponse map[string]any

var exampleDataOnce sync.Once
var exampleData map[string]any

func (c *SendTextMessage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputOnce, exampleOutputSendTextMessageBytes, &exampleOutput)
}

func (c *SendAndWaitForResponse) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputSendAndWaitForResponseOnce,
		exampleOutputSendAndWaitForResponseBytes,
		&exampleOutputSendAndWaitForResponse,
	)
}

func (t *OnAppMention) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnce, exampleDataOnAppMentionBytes, &exampleData)
}
