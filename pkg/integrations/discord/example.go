package discord

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_send_text_message.json
var exampleOutputSendTextMessageBytes []byte

//go:embed example_output_get_last_mention.json
var exampleOutputGetLastMentionBytes []byte

var exampleOutputSendTextMessageOnce sync.Once
var exampleOutputSendTextMessage map[string]any

var exampleOutputGetLastMentionOnce sync.Once
var exampleOutputGetLastMention map[string]any

func (c *SendTextMessage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputSendTextMessageOnce,
		exampleOutputSendTextMessageBytes,
		&exampleOutputSendTextMessage,
	)
}

func (c *GetLastMention) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetLastMentionOnce,
		exampleOutputGetLastMentionBytes,
		&exampleOutputGetLastMention,
	)
}
