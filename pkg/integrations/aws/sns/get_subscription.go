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

// GetSubscription resolves metadata for an SNS subscription.
type GetSubscription struct{}

// Name returns the component name.
func (c *GetSubscription) Name() string {
	return "aws.sns.getSubscription"
}

// Label returns the component label.
func (c *GetSubscription) Label() string {
	return "SNS • Get Subscription"
}

// Description returns a short component description.
func (c *GetSubscription) Description() string {
	return "Get an AWS SNS subscription by ARN"
}

// Documentation returns detailed Markdown documentation.
func (c *GetSubscription) Documentation() string {
	return `The Get Subscription component retrieves metadata and attributes for an AWS SNS subscription.

## Use Cases

- **Subscription audits**: Inspect endpoint and delivery configuration
- **Workflow enrichment**: Load subscription metadata before downstream actions
- **Validation**: Confirm subscription existence and protocol`
}

// Icon returns the icon slug.
func (c *GetSubscription) Icon() string {
	return "aws"
}

// Color returns the component color.
func (c *GetSubscription) Color() string {
	return "gray"
}

// OutputChannels declares the output channels used by this component.
func (c *GetSubscription) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

// Configuration returns the component configuration schema.
func (c *GetSubscription) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		subscriptionField(),
	}
}

// Setup validates component configuration.
func (c *GetSubscription) Setup(ctx core.SetupContext) error {
	var config GetSubscriptionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: failed to decode setup configuration: %w", c.Name(), err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return fmt.Errorf("%s: invalid region: %w", c.Name(), err)
	}

	if _, err := requireSubscriptionArn(config.SubscriptionArn); err != nil {
		return fmt.Errorf("%s: invalid subscription ARN: %w", c.Name(), err)
	}

	return nil
}

// ProcessQueueItem applies the default queue-item behavior.
func (c *GetSubscription) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

// Execute retrieves subscription data and emits it on the default output channel.
func (c *GetSubscription) Execute(ctx core.ExecutionContext) error {
	var config GetSubscriptionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: failed to decode execution configuration: %w", c.Name(), err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return fmt.Errorf("%s: invalid region: %w", c.Name(), err)
	}

	subscriptionArn, err := requireSubscriptionArn(config.SubscriptionArn)
	if err != nil {
		return fmt.Errorf("%s: invalid subscription ARN: %w", c.Name(), err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("%s: failed to load AWS credentials from integration: %w", c.Name(), err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	subscription, err := client.GetSubscription(subscriptionArn)
	if err != nil {
		return fmt.Errorf("%s: failed to get subscription %q: %w", c.Name(), subscriptionArn, err)
	}

	if err := ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.sns.subscription", []any{subscription}); err != nil {
		return fmt.Errorf("%s: failed to emit subscription payload: %w", c.Name(), err)
	}

	return nil
}

// Actions returns supported custom actions.
func (c *GetSubscription) Actions() []core.Action {
	return []core.Action{}
}

// HandleAction handles custom actions for this component.
func (c *GetSubscription) HandleAction(ctx core.ActionContext) error {
	return nil
}

// HandleWebhook handles incoming webhook requests.
func (c *GetSubscription) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

// Cancel handles execution cancellation.
func (c *GetSubscription) Cancel(ctx core.ExecutionContext) error {
	return nil
}

// Cleanup handles component cleanup.
func (c *GetSubscription) Cleanup(ctx core.SetupContext) error {
	return nil
}
