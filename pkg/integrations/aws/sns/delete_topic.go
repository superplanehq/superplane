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

func (c *DeleteTopic) Name() string {
	return "aws.sns.deleteTopic"
}

func (c *DeleteTopic) Label() string {
	return "SNS • Delete Topic"
}

func (c *DeleteTopic) Description() string {
	return "Delete an AWS SNS topic"
}

func (c *DeleteTopic) Documentation() string {
	return `The Delete Topic component deletes an AWS SNS topic.

## Use Cases

- **Cleanup workflows**: Remove temporary topics after execution
- **Lifecycle management**: Decommission unused messaging resources
- **Rollback automation**: Remove topics created in failed provisioning runs`
}

func (c *DeleteTopic) Icon() string {
	return "aws"
}

func (c *DeleteTopic) Color() string {
	return "gray"
}

func (c *DeleteTopic) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteTopic) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		topicField(),
	}
}

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

func (c *DeleteTopic) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

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

func (c *DeleteTopic) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteTopic) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteTopic) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DeleteTopic) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteTopic) Cleanup(ctx core.SetupContext) error {
	return nil
}
