package sqs

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_queue.json
var exampleOutputCreateQueueBytes []byte

var exampleOutputCreateQueueOnce sync.Once
var exampleOutputCreateQueue map[string]any

//go:embed example_output_delete_queue.json
var exampleOutputDeleteQueueBytes []byte

var exampleOutputDeleteQueueOnce sync.Once
var exampleOutputDeleteQueue map[string]any

//go:embed example_output_purge_queue.json
var exampleOutputPurgeQueueBytes []byte

var exampleOutputPurgeQueueOnce sync.Once
var exampleOutputPurgeQueue map[string]any

//go:embed example_output_get_queue.json
var exampleOutputGetQueueBytes []byte

var exampleOutputGetQueueOnce sync.Once
var exampleOutputGetQueue map[string]any

//go:embed example_output_send_message.json
var exampleOutputSendMessageBytes []byte

var exampleOutputSendMessageOnce sync.Once
var exampleOutputSendMessage map[string]any

func (c *CreateQueue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateQueueOnce, exampleOutputCreateQueueBytes, &exampleOutputCreateQueue)
}

func (c *DeleteQueue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteQueueOnce, exampleOutputDeleteQueueBytes, &exampleOutputDeleteQueue)
}

func (c *PurgeQueue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputPurgeQueueOnce, exampleOutputPurgeQueueBytes, &exampleOutputPurgeQueue)
}

func (c *GetQueue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetQueueOnce, exampleOutputGetQueueBytes, &exampleOutputGetQueue)
}

func (c *SendMessage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputSendMessageOnce, exampleOutputSendMessageBytes, &exampleOutputSendMessage)
}
