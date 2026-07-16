package slack

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_send_text_message.json
var exampleOutputSendTextMessageBytes []byte

//go:embed example_output_wait_for_button_click.json
var exampleOutputWaitForButtonClickBytes []byte

//go:embed example_data_on_app_mention.json
var exampleDataOnAppMentionBytes []byte
var exampleOutputSendTextMessage = utils.NewEmbeddedJSON(exampleOutputSendTextMessageBytes)
var exampleOutputWaitForButtonClick = utils.NewEmbeddedJSON(exampleOutputWaitForButtonClickBytes)
var exampleData = utils.NewEmbeddedJSON(exampleDataOnAppMentionBytes)

func (c *SendTextMessage) ExampleOutput() map[string]any {
	return exampleOutputSendTextMessage.Value()
}

func (c *WaitForButtonClick) ExampleOutput() map[string]any {
	return exampleOutputWaitForButtonClick.Value()
}

func (t *OnAppMention) ExampleData() map[string]any {
	return exampleData.Value()
}
