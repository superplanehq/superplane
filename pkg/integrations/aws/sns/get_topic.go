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

// GetTopic resolves metadata for an SNS topic.
type GetTopic struct{}

// Name returns the component name.
func (c *GetTopic) Name() string {
	return "aws.sns.getTopic"
}

// Label returns the component label.
func (c *GetTopic) Label() string {
	return "SNS • Get Topic"
}

// Description returns a short component description.
func (c *GetTopic) Description() string {
	return "Get an AWS SNS topic by ARN"
}

// Documentation returns detailed Markdown documentation.
func (c *GetTopic) Documentation() string {
	return `The Get Topic component retrieves metadata and attributes for an AWS SNS topic.

## Use Cases

- **Configuration audits**: Verify topic settings and attributes
- **Workflow enrichment**: Load topic metadata before downstream actions
- **Validation**: Confirm topic existence and ownership`
}

// Icon returns the icon slug.
func (c *GetTopic) Icon() string {
	return "aws"
}

// Color returns the component color.
func (c *GetTopic) Color() string {
	return "gray"
}

// OutputChannels declares the output channels used by this component.
func (c *GetTopic) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

// Configuration returns the component configuration schema.
func (c *GetTopic) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		topicField(),
	}
}

// Setup validates component configuration.
func (c *GetTopic) Setup(ctx core.SetupContext) error {
	var config GetTopicConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: failed to decode setup configuration: %w", c.Name(), err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return fmt.Errorf("%s: invalid region: %w", c.Name(), err)
	}

	if _, err := requireTopicArn(config.TopicArn); err != nil {
		return fmt.Errorf("%s: invalid topic ARN: %w", c.Name(), err)
	}

	return nil
}

// ProcessQueueItem applies the default queue-item behavior.
func (c *GetTopic) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

// Execute retrieves topic data and emits it on the default output channel.
func (c *GetTopic) Execute(ctx core.ExecutionContext) error {
	var config GetTopicConfiguration
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

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("%s: failed to load AWS credentials from integration: %w", c.Name(), err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	topic, err := client.GetTopic(topicArn)
	if err != nil {
		return fmt.Errorf("%s: failed to get topic %q: %w", c.Name(), topicArn, err)
	}

	if err := ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.sns.topic", []any{topic}); err != nil {
		return fmt.Errorf("%s: failed to emit topic payload: %w", c.Name(), err)
	}

	return nil
}

// Actions returns supported custom actions.
func (c *GetTopic) Actions() []core.Action {
	return []core.Action{}
}

// HandleAction handles custom actions for this component.
func (c *GetTopic) HandleAction(ctx core.ActionContext) error {
	return nil
}

// HandleWebhook handles incoming webhook requests.
func (c *GetTopic) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

// Cancel handles execution cancellation.
func (c *GetTopic) Cancel(ctx core.ExecutionContext) error {
	return nil
}

// Cleanup handles component cleanup.
func (c *GetTopic) Cleanup(ctx core.SetupContext) error {
	return nil
}
