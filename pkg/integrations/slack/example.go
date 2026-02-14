package slack

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_send_text_message.json
var exampleOutputSendTextMessageBytes []byte

//go:embed example_data_on_app_mention.json
var exampleDataOnAppMentionBytes []byte

var exampleOutputOnce sync.Once
var exampleOutput map[string]any

var exampleDataOnce sync.Once
var exampleData map[string]any

func (c *SendTextMessage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputOnce, exampleOutputSendTextMessageBytes, &exampleOutput)
}

func (t *OnAppMention) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnce, exampleDataOnAppMentionBytes, &exampleData)
}

//go:embed example_output_send_and_wait.json
var exampleOutputSendAndWaitBytes []byte

var exampleOutputSendAndWaitOnce sync.Once
var exampleOutputSendAndWait map[string]any

func (c *SendAndWait) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputSendAndWaitOnce, exampleOutputSendAndWaitBytes, &exampleOutputSendAndWait)
}
