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
	Region            string `json:"region" mapstructure:"region"`
	SourceBucket      string `json:"sourceBucket" mapstructure:"sourceBucket"`
	SourceKey         string `json:"sourceKey" mapstructure:"sourceKey"`
	DestinationBucket string `json:"destinationBucket" mapstructure:"destinationBucket"`
	DestinationKey    string `json:"destinationKey" mapstructure:"destinationKey"`
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
	return `The Copy Object component copies an object from one S3 location to another within the same region.

## Configuration

- **Region**: AWS region of the S3 buckets
- **Source Bucket**: Bucket containing the source object
- **Source Key**: Key (path) of the source object
- **Destination Bucket**: Bucket to copy the object to
- **Destination Key**: Key (path) for the copied object`
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
		regionField(),
		{
			Name:        "sourceBucket",
			Label:       "Source Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Bucket containing the source object",
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
			Name:        "sourceKey",
			Label:       "Source Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Key (path) of the source object",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "sourceBucket",
					Values: []string{"*"},
				},
			},
		},
		{
			Name:        "destinationBucket",
			Label:       "Destination Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Bucket to copy the object to",
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
			Name:        "destinationKey",
			Label:       "Destination Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Key (path) for the copied object",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "destinationBucket",
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

	config = c.normalizeConfig(config)

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	if config.SourceBucket == "" {
		return fmt.Errorf("source bucket is required")
	}

	if config.SourceKey == "" {
		return fmt.Errorf("source key is required")
	}

	if config.DestinationBucket == "" {
		return fmt.Errorf("destination bucket is required")
	}

	if config.DestinationKey == "" {
		return fmt.Errorf("destination key is required")
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

	config = c.normalizeConfig(config)

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	result, err := client.CopyObject(config.SourceBucket, config.SourceKey, config.DestinationBucket, config.DestinationKey)
	if err != nil {
		return fmt.Errorf("failed to copy S3 object: %w", err)
	}

	output := map[string]any{
		"sourceBucket":      config.SourceBucket,
		"sourceKey":         config.SourceKey,
		"destinationBucket": config.DestinationBucket,
		"destinationKey":    config.DestinationKey,
		"etag":              result.ETag,
		"lastModified":      result.LastModified,
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

func (c *CopyObject) normalizeConfig(config CopyObjectConfiguration) CopyObjectConfiguration {
	config.Region = strings.TrimSpace(config.Region)
	config.SourceBucket = strings.TrimSpace(config.SourceBucket)
	config.SourceKey = strings.TrimSpace(config.SourceKey)
	config.DestinationBucket = strings.TrimSpace(config.DestinationBucket)
	config.DestinationKey = strings.TrimSpace(config.DestinationKey)
	return config
}
