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
	createSubscriptionOutputChannel = "default"
	createSubscriptionPayloadType   = "gcp.pubsub.subscription"
)

type CreateSubscriptionComponent struct{}

type CreateSubscriptionConfiguration struct {
	Topic        string `json:"topic" mapstructure:"topic"`
	Subscription string `json:"subscription" mapstructure:"subscription"`
	Type         string `json:"type" mapstructure:"type"`
	PushEndpoint string `json:"pushEndpoint" mapstructure:"pushEndpoint"`
}

func (c *CreateSubscriptionComponent) Name() string  { return "gcp.pubsub.createSubscription" }
func (c *CreateSubscriptionComponent) Label() string { return "Pub/Sub • Create Subscription" }
func (c *CreateSubscriptionComponent) Description() string {
	return "Create a GCP Pub/Sub subscription"
}
func (c *CreateSubscriptionComponent) Icon() string  { return "gcp" }
func (c *CreateSubscriptionComponent) Color() string { return "gray" }

func (c *CreateSubscriptionComponent) Documentation() string {
	return `The Create Subscription component creates a new GCP Pub/Sub subscription on a topic.

## Use Cases

- **Provisioning workflows**: Wire up subscriptions as part of service deployment
- **Pull queue setup**: Create pull subscriptions for batch processing workflows
- **Push integration**: Create push subscriptions that deliver messages to an HTTP endpoint`
}

func (c *CreateSubscriptionComponent) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateSubscriptionComponent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "topic",
			Label:       "Topic",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the Pub/Sub topic to subscribe to.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:       ResourceTypeTopic,
					Parameters: []configuration.ParameterRef{},
				},
			},
		},
		{
			Name:        "subscription",
			Label:       "Subscription",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The name for the new subscription (e.g. my-subscription).",
			Placeholder: "my-subscription",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "topic", Values: []string{"*"}},
			},
		},
		{
			Name:     "type",
			Label:    "Subscription Type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "pull",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "topic", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Pull", Value: "pull"},
						{Label: "Push", Value: "push"},
					},
				},
			},
		},
		{
			Name:        "pushEndpoint",
			Label:       "Push Endpoint",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "HTTPS URL to deliver messages to (push subscriptions only).",
			Placeholder: "https://my-service.example.com/pubsub",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "topic", Values: []string{"*"}},
				{Field: "type", Values: []string{"push"}},
			},
		},
	}
}

func (c *CreateSubscriptionComponent) Setup(ctx core.SetupContext) error {
	var config CreateSubscriptionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(config.Topic) == "" {
		return fmt.Errorf("topic is required")
	}
	if strings.TrimSpace(config.Subscription) == "" {
		return fmt.Errorf("subscription is required")
	}
	if config.Type == "push" && strings.TrimSpace(config.PushEndpoint) == "" {
		return fmt.Errorf("pushEndpoint is required for push subscriptions")
	}
	return nil
}

func (c *CreateSubscriptionComponent) Execute(ctx core.ExecutionContext) error {
	var config CreateSubscriptionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	config.Topic = strings.TrimSpace(config.Topic)
	config.Subscription = strings.TrimSpace(config.Subscription)

	if config.Topic == "" {
		return ctx.ExecutionState.Fail("error", "topic is required")
	}
	if config.Subscription == "" {
		return ctx.ExecutionState.Fail("error", "subscription is required")
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	projectID := client.ProjectID()
	reqCtx := context.Background()

	if config.Type == "push" {
		endpoint := strings.TrimSpace(config.PushEndpoint)
		if endpoint == "" {
			return ctx.ExecutionState.Fail("error", "pushEndpoint is required for push subscriptions")
		}
		if err := CreatePushSubscription(reqCtx, client, projectID, config.Subscription, config.Topic, endpoint); err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create push subscription: %v", err))
		}
	} else {
		if err := CreatePullSubscription(reqCtx, client, projectID, config.Subscription, config.Topic); err != nil {
			return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create pull subscription: %v", err))
		}
	}

	return ctx.ExecutionState.Emit(createSubscriptionOutputChannel, createSubscriptionPayloadType, []any{
		map[string]any{
			"subscription": config.Subscription,
			"topic":        config.Topic,
			"type":         config.Type,
			"name":         fmt.Sprintf("projects/%s/subscriptions/%s", projectID, config.Subscription),
		},
	})
}

func (c *CreateSubscriptionComponent) Actions() []core.Action                  { return nil }
func (c *CreateSubscriptionComponent) HandleAction(_ core.ActionContext) error { return nil }
func (c *CreateSubscriptionComponent) Cancel(_ core.ExecutionContext) error    { return nil }
func (c *CreateSubscriptionComponent) Cleanup(_ core.SetupContext) error       { return nil }
func (c *CreateSubscriptionComponent) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *CreateSubscriptionComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
