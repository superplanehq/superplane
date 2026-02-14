package sns

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type GetSubscription struct{}

type GetSubscriptionConfiguration struct {
	Region          string `json:"region" mapstructure:"region"`
	TopicArn        string `json:"topicArn" mapstructure:"topicArn"`
	SubscriptionArn string `json:"subscriptionArn" mapstructure:"subscriptionArn"`
}

func (c *GetSubscription) Name() string {
	return "aws.sns.getSubscription"
}

func (c *GetSubscription) Label() string {
	return "SNS • Get Subscription"
}

func (c *GetSubscription) Description() string {
	return "Get an AWS SNS subscription by ARN"
}

func (c *GetSubscription) Documentation() string {
	return `The Get Subscription component retrieves metadata and attributes for an AWS SNS subscription.

## Use Cases

- **Subscription audits**: Inspect endpoint and delivery configuration
- **Workflow enrichment**: Load subscription metadata before downstream actions
- **Validation**: Confirm subscription existence and protocol`
}

func (c *GetSubscription) Icon() string {
	return "aws"
}

func (c *GetSubscription) Color() string {
	return "gray"
}

func (c *GetSubscription) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetSubscription) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		topicField(),
		{
			Name:        "subscriptionArn",
			Label:       "Subscription",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "ARN of the SNS subscription",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
				{
					Field:  "topicArn",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "sns.subscription",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
						{
							Name: "topicArn",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "topicArn",
							},
						},
					},
				},
			},
		},
	}
}

func (c *GetSubscription) Setup(ctx core.SetupContext) error {
	var config GetSubscriptionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode setup configuration: %w", err)
	}

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if config.TopicArn == "" {
		return fmt.Errorf("topic ARN is required")
	}

	if config.SubscriptionArn == "" {
		return fmt.Errorf("subscription ARN is required")
	}

	return nil
}

func (c *GetSubscription) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetSubscription) Execute(ctx core.ExecutionContext) error {
	var config GetSubscriptionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: failed to decode execution configuration: %w", c.Name(), err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("%s: failed to load AWS credentials from integration: %w", c.Name(), err)
	}

	client := NewClient(ctx.HTTP, credentials, config.Region)
	subscription, err := client.GetSubscription(config.SubscriptionArn)
	if err != nil {
		return fmt.Errorf("failed to get subscription %q: %w", config.SubscriptionArn, err)
	}

	if err := ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.sns.subscription", []any{subscription}); err != nil {
		return fmt.Errorf("%s: failed to emit subscription payload: %w", c.Name(), err)
	}

	return nil
}

func (c *GetSubscription) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetSubscription) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetSubscription) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetSubscription) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetSubscription) Cleanup(ctx core.SetupContext) error {
	return nil
}
