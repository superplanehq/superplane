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
	return "Retrieve metadata for an S3 object without downloading it"
}

func (c *HeadObject) Documentation() string {
	return `The Head Object component retrieves metadata for an S3 object without downloading it.

## Configuration

- **Region**: AWS region of the S3 bucket
- **Bucket**: Target S3 bucket
- **Key**: Object key to inspect`
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
			Description: "Key of the object",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "bucket",
					Values: []string{"*"},
				},
			},
		},
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
		return fmt.Errorf("key is required")
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

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if config.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}

	if config.Key == "" {
		return fmt.Errorf("key is required")
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	headers, err := client.HeadObject(config.Bucket, config.Key)
	if err != nil {
		return fmt.Errorf("failed to head S3 object: %w", err)
	}

	output := map[string]any{
		"bucket":        config.Bucket,
		"key":           config.Key,
		"contentType":   headers.Get("Content-Type"),
		"contentLength": headers.Get("Content-Length"),
		"etag":          headers.Get("ETag"),
		"lastModified":  headers.Get("Last-Modified"),
		"storageClass":  headers.Get("X-Amz-Storage-Class"),
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.s3.object",
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
