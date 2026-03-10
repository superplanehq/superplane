package pubsub

import (
	_ "embed"
	"sync"

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
	exampleDataOnMessageOnce sync.Once
	exampleDataOnMessage     map[string]any

	exampleOutputPublishMessageOnce sync.Once
	exampleOutputPublishMessage     map[string]any

	exampleOutputCreateTopicOnce sync.Once
	exampleOutputCreateTopic     map[string]any

	exampleOutputDeleteTopicOnce sync.Once
	exampleOutputDeleteTopic     map[string]any

	exampleOutputCreateSubscriptionOnce sync.Once
	exampleOutputCreateSubscription     map[string]any

	exampleOutputDeleteSubscriptionOnce sync.Once
	exampleOutputDeleteSubscription     map[string]any
)

func (t *OnMessage) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnMessageOnce, exampleDataOnMessageBytes, &exampleDataOnMessage)
}

func (c *PublishMessage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputPublishMessageOnce, exampleOutputPublishMessageBytes, &exampleOutputPublishMessage)
}

func (c *CreateTopicComponent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateTopicOnce, exampleOutputCreateTopicBytes, &exampleOutputCreateTopic)
}

func (c *DeleteTopicComponent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteTopicOnce, exampleOutputDeleteTopicBytes, &exampleOutputDeleteTopic)
}

func (c *CreateSubscriptionComponent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateSubscriptionOnce, exampleOutputCreateSubscriptionBytes, &exampleOutputCreateSubscription)
}

func (c *DeleteSubscriptionComponent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteSubscriptionOnce, exampleOutputDeleteSubscriptionBytes, &exampleOutputDeleteSubscription)
}
