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

type HeadObject struct{}

type HeadObjectConfiguration struct {
	Region string `json:"region" mapstructure:"region"`
	Bucket string `json:"bucket" mapstructure:"bucket"`
	Key    string `json:"key" mapstructure:"key"`
}

func (c *HeadObject) Name() string {
	return "aws.s3.headObject"
}

func (c *HeadObject) Label() string {
	return "S3 • Head Object"
}

func (c *HeadObject) Description() string {
	return "Retrieve metadata of an S3 object without downloading its content"
}

func (c *HeadObject) Documentation() string {
	return `The Head Object component retrieves metadata of an object in an AWS S3 bucket without downloading the object itself.

## Configuration

- **Region**: AWS region of the S3 bucket
- **Bucket**: Target S3 bucket
- **Object Key**: Key (path) of the object`
}

func (c *HeadObject) Icon() string {
	return "aws"
}

func (c *HeadObject) Color() string {
	return "gray"
}

func (c *HeadObject) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *HeadObject) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		bucketField(),
		objectKeyField(),
	}
}

func (c *HeadObject) Setup(ctx core.SetupContext) error {
	var config HeadObjectConfiguration
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

func (c *HeadObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *HeadObject) Execute(ctx core.ExecutionContext) error {
	var config HeadObjectConfiguration
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

	client := NewClient(ctx.HTTP, creds, config.Region)
	result, err := client.HeadObject(config.Bucket, config.Key)
	if err != nil {
		return fmt.Errorf("failed to head S3 object: %w", err)
	}

	output := map[string]any{
		"bucket":        config.Bucket,
		"key":           config.Key,
		"contentLength": result.ContentLength,
		"contentType":   result.ContentType,
		"etag":          result.ETag,
		"lastModified":  result.LastModified,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.s3.object.head",
		[]any{output},
	)
}

func (c *HeadObject) Actions() []core.Action {
	return []core.Action{}
}

func (c *HeadObject) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *HeadObject) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *HeadObject) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *HeadObject) Cleanup(ctx core.SetupContext) error {
	return nil
}
