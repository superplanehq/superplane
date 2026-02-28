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
	Region      string  `json:"region" mapstructure:"region"`
	Bucket      string  `json:"bucket" mapstructure:"bucket"`
	Key         string  `json:"key" mapstructure:"key"`
	Content     string  `json:"content" mapstructure:"content"`
	ContentType *string `json:"contentType,omitempty" mapstructure:"contentType"`
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
	return `The Put Object component uploads content to an S3 object.

## Use Cases

- **Artifact publishing**: Upload build artifacts, reports, or configurations
- **Data pipeline output**: Write processed data to S3
- **Configuration management**: Store configuration files in S3`
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
		keyField("Object Key", "Key for the object to upload"),
		{
			Name:        "content",
			Label:       "Content",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "Content to upload as the object body",
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
			Description: "MIME type for the object (e.g. application/json, text/plain)",
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
		return fmt.Errorf("failed to decode setup configuration: %w", err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return fmt.Errorf("invalid region: %w", err)
	}

	if _, err := requireBucket(config.Bucket); err != nil {
		return err
	}

	if _, err := requireKey(config.Key); err != nil {
		return err
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

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials from integration: %w", err)
	}

	contentType := "application/octet-stream"
	if config.ContentType != nil && strings.TrimSpace(*config.ContentType) != "" {
		contentType = strings.TrimSpace(*config.ContentType)
	}

	client := NewClient(ctx.HTTP, credentials, config.Region)
	result, err := client.PutObject(config.Bucket, config.Key, []byte(config.Content), contentType)
	if err != nil {
		return fmt.Errorf("failed to put object %q in bucket %q: %w", config.Key, config.Bucket, err)
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
