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

type GetTopic struct{}

type GetTopicConfiguration struct {
	Region   string `json:"region" mapstructure:"region"`
	TopicArn string `json:"topicArn" mapstructure:"topicArn"`
}

func (c *GetTopic) Name() string {
	return "aws.sns.getTopic"
}

func (c *GetTopic) Label() string {
	return "SNS • Get Topic"
}

func (c *GetTopic) Description() string {
	return "Get an AWS SNS topic by ARN"
}

func (c *GetTopic) Documentation() string {
	return `The Get Topic component retrieves metadata and attributes for an AWS SNS topic.

## Use Cases

- **Configuration audits**: Verify topic settings and attributes
- **Workflow enrichment**: Load topic metadata before downstream actions
- **Validation**: Confirm topic existence and ownership`
}

func (c *GetTopic) Icon() string {
	return "aws"
}

func (c *GetTopic) Color() string {
	return "gray"
}

func (c *GetTopic) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetTopic) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		topicField(),
	}
}

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

func (c *GetTopic) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

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

func (c *GetTopic) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetTopic) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetTopic) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetTopic) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetTopic) Cleanup(ctx core.SetupContext) error {
	return nil
}
