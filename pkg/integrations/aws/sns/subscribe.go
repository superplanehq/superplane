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

// Subscribe creates a subscription for an SNS topic.
type Subscribe struct{}

// Name returns the component name.
func (c *Subscribe) Name() string {
	return "aws.sns.subscribe"
}

// Label returns the component label.
func (c *Subscribe) Label() string {
	return "SNS • Subscribe"
}

// Description returns a short component description.
func (c *Subscribe) Description() string {
	return "Create an AWS SNS subscription"
}

// Documentation returns detailed Markdown documentation.
func (c *Subscribe) Documentation() string {
	return `The Subscribe component creates a new AWS SNS subscription for a topic.

## Use Cases

- **Dynamic delivery setup**: Register temporary workflow endpoints
- **Provisioning workflows**: Configure subscribers during environment setup
- **Automation bootstrap**: Attach systems to topics on demand`
}

// Icon returns the icon slug.
func (c *Subscribe) Icon() string {
	return "aws"
}

// Color returns the component color.
func (c *Subscribe) Color() string {
	return "gray"
}

// OutputChannels declares the output channels used by this component.
func (c *Subscribe) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

// Configuration returns the component configuration schema.
func (c *Subscribe) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		topicField(),
		{
			Name:     "protocol",
			Label:    "Protocol",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "https",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: SubscriptionProtocols,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "topicArn",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:     "endpoint",
			Label:    "Endpoint",
			Type:     configuration.FieldTypeString,
			Required: true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "protocol",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:        "attributes",
			Label:       "Attributes",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "Optional subscription attributes as key-value pairs",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "endpoint",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:        "returnSubscriptionArn",
			Label:       "Return Subscription ARN",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     "false",
			Description: "Request an ARN even when the subscription is pending confirmation",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "endpoint",
					Values: []string{"*"},
				},
			},
		},
	}
}

// Setup validates component configuration.
func (c *Subscribe) Setup(ctx core.SetupContext) error {
	var config SubscribeConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: failed to decode setup configuration: %w", c.Name(), err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return fmt.Errorf("%s: invalid region: %w", c.Name(), err)
	}

	if _, err := requireTopicArn(config.TopicArn); err != nil {
		return fmt.Errorf("%s: invalid topic ARN: %w", c.Name(), err)
	}

	if _, err := requireProtocol(config.Protocol); err != nil {
		return fmt.Errorf("%s: invalid protocol: %w", c.Name(), err)
	}

	if _, err := requireEndpoint(config.Endpoint); err != nil {
		return fmt.Errorf("%s: invalid endpoint: %w", c.Name(), err)
	}

	return nil
}

// ProcessQueueItem applies the default queue-item behavior.
func (c *Subscribe) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

// Execute creates a subscription and emits the resulting subscription metadata.
func (c *Subscribe) Execute(ctx core.ExecutionContext) error {
	var config SubscribeConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: failed to decode execution configuration: %w", c.Name(), err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return fmt.Errorf("%s: invalid region: %w", c.Name(), err)
	}

	topicArn, err := requireTopicArn(config.TopicArn)
	if err != nil {
		return fmt.Errorf("%s: invalid topic ARN: %w", c.Name(), err)
	}

	protocol, err := requireProtocol(config.Protocol)
	if err != nil {
		return fmt.Errorf("%s: invalid protocol: %w", c.Name(), err)
	}

	endpoint, err := requireEndpoint(config.Endpoint)
	if err != nil {
		return fmt.Errorf("%s: invalid endpoint: %w", c.Name(), err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("%s: failed to load AWS credentials from integration: %w", c.Name(), err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	subscription, err := client.Subscribe(SubscribeParameters{
		TopicArn:              topicArn,
		Protocol:              protocol,
		Endpoint:              endpoint,
		Attributes:            mapAnyToStringMap(config.Attributes),
		ReturnSubscriptionARN: config.ReturnSubscriptionArn,
	})
	if err != nil {
		return fmt.Errorf("%s: failed to create subscription for topic %q: %w", c.Name(), topicArn, err)
	}

	if err := ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.sns.subscription", []any{subscription}); err != nil {
		return fmt.Errorf("%s: failed to emit subscription payload: %w", c.Name(), err)
	}

	return nil
}

// Actions returns supported custom actions.
func (c *Subscribe) Actions() []core.Action {
	return []core.Action{}
}

// HandleAction handles custom actions for this component.
func (c *Subscribe) HandleAction(ctx core.ActionContext) error {
	return nil
}

// HandleWebhook handles incoming webhook requests.
func (c *Subscribe) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

// Cancel handles execution cancellation.
func (c *Subscribe) Cancel(ctx core.ExecutionContext) error {
	return nil
}

// Cleanup handles component cleanup.
func (c *Subscribe) Cleanup(ctx core.SetupContext) error {
	return nil
}
