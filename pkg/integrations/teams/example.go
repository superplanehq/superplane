package teams

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_send_text_message.json
var exampleOutputSendTextMessageBytes []byte

//go:embed example_data_on_mention.json
var exampleDataOnMentionBytes []byte

//go:embed example_data_on_message.json
var exampleDataOnMessageBytes []byte
var exampleOutput = utils.NewEmbeddedJSON(exampleOutputSendTextMessageBytes)
var exampleDataOnMention = utils.NewEmbeddedJSON(exampleDataOnMentionBytes)
var exampleDataOnMessage = utils.NewEmbeddedJSON(exampleDataOnMessageBytes)

func (c *SendTextMessage) ExampleOutput() map[string]any {
	return exampleOutput.Value()
}

func (t *OnMention) ExampleData() map[string]any {
	return exampleDataOnMention.Value()
}

func (t *OnMessage) ExampleData() map[string]any {
	return exampleDataOnMessage.Value()
}
