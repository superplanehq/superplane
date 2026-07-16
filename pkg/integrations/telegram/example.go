package telegram

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_send_message.json
var exampleOutputSendMessageBytes []byte

//go:embed example_data_on_mention.json
var exampleDataOnMentionBytes []byte

//go:embed example_output_wait_for_button_click.json
var exampleOutputWaitForButtonClickBytes []byte
var exampleOutput = utils.NewEmbeddedJSON(exampleOutputSendMessageBytes)
var exampleData = utils.NewEmbeddedJSON(exampleDataOnMentionBytes)
var exampleOutputWaitForButtonClick = utils.NewEmbeddedJSON(exampleOutputWaitForButtonClickBytes)

func (c *SendMessage) ExampleOutput() map[string]any {
	return exampleOutput.Value()
}

func (t *OnMention) ExampleData() map[string]any {
	return exampleData.Value()
}

func (c *WaitForButtonClick) ExampleOutput() map[string]any {
	return exampleOutputWaitForButtonClick.Value()
}
