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

type CopyObject struct{}

type CopyObjectConfiguration struct {
	Region     string `json:"region" mapstructure:"region"`
	Bucket     string `json:"bucket" mapstructure:"bucket"`
	Key        string `json:"key" mapstructure:"key"`
	CopySource string `json:"copySource" mapstructure:"copySource"`
}

func (c *CopyObject) Name() string {
	return "aws.s3.copyObject"
}

func (c *CopyObject) Label() string {
	return "S3 • Copy Object"
}

func (c *CopyObject) Description() string {
	return "Copy an object from one S3 location to another"
}

func (c *CopyObject) Documentation() string {
	return `The Copy Object component copies an object from one S3 location to another.

## Configuration

- **Region**: AWS region of the destination S3 bucket
- **Bucket**: Destination S3 bucket
- **Key**: Destination object key
- **Copy Source**: Source object path in the format bucket/key`
}

func (c *CopyObject) Icon() string {
	return "aws"
}

func (c *CopyObject) Color() string {
	return "gray"
}

func (c *CopyObject) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CopyObject) Configuration() []configuration.Field {
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
			Label:       "Destination Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Destination S3 bucket",
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
			Label:       "Destination Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Object key in the destination bucket",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "bucket",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:        "copySource",
			Label:       "Copy Source",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Source object path (bucket/key)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "bucket",
					Values: []string{"*"},
				},
			},
		},
	}
}

func (c *CopyObject) Setup(ctx core.SetupContext) error {
	var config CopyObjectConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.Bucket = strings.TrimSpace(config.Bucket)
	config.Key = strings.TrimSpace(config.Key)
	config.CopySource = strings.TrimSpace(config.CopySource)

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if config.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}

	if config.Key == "" {
		return fmt.Errorf("key is required")
	}

	if config.CopySource == "" {
		return fmt.Errorf("copy source is required")
	}

	return nil
}

func (c *CopyObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CopyObject) Execute(ctx core.ExecutionContext) error {
	var config CopyObjectConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.Bucket = strings.TrimSpace(config.Bucket)
	config.Key = strings.TrimSpace(config.Key)
	config.CopySource = strings.TrimSpace(config.CopySource)

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if config.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}

	if config.Key == "" {
		return fmt.Errorf("key is required")
	}

	if config.CopySource == "" {
		return fmt.Errorf("copy source is required")
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	result, err := client.CopyObject(config.Bucket, config.Key, config.CopySource)
	if err != nil {
		return fmt.Errorf("failed to copy S3 object: %w", err)
	}

	output := map[string]any{
		"bucket":       config.Bucket,
		"key":          config.Key,
		"copySource":   config.CopySource,
		"etag":         result.ETag,
		"lastModified": result.LastModified,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.s3.object.copied",
		[]any{output},
	)
}

func (c *CopyObject) Actions() []core.Action {
	return []core.Action{}
}

func (c *CopyObject) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CopyObject) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CopyObject) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CopyObject) Cleanup(ctx core.SetupContext) error {
	return nil
}
