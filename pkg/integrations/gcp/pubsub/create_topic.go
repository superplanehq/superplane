package pubsub

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

const (
	createTopicOutputChannel = "default"
	createTopicPayloadType   = "gcp.pubsub.topic"
)

type CreateTopicComponent struct{}

type CreateTopicConfiguration struct {
	Topic string `json:"topic" mapstructure:"topic"`
}

func (c *CreateTopicComponent) Name() string        { return "gcp.pubsub.createTopic" }
func (c *CreateTopicComponent) Label() string       { return "Pub/Sub • Create Topic" }
func (c *CreateTopicComponent) Description() string { return "Create a GCP Pub/Sub topic" }
func (c *CreateTopicComponent) Icon() string        { return "gcp" }
func (c *CreateTopicComponent) Color() string       { return "gray" }

func (c *CreateTopicComponent) Documentation() string {
	return `The Create Topic component creates a new GCP Pub/Sub topic.

## Use Cases

- **Provisioning workflows**: Create topics as part of environment setup
- **Dynamic routing**: Create topics on demand for new services or tenants
- **Automation bootstrap**: Prepare messaging infrastructure before publishing`
}

func (c *CreateTopicComponent) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateTopicComponent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "topic",
			Label:       "Topic",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The name for the new Pub/Sub topic (e.g. my-topic).",
			Placeholder: "my-topic",
		},
	}
}

func (c *CreateTopicComponent) Setup(ctx core.SetupContext) error {
	var config CreateTopicConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(config.Topic) == "" {
		return fmt.Errorf("topic is required")
	}
	return nil
}

func (c *CreateTopicComponent) Execute(ctx core.ExecutionContext) error {
	var config CreateTopicConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	config.Topic = strings.TrimSpace(config.Topic)
	if config.Topic == "" {
		return ctx.ExecutionState.Fail("error", "topic is required")
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	projectID := client.ProjectID()
	if err := CreateTopic(context.Background(), client, projectID, config.Topic); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create topic: %v", err))
	}

	return ctx.ExecutionState.Emit(createTopicOutputChannel, createTopicPayloadType, []any{
		map[string]any{
			"topic": config.Topic,
			"name":  fmt.Sprintf("projects/%s/topics/%s", projectID, config.Topic),
		},
	})
}

func (c *CreateTopicComponent) Actions() []core.Action                  { return nil }
func (c *CreateTopicComponent) HandleAction(_ core.ActionContext) error { return nil }
func (c *CreateTopicComponent) Cancel(_ core.ExecutionContext) error    { return nil }
func (c *CreateTopicComponent) Cleanup(_ core.SetupContext) error       { return nil }
func (c *CreateTopicComponent) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *CreateTopicComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
