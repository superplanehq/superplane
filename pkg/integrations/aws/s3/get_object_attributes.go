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

var objectAttributeOptions = []configuration.FieldOption{
	{Label: "ETag", Value: "ETag"},
	{Label: "Checksum", Value: "Checksum"},
	{Label: "ObjectParts", Value: "ObjectParts"},
	{Label: "StorageClass", Value: "StorageClass"},
	{Label: "ObjectSize", Value: "ObjectSize"},
}

type GetObjectAttributes struct{}

type GetObjectAttributesConfiguration struct {
	Region     string   `json:"region" mapstructure:"region"`
	Bucket     string   `json:"bucket" mapstructure:"bucket"`
	Key        string   `json:"key" mapstructure:"key"`
	Attributes []string `json:"attributes" mapstructure:"attributes"`
}

func (c *GetObjectAttributes) Name() string {
	return "aws.s3.getObjectAttributes"
}

func (c *GetObjectAttributes) Label() string {
	return "S3 • Get Object Attributes"
}

func (c *GetObjectAttributes) Description() string {
	return "Retrieve specific attributes of an object in an AWS S3 bucket"
}

func (c *GetObjectAttributes) Documentation() string {
	return `The Get Object Attributes component retrieves specific attributes of an S3 object such as ETag, storage class, and size.

## Use Cases

- **Object validation**: Verify object integrity using ETag
- **Storage analysis**: Check object size and storage class
- **Workflow decisions**: Branch based on object attributes`
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
		regionField(),
		bucketField(),
		keyField("Object Key", "Key (path) of the object"),
		{
			Name:        "attributes",
			Label:       "Attributes",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    true,
			Description: "Attributes to retrieve for the object",
			Default:     []any{"ETag", "ObjectSize", "StorageClass"},
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "key",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: objectAttributeOptions,
				},
			},
		},
	}
}

func (c *GetObjectAttributes) Setup(ctx core.SetupContext) error {
	var config GetObjectAttributesConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode setup configuration: %w", err)
	}

	if _, err := requireRegion(config.Region); err != nil {
		return fmt.Errorf("invalid region: %w", err)
	}

	if _, err := requireBucket(config.Bucket); err != nil {
		return fmt.Errorf("invalid bucket: %w", err)
	}

	if _, err := requireKey(config.Key); err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}

	if len(config.Attributes) == 0 {
		return fmt.Errorf("at least one attribute is required")
	}

	return nil
}

func (c *GetObjectAttributes) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetObjectAttributes) Execute(ctx core.ExecutionContext) error {
	var config GetObjectAttributesConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode execution configuration: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return fmt.Errorf("invalid region: %w", err)
	}

	bucket, err := requireBucket(config.Bucket)
	if err != nil {
		return fmt.Errorf("invalid bucket: %w", err)
	}

	key, err := requireKey(config.Key)
	if err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials from integration: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	result, err := client.GetObjectAttributes(bucket, key, config.Attributes)
	if err != nil {
		return fmt.Errorf("failed to get object attributes for %q in bucket %q: %w", key, bucket, err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.s3.object.attributes", []any{result})
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
