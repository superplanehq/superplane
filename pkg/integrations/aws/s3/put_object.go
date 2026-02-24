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
	ContentType string `json:"contentType" mapstructure:"contentType"`
	Body        string `json:"body" mapstructure:"body"`
}

func (c *PutObject) Name() string {
	return "aws.s3.putObject"
}

func (c *PutObject) Label() string {
	return "S3 • Put Object"
}

func (c *PutObject) Description() string {
	return "Upload an object to an S3 bucket"
}

func (c *PutObject) Documentation() string {
	return `The Put Object component uploads an object to an AWS S3 bucket.

## Configuration

- **Region**: AWS region of the S3 bucket
- **Bucket**: Target S3 bucket
- **Object Key**: Key (path) for the object
- **Content Type**: MIME type of the content (e.g. application/json, text/plain)
- **Body**: Content to upload`
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
		objectKeyField(),
		{
			Name:        "contentType",
			Label:       "Content Type",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "application/octet-stream",
			Description: "MIME type of the content",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "key",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "Content to upload",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "key",
					Values: []string{"*"},
				},
			},
		},
	}
}

func (c *PutObject) Setup(ctx core.SetupContext) error {
	var config PutObjectConfiguration
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

func (c *PutObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PutObject) Execute(ctx core.ExecutionContext) error {
	var config PutObjectConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.Bucket = strings.TrimSpace(config.Bucket)
	config.Key = strings.TrimSpace(config.Key)
	config.ContentType = strings.TrimSpace(config.ContentType)

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	result, err := client.PutObject(config.Bucket, config.Key, config.ContentType, []byte(config.Body))
	if err != nil {
		return fmt.Errorf("failed to put S3 object: %w", err)
	}

	output := map[string]any{
		"bucket": config.Bucket,
		"key":    config.Key,
		"etag":   result.ETag,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.s3.object",
		[]any{output},
	)
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
