package sns

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_topic_message.json
var exampleDataOnTopicMessageBytes []byte

//go:embed example_output_get_topic.json
var exampleOutputGetTopicBytes []byte

//go:embed example_output_get_subscription.json
var exampleOutputGetSubscriptionBytes []byte

//go:embed example_output_create_topic.json
var exampleOutputCreateTopicBytes []byte

//go:embed example_output_delete_topic.json
var exampleOutputDeleteTopicBytes []byte

//go:embed example_output_publish_message.json
var exampleOutputPublishMessageBytes []byte

var exampleDataOnTopicMessageOnce sync.Once
var exampleDataOnTopicMessage map[string]any

var exampleOutputGetTopicOnce sync.Once
var exampleOutputGetTopic map[string]any

var exampleOutputGetSubscriptionOnce sync.Once
var exampleOutputGetSubscription map[string]any

var exampleOutputCreateTopicOnce sync.Once
var exampleOutputCreateTopic map[string]any

var exampleOutputDeleteTopicOnce sync.Once
var exampleOutputDeleteTopic map[string]any

var exampleOutputPublishMessageOnce sync.Once
var exampleOutputPublishMessage map[string]any

// ExampleData returns an example payload for OnTopicMessage events.
func (t *OnTopicMessage) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnTopicMessageOnce, exampleDataOnTopicMessageBytes, &exampleDataOnTopicMessage)
}

// ExampleOutput returns an example payload for GetTopic.
func (c *GetTopic) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetTopicOnce, exampleOutputGetTopicBytes, &exampleOutputGetTopic)
}

// ExampleOutput returns an example payload for GetSubscription.
func (c *GetSubscription) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetSubscriptionOnce, exampleOutputGetSubscriptionBytes, &exampleOutputGetSubscription)
}

// ExampleOutput returns an example payload for CreateTopic.
func (c *CreateTopic) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateTopicOnce, exampleOutputCreateTopicBytes, &exampleOutputCreateTopic)
}

// ExampleOutput returns an example payload for DeleteTopic.
func (c *DeleteTopic) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteTopicOnce, exampleOutputDeleteTopicBytes, &exampleOutputDeleteTopic)
}

// ExampleOutput returns an example payload for PublishMessage.
func (c *PublishMessage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputPublishMessageOnce, exampleOutputPublishMessageBytes, &exampleOutputPublishMessage)
}
