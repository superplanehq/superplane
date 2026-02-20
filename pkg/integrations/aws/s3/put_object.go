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

type PutObject struct{}

type PutObjectConfiguration struct {
	Region      string `json:"region" mapstructure:"region"`
	Bucket      string `json:"bucket" mapstructure:"bucket"`
	Key         string `json:"key" mapstructure:"key"`
	Body        string `json:"body" mapstructure:"body"`
	ContentType string `json:"contentType" mapstructure:"contentType"`
}

func (c *PutObject) Name() string {
	return "aws.s3.putObject"
}

func (c *PutObject) Label() string {
	return "S3 • Put Object"
}

func (c *PutObject) Description() string {
	return "Upload an object to an AWS S3 bucket"
}

func (c *PutObject) Documentation() string {
	return `The Put Object component uploads an object to an AWS S3 bucket.

## Use Cases

- **Data storage**: Store workflow outputs in S3
- **Configuration management**: Upload configuration files to buckets
- **Artifact publishing**: Publish build artifacts to S3 storage`
}

func (c *PutObject) Icon() string {
	return "aws"
}

func (c *PutObject) Color() string {
	return "gray"
}

func (c *PutObject) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PutObject) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		bucketField(),
		keyField("Object Key", "Key (path) for the object in the bucket"),
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "Content of the object to upload",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "key",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:        "contentType",
			Label:       "Content Type",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Default:     "application/octet-stream",
			Description: "MIME type of the object",
		},
	}
}

func (c *PutObject) Setup(ctx core.SetupContext) error {
	var config PutObjectConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode setup configuration: %w", err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return fmt.Errorf("invalid region: %w", err)
	}

	if _, err := requireBucket(config.Bucket); err != nil {
		return fmt.Errorf("invalid bucket: %w", err)
	}

	if _, err := requireKey(config.Key); err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}

	return nil
}

func (c *PutObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PutObject) Execute(ctx core.ExecutionContext) error {
	var config PutObjectConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode execution configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return fmt.Errorf("invalid region: %w", err)
	}

	bucket, err := requireBucket(config.Bucket)
	if err != nil {
		return fmt.Errorf("invalid bucket: %w", err)
	}

	key, err := requireKey(config.Key)
	if err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials from integration: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	result, err := client.PutObject(bucket, key, []byte(config.Body), strings.TrimSpace(config.ContentType))
	if err != nil {
		return fmt.Errorf("failed to put object %q in bucket %q: %w", key, bucket, err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.s3.object", []any{result})
}

func (c *PutObject) Actions() []core.Action {
	return []core.Action{}
}

func (c *PutObject) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *PutObject) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *PutObject) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *PutObject) Cleanup(ctx core.SetupContext) error {
	return nil
}
