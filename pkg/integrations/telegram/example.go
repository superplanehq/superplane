package telegram

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_send_message.json
var exampleOutputSendMessageBytes []byte

//go:embed example_data_on_mention.json
var exampleDataOnMentionBytes []byte

//go:embed example_output_wait_for_button_click.json
var exampleOutputWaitForButtonClickBytes []byte

var exampleOutputOnce sync.Once
var exampleOutput map[string]any

var exampleDataOnce sync.Once
var exampleData map[string]any

var exampleOutputWaitForButtonClickOnce sync.Once
var exampleOutputWaitForButtonClick map[string]any

func (c *SendMessage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputOnce, exampleOutputSendMessageBytes, &exampleOutput)
}

func (t *OnMention) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnce, exampleDataOnMentionBytes, &exampleData)
}

func (c *WaitForButtonClick) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputWaitForButtonClickOnce, exampleOutputWaitForButtonClickBytes, &exampleOutputWaitForButtonClick)
}
