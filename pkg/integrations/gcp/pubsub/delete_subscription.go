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
	deleteSubscriptionOutputChannel = "default"
	deleteSubscriptionPayloadType   = "gcp.pubsub.subscription.deleted"
)

type DeleteSubscriptionComponent struct{}

type DeleteSubscriptionConfiguration struct {
	Subscription string `json:"subscription" mapstructure:"subscription"`
}

func (c *DeleteSubscriptionComponent) Name() string  { return "gcp.pubsub.deleteSubscription" }
func (c *DeleteSubscriptionComponent) Label() string { return "Pub/Sub • Delete Subscription" }
func (c *DeleteSubscriptionComponent) Description() string {
	return "Delete a GCP Pub/Sub subscription"
}
func (c *DeleteSubscriptionComponent) Icon() string  { return "gcp" }
func (c *DeleteSubscriptionComponent) Color() string { return "gray" }

func (c *DeleteSubscriptionComponent) Documentation() string {
	return `The Delete Subscription component deletes a GCP Pub/Sub subscription.

## Use Cases

- **Cleanup workflows**: Remove subscriptions as part of service teardown
- **Lifecycle management**: Decommission subscriptions that are no longer needed
- **Rollback automation**: Remove subscriptions created in failed provisioning runs`
}

func (c *DeleteSubscriptionComponent) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteSubscriptionComponent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "subscription",
			Label:       "Subscription",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the Pub/Sub subscription to delete.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeSubscription,
					Parameters: []configuration.ParameterRef{},
				},
			},
		},
	}
}

func (c *DeleteSubscriptionComponent) Setup(ctx core.SetupContext) error {
	var config DeleteSubscriptionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(config.Subscription) == "" {
		return fmt.Errorf("subscription is required")
	}
	return nil
}

func (c *DeleteSubscriptionComponent) Execute(ctx core.ExecutionContext) error {
	var config DeleteSubscriptionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	config.Subscription = strings.TrimSpace(config.Subscription)
	if config.Subscription == "" {
		return ctx.ExecutionState.Fail("error", "subscription is required")
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	projectID := client.ProjectID()
	if err := DeleteSubscription(context.Background(), client, projectID, config.Subscription); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to delete subscription: %v", err))
	}

	return ctx.ExecutionState.Emit(deleteSubscriptionOutputChannel, deleteSubscriptionPayloadType, []any{
		map[string]any{
			"subscription": config.Subscription,
			"deleted":      true,
		},
	})
}

func (c *DeleteSubscriptionComponent) Actions() []core.Action                  { return nil }
func (c *DeleteSubscriptionComponent) HandleAction(_ core.ActionContext) error { return nil }
func (c *DeleteSubscriptionComponent) Cancel(_ core.ExecutionContext) error    { return nil }
func (c *DeleteSubscriptionComponent) Cleanup(_ core.SetupContext) error       { return nil }
func (c *DeleteSubscriptionComponent) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *DeleteSubscriptionComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
