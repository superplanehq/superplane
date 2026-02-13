package sns

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type CreateTopic struct{}

type CreateTopicConfiguration struct {
	Region string `json:"region" mapstructure:"region"`
	Name   string `json:"name" mapstructure:"name"`
}

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
	}
}

func (c *CreateTopic) Setup(ctx core.SetupContext) error {
	var config CreateTopicConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode setup configuration: %w", err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return fmt.Errorf("invalid region: %w", err)
	}

	name := strings.TrimSpace(config.Name)
	if name == "" {
		return fmt.Errorf("topic name is required")
	}

	return nil
}

func (c *CreateTopic) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateTopic) Execute(ctx core.ExecutionContext) error {
	var config CreateTopicConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode execution configuration: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials from integration: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, config.Region)
	topic, err := client.CreateTopic(config.Name)
	if err != nil {
		return fmt.Errorf("failed to create topic %q: %w", config.Name, err)
	}

	if err := ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.sns.topic", []any{topic}); err != nil {
		return fmt.Errorf("failed to emit created topic payload: %w", err)
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
