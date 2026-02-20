package slack

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_send_text_message.json
var exampleOutputSendTextMessageBytes []byte

//go:embed example_output_wait_for_button_click.json
var exampleOutputWaitForButtonClickBytes []byte

//go:embed example_data_on_app_mention.json
var exampleDataOnAppMentionBytes []byte

var exampleOutputSendTextMessageOnce sync.Once
var exampleOutputSendTextMessage map[string]any

var exampleOutputWaitForButtonClickOnce sync.Once
var exampleOutputWaitForButtonClick map[string]any

var exampleDataOnce sync.Once
var exampleData map[string]any

func (c *SendTextMessage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputSendTextMessageOnce, exampleOutputSendTextMessageBytes, &exampleOutputSendTextMessage)
}

func (c *WaitForButtonClick) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputWaitForButtonClickOnce, exampleOutputWaitForButtonClickBytes, &exampleOutputWaitForButtonClick)
}

func (t *OnAppMention) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnce, exampleDataOnAppMentionBytes, &exampleData)
}
