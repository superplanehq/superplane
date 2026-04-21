package compute

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteInstance struct{}

func (c *DeleteInstance) Name() string        { return "deleteInstance" }
func (c *DeleteInstance) Label() string       { return "Delete Instance" }
func (c *DeleteInstance) Description() string { return "Terminates a Compute instance in OCI" }
func (c *DeleteInstance) Icon() string        { return "oci" }
func (c *DeleteInstance) Color() string       { return "#f30000" }
func (c *DeleteInstance) Documentation() string {
	return "Permanently terminates the specified OCI Compute instance."
}

func (c *DeleteInstance) ExampleOutput() map[string]any {
	return map[string]any{
		"type": "status",
		"data": map[string]any{
			"status": "terminated",
		},
		"timestamp": time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
	}
}

func (c *DeleteInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instanceId",
			Label:       "Instance OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the instance to delete",
		},
	}
}

func (c *DeleteInstance) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return nil, nil
}

func (c *DeleteInstance) Execute(ctx core.ExecutionContext) error {
	var input struct {
		InstanceID string `mapstructure:"instanceId"`
	}
	if err := mapstructure.Decode(ctx.Data, &input); err != nil {
		return err
	}

	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	req := ocicore.TerminateInstanceRequest{
		InstanceId: &input.InstanceID,
	}

	_, err = client.TerminateInstance(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	output := map[string]any{
		"status": "terminated",
	}

	return ctx.ExecutionState.Emit("default", "status", []any{output})
}

func (c *DeleteInstance) Actions() []core.Action {
	return nil
}

func (c *DeleteInstance) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusNotImplemented, nil, errors.New("not implemented")
}

func (c *DeleteInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}
