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

type DeleteObject struct{}

type DeleteObjectConfiguration struct {
	Region string `json:"region" mapstructure:"region"`
	Bucket string `json:"bucket" mapstructure:"bucket"`
	Key    string `json:"key" mapstructure:"key"`
}

func (c *DeleteObject) Name() string {
	return "aws.s3.deleteObject"
}

func (c *DeleteObject) Label() string {
	return "S3 • Delete Object"
}

func (c *DeleteObject) Description() string {
	return "Delete an object from an S3 bucket"
}

func (c *DeleteObject) Documentation() string {
	return `The Delete Object component deletes an object from an AWS S3 bucket.

## Configuration

- **Region**: AWS region of the S3 bucket
- **Bucket**: Target S3 bucket
- **Key**: Object key to delete`
}

func (c *DeleteObject) Icon() string {
	return "aws"
}

func (c *DeleteObject) Color() string {
	return "gray"
}

func (c *DeleteObject) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteObject) Configuration() []configuration.Field {
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
			Description: "Key of the object to delete",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "bucket",
					Values: []string{"*"},
				},
			},
		},
	}
}

func (c *DeleteObject) Setup(ctx core.SetupContext) error {
	var config DeleteObjectConfiguration
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

func (c *DeleteObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteObject) Execute(ctx core.ExecutionContext) error {
	var config DeleteObjectConfiguration
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
	if err := client.DeleteObject(config.Bucket, config.Key); err != nil {
		return fmt.Errorf("failed to delete S3 object: %w", err)
	}

	output := map[string]any{
		"bucket":  config.Bucket,
		"key":     config.Key,
		"deleted": true,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.s3.object.deleted",
		[]any{output},
	)
}

func (c *DeleteObject) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteObject) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteObject) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DeleteObject) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteObject) Cleanup(ctx core.SetupContext) error {
	return nil
}
