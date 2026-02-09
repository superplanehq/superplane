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

// Unsubscribe deletes an SNS subscription.
type Unsubscribe struct{}

// Name returns the component name.
func (c *Unsubscribe) Name() string {
	return "aws.sns.unsubscribe"
}

// Label returns the component label.
func (c *Unsubscribe) Label() string {
	return "SNS • Unsubscribe"
}

// Description returns a short component description.
func (c *Unsubscribe) Description() string {
	return "Delete an AWS SNS subscription"
}

// Documentation returns detailed Markdown documentation.
func (c *Unsubscribe) Documentation() string {
	return `The Unsubscribe component removes an AWS SNS subscription.

## Use Cases

- **Cleanup workflows**: Remove temporary subscriptions after execution
- **Lifecycle management**: Detach deprecated subscribers
- **Rollback automation**: Revert dynamic subscription changes`
}

// Icon returns the icon slug.
func (c *Unsubscribe) Icon() string {
	return "aws"
}

// Color returns the component color.
func (c *Unsubscribe) Color() string {
	return "gray"
}

// OutputChannels declares the output channels used by this component.
func (c *Unsubscribe) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

// Configuration returns the component configuration schema.
func (c *Unsubscribe) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		subscriptionField(),
	}
}

// Setup validates component configuration.
func (c *Unsubscribe) Setup(ctx core.SetupContext) error {
	var config UnsubscribeConfiguration
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
func (c *Unsubscribe) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

// Execute removes the subscription and emits a deletion payload.
func (c *Unsubscribe) Execute(ctx core.ExecutionContext) error {
	var config UnsubscribeConfiguration
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
	if err := client.Unsubscribe(subscriptionArn); err != nil {
		return fmt.Errorf("%s: failed to unsubscribe ARN %q: %w", c.Name(), subscriptionArn, err)
	}

	if err := ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.sns.subscription.unsubscribed", []any{
		map[string]any{
			"subscriptionArn": subscriptionArn,
			"deleted":         true,
		},
	}); err != nil {
		return fmt.Errorf("%s: failed to emit unsubscription payload: %w", c.Name(), err)
	}

	return nil
}

// Actions returns supported custom actions.
func (c *Unsubscribe) Actions() []core.Action {
	return []core.Action{}
}

// HandleAction handles custom actions for this component.
func (c *Unsubscribe) HandleAction(ctx core.ActionContext) error {
	return nil
}

// HandleWebhook handles incoming webhook requests.
func (c *Unsubscribe) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

// Cancel handles execution cancellation.
func (c *Unsubscribe) Cancel(ctx core.ExecutionContext) error {
	return nil
}

// Cleanup handles component cleanup.
func (c *Unsubscribe) Cleanup(ctx core.SetupContext) error {
	return nil
}
