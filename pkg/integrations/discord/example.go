package discord

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_send_text_message.json
var exampleOutputSendTextMessageBytes []byte

//go:embed example_output_get_last_mention.json
var exampleOutputGetLastMentionBytes []byte
var exampleOutputSendTextMessage = utils.NewEmbeddedJSON(exampleOutputSendTextMessageBytes)
var exampleOutputGetLastMention = utils.NewEmbeddedJSON(exampleOutputGetLastMentionBytes)

func (c *SendTextMessage) ExampleOutput() map[string]any {
	return exampleOutputSendTextMessage.Value()
}

func (c *GetLastMention) ExampleOutput() map[string]any {
	return exampleOutputGetLastMention.Value()
}
