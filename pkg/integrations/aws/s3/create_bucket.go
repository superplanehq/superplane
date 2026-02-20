package s3

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

type CreateBucket struct{}

type CreateBucketConfiguration struct {
	Region string `json:"region" mapstructure:"region"`
	Name   string `json:"name" mapstructure:"name"`
}

func (c *CreateBucket) Name() string {
	return "aws.s3.createBucket"
}

func (c *CreateBucket) Label() string {
	return "S3 • Create Bucket"
}

func (c *CreateBucket) Description() string {
	return "Create an AWS S3 bucket"
}

func (c *CreateBucket) Documentation() string {
	return `The Create Bucket component creates an AWS S3 bucket and returns its metadata.

## Use Cases

- **Provisioning workflows**: Create buckets as part of environment setup
- **Automation bootstrap**: Prepare storage resources before uploading objects
- **Self-service operations**: Provision storage on demand`
}

func (c *CreateBucket) Icon() string {
	return "aws"
}

func (c *CreateBucket) Color() string {
	return "gray"
}

func (c *CreateBucket) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateBucket) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		{
			Name:        "name",
			Label:       "Bucket Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name of the S3 bucket to create",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
		},
	}
}

func (c *CreateBucket) Setup(ctx core.SetupContext) error {
	var config CreateBucketConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode setup configuration: %w", err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return fmt.Errorf("invalid region: %w", err)
	}

	name := strings.TrimSpace(config.Name)
	if name == "" {
		return fmt.Errorf("bucket name is required")
	}

	return nil
}

func (c *CreateBucket) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateBucket) Execute(ctx core.ExecutionContext) error {
	var config CreateBucketConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode execution configuration: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials from integration: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, config.Region)
	bucket, err := client.CreateBucket(config.Name)
	if err != nil {
		return fmt.Errorf("failed to create bucket %q: %w", config.Name, err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.s3.bucket", []any{bucket})
}

func (c *CreateBucket) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateBucket) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateBucket) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateBucket) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateBucket) Cleanup(ctx core.SetupContext) error {
	return nil
}
