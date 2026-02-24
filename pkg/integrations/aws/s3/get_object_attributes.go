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

type GetObjectAttributes struct{}

type GetObjectAttributesConfiguration struct {
	Region string `json:"region" mapstructure:"region"`
	Bucket string `json:"bucket" mapstructure:"bucket"`
	Key    string `json:"key" mapstructure:"key"`
}

func (c *GetObjectAttributes) Name() string {
	return "aws.s3.getObjectAttributes"
}

func (c *GetObjectAttributes) Label() string {
	return "S3 • Get Object Attributes"
}

func (c *GetObjectAttributes) Description() string {
	return "Retrieve attributes of an S3 object"
}

func (c *GetObjectAttributes) Documentation() string {
	return `The Get Object Attributes component retrieves attributes of an object in an AWS S3 bucket.

## Configuration

- **Region**: AWS region of the S3 bucket
- **Bucket**: Target S3 bucket
- **Object Key**: Key (path) of the object`
}

func (c *GetObjectAttributes) Icon() string {
	return "aws"
}

func (c *GetObjectAttributes) Color() string {
	return "gray"
}

func (c *GetObjectAttributes) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetObjectAttributes) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		bucketField(),
		objectKeyField(),
	}
}

func (c *GetObjectAttributes) Setup(ctx core.SetupContext) error {
	var config GetObjectAttributesConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.Bucket = strings.TrimSpace(config.Bucket)
	config.Key = strings.TrimSpace(config.Key)

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if config.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}

	if config.Key == "" {
		return fmt.Errorf("object key is required")
	}

	return nil
}

func (c *GetObjectAttributes) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetObjectAttributes) Execute(ctx core.ExecutionContext) error {
	var config GetObjectAttributesConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.Bucket = strings.TrimSpace(config.Bucket)
	config.Key = strings.TrimSpace(config.Key)

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	attributes := []string{"ETag", "ObjectSize", "StorageClass"}
	client := NewClient(ctx.HTTP, creds, config.Region)
	result, err := client.GetObjectAttributes(config.Bucket, config.Key, attributes)
	if err != nil {
		return fmt.Errorf("failed to get S3 object attributes: %w", err)
	}

	output := map[string]any{
		"bucket":       config.Bucket,
		"key":          config.Key,
		"etag":         result.ETag,
		"objectSize":   result.ObjectSize,
		"storageClass": result.StorageClass,
		"lastModified": result.LastModified,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.s3.object.attributes",
		[]any{output},
	)
}

func (c *GetObjectAttributes) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetObjectAttributes) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetObjectAttributes) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetObjectAttributes) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetObjectAttributes) Cleanup(ctx core.SetupContext) error {
	return nil
}
