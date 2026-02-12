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

// CreateTopic creates a new SNS topic.
type CreateTopic struct{}

func (c *CreateTopic) Name() string {
	return "aws.sns.createTopic"
}

func (c *CreateTopic) Label() string {
	return "SNS • Create Topic"
}

func (c *CreateTopic) Description() string {
	return "Create an AWS SNS topic"
}

func (c *CreateTopic) Documentation() string {
	return `The Create Topic component creates an AWS SNS topic and returns its metadata.

## Use Cases

- **Provisioning workflows**: Create topics as part of environment setup
- **Automation bootstrap**: Prepare topics before publishing messages
- **Self-service operations**: Provision messaging resources on demand`
}

func (c *CreateTopic) Icon() string {
	return "aws"
}

func (c *CreateTopic) Color() string {
	return "gray"
}

func (c *CreateTopic) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateTopic) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		{
			Name:        "name",
			Label:       "Topic Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name of the SNS topic to create",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:        "attributes",
			Label:       "Attributes",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Description: "Optional topic attributes as key-value pairs",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "name",
					Values: []string{"*"},
				},
			},
		},
	}
}

func (c *CreateTopic) Setup(ctx core.SetupContext) error {
	var config CreateTopicConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: failed to decode setup configuration: %w", c.Name(), err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return fmt.Errorf("%s: invalid region: %w", c.Name(), err)
	}

	if _, err := requireTopicName(config.Name); err != nil {
		return fmt.Errorf("%s: invalid topic name: %w", c.Name(), err)
	}

	return nil
}

func (c *CreateTopic) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateTopic) Execute(ctx core.ExecutionContext) error {
	var config CreateTopicConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: failed to decode execution configuration: %w", c.Name(), err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return fmt.Errorf("%s: invalid region: %w", c.Name(), err)
	}

	name, err := requireTopicName(config.Name)
	if err != nil {
		return fmt.Errorf("%s: invalid topic name: %w", c.Name(), err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("%s: failed to load AWS credentials from integration: %w", c.Name(), err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	topic, err := client.CreateTopic(name, mapAnyToStringMap(config.Attributes))
	if err != nil {
		return fmt.Errorf("%s: failed to create topic %q: %w", c.Name(), name, err)
	}

	if err := ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.sns.topic", []any{topic}); err != nil {
		return fmt.Errorf("%s: failed to emit created topic payload: %w", c.Name(), err)
	}

	return nil
}

func (c *CreateTopic) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateTopic) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateTopic) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateTopic) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateTopic) Cleanup(ctx core.SetupContext) error {
	return nil
}
