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
	return `The Get Object Attributes component retrieves attributes such as ETag, storage class, and size for an S3 object.

## Use Cases

- **Integrity checks**: Verify object ETag after uploads or copies
- **Storage audits**: Inspect object storage class and size
- **Workflow enrichment**: Load object attributes before downstream processing`
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
		keyField("Object Key", "Key of the object to inspect"),
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
		return err
	}

	if _, err := requireKey(config.Key); err != nil {
		return err
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

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials from integration: %w", err)
	}

	attributes := []string{"ETag", "StorageClass", "ObjectSize"}
	client := NewClient(ctx.HTTP, credentials, config.Region)
	result, err := client.GetObjectAttributes(config.Bucket, config.Key, attributes)
	if err != nil {
		return fmt.Errorf("failed to get attributes for object %q in bucket %q: %w", config.Key, config.Bucket, err)
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
