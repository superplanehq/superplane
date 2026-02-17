package teams

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_send_text_message.json
var exampleOutputSendTextMessageBytes []byte

//go:embed example_data_on_mention.json
var exampleDataOnMentionBytes []byte

//go:embed example_data_on_message.json
var exampleDataOnMessageBytes []byte

var exampleOutputOnce sync.Once
var exampleOutput map[string]any

var exampleDataOnMentionOnce sync.Once
var exampleDataOnMention map[string]any

var exampleDataOnMessageOnce sync.Once
var exampleDataOnMessage map[string]any

func (c *SendTextMessage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputOnce, exampleOutputSendTextMessageBytes, &exampleOutput)
}

func (t *OnMention) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnMentionOnce, exampleDataOnMentionBytes, &exampleDataOnMention)
}

func (t *OnMessage) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnMessageOnce, exampleDataOnMessageBytes, &exampleDataOnMessage)
}
