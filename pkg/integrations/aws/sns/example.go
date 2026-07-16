package sns

import (
	_ "embed"

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
var exampleDataOnTopicMessage = utils.NewEmbeddedJSON(exampleDataOnTopicMessageBytes)
var exampleOutputGetTopic = utils.NewEmbeddedJSON(exampleOutputGetTopicBytes)
var exampleOutputGetSubscription = utils.NewEmbeddedJSON(exampleOutputGetSubscriptionBytes)
var exampleOutputCreateTopic = utils.NewEmbeddedJSON(exampleOutputCreateTopicBytes)
var exampleOutputDeleteTopic = utils.NewEmbeddedJSON(exampleOutputDeleteTopicBytes)
var exampleOutputPublishMessage = utils.NewEmbeddedJSON(exampleOutputPublishMessageBytes)

// ExampleData returns an example payload for OnTopicMessage events.
func (t *OnTopicMessage) ExampleData() map[string]any {
	return exampleDataOnTopicMessage.Value()
}

// ExampleOutput returns an example payload for GetTopic.
func (c *GetTopic) ExampleOutput() map[string]any {
	return exampleOutputGetTopic.Value()
}

// ExampleOutput returns an example payload for GetSubscription.
func (c *GetSubscription) ExampleOutput() map[string]any {
	return exampleOutputGetSubscription.Value()
}

// ExampleOutput returns an example payload for CreateTopic.
func (c *CreateTopic) ExampleOutput() map[string]any {
	return exampleOutputCreateTopic.Value()
}

// ExampleOutput returns an example payload for DeleteTopic.
func (c *DeleteTopic) ExampleOutput() map[string]any {
	return exampleOutputDeleteTopic.Value()
}

// ExampleOutput returns an example payload for PublishMessage.
func (c *PublishMessage) ExampleOutput() map[string]any {
	return exampleOutputPublishMessage.Value()
}
