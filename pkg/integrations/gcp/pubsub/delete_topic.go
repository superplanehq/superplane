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
	deleteTopicOutputChannel = "default"
	deleteTopicPayloadType   = "gcp.pubsub.topic.deleted"
)

type DeleteTopicComponent struct{}

type DeleteTopicConfiguration struct {
	Topic string `json:"topic" mapstructure:"topic"`
}

func (c *DeleteTopicComponent) Name() string        { return "gcp.pubsub.deleteTopic" }
func (c *DeleteTopicComponent) Label() string       { return "Pub/Sub • Delete Topic" }
func (c *DeleteTopicComponent) Description() string { return "Delete a GCP Pub/Sub topic" }
func (c *DeleteTopicComponent) Icon() string        { return "gcp" }
func (c *DeleteTopicComponent) Color() string       { return "gray" }

func (c *DeleteTopicComponent) Documentation() string {
	return `The Delete Topic component deletes a GCP Pub/Sub topic.

## Use Cases

- **Cleanup workflows**: Remove temporary topics after execution
- **Lifecycle management**: Decommission unused messaging resources
- **Rollback automation**: Remove topics created in failed provisioning runs`
}

func (c *DeleteTopicComponent) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteTopicComponent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "topic",
			Label:       "Topic",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the Pub/Sub topic to delete.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeTopic,
					Parameters: []configuration.ParameterRef{},
				},
			},
		},
	}
}

func (c *DeleteTopicComponent) Setup(ctx core.SetupContext) error {
	var config DeleteTopicConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(config.Topic) == "" {
		return fmt.Errorf("topic is required")
	}
	return nil
}

func (c *DeleteTopicComponent) Execute(ctx core.ExecutionContext) error {
	var config DeleteTopicConfiguration
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
	if err := DeleteTopic(context.Background(), client, projectID, config.Topic); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to delete topic: %v", err))
	}

	return ctx.ExecutionState.Emit(deleteTopicOutputChannel, deleteTopicPayloadType, []any{
		map[string]any{
			"topic":   config.Topic,
			"deleted": true,
		},
	})
}

func (c *DeleteTopicComponent) Actions() []core.Action                  { return nil }
func (c *DeleteTopicComponent) HandleAction(_ core.ActionContext) error { return nil }
func (c *DeleteTopicComponent) Cancel(_ core.ExecutionContext) error    { return nil }
func (c *DeleteTopicComponent) Cleanup(_ core.SetupContext) error       { return nil }
func (c *DeleteTopicComponent) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *DeleteTopicComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
