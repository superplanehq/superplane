package s3

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	PutObjectFormatJSON = "json"
	PutObjectFormatText = "text"
)

type PutObject struct{}

type PutObjectConfiguration struct {
	Region string  `json:"region" mapstructure:"region"`
	Bucket string  `json:"bucket" mapstructure:"bucket"`
	Key    string  `json:"key" mapstructure:"key"`
	Format string  `json:"format" mapstructure:"format"`
	JSON   *any    `json:"json" mapstructure:"json"`
	Text   *string `json:"text" mapstructure:"text"`
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
- **Key**: Object key
- **Format**: Content format (JSON or Text)
- **Content**: Object content to upload`
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
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-east-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: common.AllRegions,
				},
			},
		},
		{
			Name:        "bucket",
			Label:       "Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Target S3 bucket",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "s3.bucket",
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
		},
		{
			Name:        "key",
			Label:       "Object Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Key for the object in the bucket",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "bucket",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:     "format",
			Label:    "Content Format",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  PutObjectFormatJSON,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Value: PutObjectFormatJSON, Label: "JSON"},
						{Value: PutObjectFormatText, Label: "Text"},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "key",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:     "json",
			Label:    "JSON Content",
			Type:     configuration.FieldTypeObject,
			Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "key",
					Values: []string{"*"},
				},
				{
					Field:  "format",
					Values: []string{PutObjectFormatJSON},
				},
			},
		},
		{
			Name:     "text",
			Label:    "Text Content",
			Type:     configuration.FieldTypeText,
			Required: false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "key",
					Values: []string{"*"},
				},
				{
					Field:  "format",
					Values: []string{PutObjectFormatText},
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
		return fmt.Errorf("key is required")
	}

	if config.Format == "" {
		return fmt.Errorf("format is required")
	}

	if config.Format == PutObjectFormatJSON && config.JSON == nil {
		return fmt.Errorf("JSON content is required")
	}

	if config.Format == PutObjectFormatText && config.Text == nil {
		return fmt.Errorf("text content is required")
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

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if config.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}

	if config.Key == "" {
		return fmt.Errorf("key is required")
	}

	data, contentType, err := c.buildBody(config)
	if err != nil {
		return fmt.Errorf("failed to build object body: %w", err)
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	if err := client.PutObject(config.Bucket, config.Key, data, contentType); err != nil {
		return fmt.Errorf("failed to put S3 object: %w", err)
	}

	output := map[string]any{
		"bucket": config.Bucket,
		"key":    config.Key,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.s3.object",
		[]any{output},
	)
}

func (c *PutObject) buildBody(config PutObjectConfiguration) ([]byte, string, error) {
	if config.Format == PutObjectFormatText {
		if config.Text == nil || *config.Text == "" {
			return nil, "", fmt.Errorf("text content is required")
		}
		return []byte(*config.Text), "text/plain", nil
	}

	if config.JSON == nil {
		return nil, "", fmt.Errorf("JSON content is required")
	}

	data, err := json.Marshal(config.JSON)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal JSON content: %w", err)
	}

	return data, "application/json", nil
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
