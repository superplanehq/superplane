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
	return "Retrieve attributes for an S3 object"
}

func (c *GetObjectAttributes) Documentation() string {
	return `The Get Object Attributes component retrieves attributes for an S3 object including ETag, storage class, and object size.

## Configuration

- **Region**: AWS region of the S3 bucket
- **Bucket**: Target S3 bucket
- **Key**: Object key to get attributes for`
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
		return fmt.Errorf("key is required")
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
	attributes := []string{"ETag", "StorageClass", "ObjectSize"}
	result, _, err := client.GetObjectAttributes(config.Bucket, config.Key, attributes)
	if err != nil {
		return fmt.Errorf("failed to get S3 object attributes: %w", err)
	}

	output := map[string]any{
		"bucket":       config.Bucket,
		"key":          config.Key,
		"etag":         result.ETag,
		"storageClass": result.StorageClass,
		"objectSize":   result.ObjectSize,
		"lastModified": result.LastModified,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.s3.object",
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
