package pubsub

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_message.json
var exampleDataOnMessageBytes []byte

//go:embed example_output_publish_message.json
var exampleOutputPublishMessageBytes []byte

//go:embed example_output_create_topic.json
var exampleOutputCreateTopicBytes []byte

//go:embed example_output_delete_topic.json
var exampleOutputDeleteTopicBytes []byte

//go:embed example_output_create_subscription.json
var exampleOutputCreateSubscriptionBytes []byte

//go:embed example_output_delete_subscription.json
var exampleOutputDeleteSubscriptionBytes []byte

var (
	exampleDataOnMessage            = utils.NewEmbeddedJSON(exampleDataOnMessageBytes)
	exampleOutputPublishMessage     = utils.NewEmbeddedJSON(exampleOutputPublishMessageBytes)
	exampleOutputCreateTopic        = utils.NewEmbeddedJSON(exampleOutputCreateTopicBytes)
	exampleOutputDeleteTopic        = utils.NewEmbeddedJSON(exampleOutputDeleteTopicBytes)
	exampleOutputCreateSubscription = utils.NewEmbeddedJSON(exampleOutputCreateSubscriptionBytes)
	exampleOutputDeleteSubscription = utils.NewEmbeddedJSON(exampleOutputDeleteSubscriptionBytes)
)

func (t *OnMessage) ExampleData() map[string]any {
	return exampleDataOnMessage.Value()
}

func (c *PublishMessage) ExampleOutput() map[string]any {
	return exampleOutputPublishMessage.Value()
}

func (c *CreateTopicComponent) ExampleOutput() map[string]any {
	return exampleOutputCreateTopic.Value()
}

func (c *DeleteTopicComponent) ExampleOutput() map[string]any {
	return exampleOutputDeleteTopic.Value()
}

func (c *CreateSubscriptionComponent) ExampleOutput() map[string]any {
	return exampleOutputCreateSubscription.Value()
}

func (c *DeleteSubscriptionComponent) ExampleOutput() map[string]any {
	return exampleOutputDeleteSubscription.Value()
}
