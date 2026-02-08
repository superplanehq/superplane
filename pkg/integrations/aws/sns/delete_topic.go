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

// DeleteTopic deletes an existing SNS topic.
type DeleteTopic struct{}

// Name returns the component name.
func (c *DeleteTopic) Name() string {
	return "aws.sns.deleteTopic"
}

// Label returns the component label.
func (c *DeleteTopic) Label() string {
	return "SNS • Delete Topic"
}

// Description returns a short component description.
func (c *DeleteTopic) Description() string {
	return "Delete an AWS SNS topic"
}

// Documentation returns detailed Markdown documentation.
func (c *DeleteTopic) Documentation() string {
	return `The Delete Topic component deletes an AWS SNS topic.

## Use Cases

- **Cleanup workflows**: Remove temporary topics after execution
- **Lifecycle management**: Decommission unused messaging resources
- **Rollback automation**: Remove topics created in failed provisioning runs`
}

// Icon returns the icon slug.
func (c *DeleteTopic) Icon() string {
	return "aws"
}

// Color returns the component color.
func (c *DeleteTopic) Color() string {
	return "gray"
}

// OutputChannels declares the output channels used by this component.
func (c *DeleteTopic) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

// Configuration returns the component configuration schema.
func (c *DeleteTopic) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		topicField(),
	}
}

// Setup validates component configuration.
func (c *DeleteTopic) Setup(ctx core.SetupContext) error {
	var config DeleteTopicConfiguration
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
func (c *DeleteTopic) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

// Execute deletes a topic and emits a deletion payload.
func (c *DeleteTopic) Execute(ctx core.ExecutionContext) error {
	var config DeleteTopicConfiguration
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
	if err := client.DeleteTopic(topicArn); err != nil {
		return fmt.Errorf("%s: failed to delete topic %q: %w", c.Name(), topicArn, err)
	}

	if err := ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.sns.topic.deleted", []any{
		map[string]any{
			"topicArn": topicArn,
			"deleted":  true,
		},
	}); err != nil {
		return fmt.Errorf("%s: failed to emit topic deletion payload: %w", c.Name(), err)
	}

	return nil
}

// Actions returns supported custom actions.
func (c *DeleteTopic) Actions() []core.Action {
	return []core.Action{}
}

// HandleAction handles custom actions for this component.
func (c *DeleteTopic) HandleAction(ctx core.ActionContext) error {
	return nil
}

// HandleWebhook handles incoming webhook requests.
func (c *DeleteTopic) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

// Cancel handles execution cancellation.
func (c *DeleteTopic) Cancel(ctx core.ExecutionContext) error {
	return nil
}

// Cleanup handles component cleanup.
func (c *DeleteTopic) Cleanup(ctx core.SetupContext) error {
	return nil
}
