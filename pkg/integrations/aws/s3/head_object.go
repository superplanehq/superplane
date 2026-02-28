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
	return `The Head Object component retrieves metadata for an S3 object without downloading the object data.

## Use Cases

- **Existence checks**: Verify an object exists before processing
- **Metadata inspection**: Check content type, size, or last modified date
- **Conditional workflows**: Branch workflows based on object properties`
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
		regionField(),
		bucketField(),
		keyField("Object Key", "Key of the object to inspect"),
	}
}

func (c *HeadObject) Setup(ctx core.SetupContext) error {
	var config HeadObjectConfiguration
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

func (c *HeadObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *HeadObject) Execute(ctx core.ExecutionContext) error {
	var config HeadObjectConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode execution configuration: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials from integration: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, config.Region)
	metadata, err := client.HeadObject(config.Bucket, config.Key)
	if err != nil {
		return fmt.Errorf("failed to head object %q in bucket %q: %w", config.Key, config.Bucket, err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.s3.object", []any{metadata})
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
