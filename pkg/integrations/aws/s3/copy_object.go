package s3

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type CopyObject struct{}

type CopyObjectConfiguration struct {
	Region       string `json:"region" mapstructure:"region"`
	SourceBucket string `json:"sourceBucket" mapstructure:"sourceBucket"`
	SourceKey    string `json:"sourceKey" mapstructure:"sourceKey"`
	Bucket       string `json:"bucket" mapstructure:"bucket"`
	Key          string `json:"key" mapstructure:"key"`
}

func (c *CopyObject) Name() string {
	return "aws.s3.copyObject"
}

func (c *CopyObject) Label() string {
	return "S3 • Copy Object"
}

func (c *CopyObject) Description() string {
	return "Copy an object within or between AWS S3 buckets"
}

func (c *CopyObject) Documentation() string {
	return `The Copy Object component copies an object from one S3 location to another within the same region.

## Use Cases

- **Data replication**: Copy objects between buckets for backup or distribution
- **File organization**: Move objects to different key paths
- **Workflow pipelines**: Promote artifacts between staging and production buckets`
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
			Description: "Name of the source S3 bucket",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "s3.bucket",
					UseNameAsValue: true,
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
			Name:        "bucket",
			Label:       "Destination Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Name of the destination S3 bucket",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "sourceKey",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "s3.bucket",
					UseNameAsValue: true,
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
			Description: "Key (path) for the destination object",
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
		return fmt.Errorf("failed to decode setup configuration: %w", err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return fmt.Errorf("invalid region: %w", err)
	}

	if _, err := requireBucket(config.SourceBucket); err != nil {
		return fmt.Errorf("invalid source bucket: %w", err)
	}

	if _, err := requireKey(config.SourceKey); err != nil {
		return fmt.Errorf("invalid source key: %w", err)
	}

	if _, err := requireBucket(config.Bucket); err != nil {
		return fmt.Errorf("invalid destination bucket: %w", err)
	}

	if _, err := requireKey(config.Key); err != nil {
		return fmt.Errorf("invalid destination key: %w", err)
	}

	return nil
}

func (c *CopyObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CopyObject) Execute(ctx core.ExecutionContext) error {
	var config CopyObjectConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode execution configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return fmt.Errorf("invalid region: %w", err)
	}

	sourceBucket, err := requireBucket(config.SourceBucket)
	if err != nil {
		return fmt.Errorf("invalid source bucket: %w", err)
	}

	sourceKey, err := requireKey(config.SourceKey)
	if err != nil {
		return fmt.Errorf("invalid source key: %w", err)
	}

	destBucket, err := requireBucket(config.Bucket)
	if err != nil {
		return fmt.Errorf("invalid destination bucket: %w", err)
	}

	destKey, err := requireKey(config.Key)
	if err != nil {
		return fmt.Errorf("invalid destination key: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials from integration: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	result, err := client.CopyObject(sourceBucket, sourceKey, destBucket, destKey)
	if err != nil {
		return fmt.Errorf("failed to copy object from %s/%s to %s/%s: %w", sourceBucket, sourceKey, destBucket, destKey, err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.s3.object.copied", []any{result})
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
