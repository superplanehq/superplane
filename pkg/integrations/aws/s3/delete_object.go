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
	return "Delete an object from an AWS S3 bucket"
}

func (c *DeleteObject) Documentation() string {
	return `The Delete Object component deletes an object from an AWS S3 bucket.

## Use Cases

- **Cleanup workflows**: Remove temporary files after processing
- **Data lifecycle management**: Delete expired or obsolete objects
- **Rollback automation**: Remove artifacts from failed deployments`
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
		regionField(),
		bucketField(),
		keyField("Object Key", "Key of the object to delete"),
	}
}

func (c *DeleteObject) Setup(ctx core.SetupContext) error {
	var config DeleteObjectConfiguration
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

func (c *DeleteObject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteObject) Execute(ctx core.ExecutionContext) error {
	var config DeleteObjectConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode execution configuration: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials from integration: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, config.Region)
	if err := client.DeleteObject(config.Bucket, config.Key); err != nil {
		return fmt.Errorf("failed to delete object %q from bucket %q: %w", config.Key, config.Bucket, err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "aws.s3.object.deleted", []any{
		map[string]any{
			"bucket":  config.Bucket,
			"key":     config.Key,
			"deleted": true,
		},
	})
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
