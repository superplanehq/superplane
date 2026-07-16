package sqs

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_queue.json
var exampleOutputCreateQueueBytes []byte
var exampleOutputCreateQueue = utils.NewEmbeddedJSON(exampleOutputCreateQueueBytes)

//go:embed example_output_delete_queue.json
var exampleOutputDeleteQueueBytes []byte
var exampleOutputDeleteQueue = utils.NewEmbeddedJSON(exampleOutputDeleteQueueBytes)

//go:embed example_output_purge_queue.json
var exampleOutputPurgeQueueBytes []byte
var exampleOutputPurgeQueue = utils.NewEmbeddedJSON(exampleOutputPurgeQueueBytes)

//go:embed example_output_get_queue.json
var exampleOutputGetQueueBytes []byte
var exampleOutputGetQueue = utils.NewEmbeddedJSON(exampleOutputGetQueueBytes)

//go:embed example_output_send_message.json
var exampleOutputSendMessageBytes []byte
var exampleOutputSendMessage = utils.NewEmbeddedJSON(exampleOutputSendMessageBytes)

func (c *CreateQueue) ExampleOutput() map[string]any {
	return exampleOutputCreateQueue.Value()
}

func (c *DeleteQueue) ExampleOutput() map[string]any {
	return exampleOutputDeleteQueue.Value()
}

func (c *PurgeQueue) ExampleOutput() map[string]any {
	return exampleOutputPurgeQueue.Value()
}

func (c *GetQueue) ExampleOutput() map[string]any {
	return exampleOutputGetQueue.Value()
}

func (c *SendMessage) ExampleOutput() map[string]any {
	return exampleOutputSendMessage.Value()
}
