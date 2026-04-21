package compute

import (
	"context"
	"fmt"

	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteInstance struct{}

func (c *DeleteInstance) Name() string {
	return "deleteInstance"
}

func (c *DeleteInstance) Label() string {
	return "Delete Instance"
}

func (c *DeleteInstance) Description() string {
	return "Terminate an OCI Compute instance"
}

func (c *DeleteInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "id",
			Label:       "Instance OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the instance to terminate",
		},
	}
}

func (c *DeleteInstance) Execute(ctx core.ExecutionContext) error {
	var input struct {
		ID string `mapstructure:"id"`
	}

	if err := mapstructure.Decode(ctx.Configuration, &input); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	client, err := getClient(ctx)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get OCI client: %v", err))
	}

	_, err = client.TerminateInstance(ctx.Context, ocicore.TerminateInstanceRequest{
		InstanceId: common.String(input.ID),
	})
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to terminate OCI instance: %v", err))
	}

	return ctx.ExecutionState.Emit("success", "oci.instance.deleted", []any{map[string]interface{}{
		"id":     input.ID,
		"status": "termination_requested",
	}})
}

func (c *DeleteInstance) Setup(ctx core.SetupContext) error { return nil }
func (c *DeleteInstance) Cleanup(ctx core.SetupContext) error { return nil }
func (c *DeleteInstance) Actions() []core.Action { return nil }
func (c *DeleteInstance) HandleAction(ctx core.ActionContext) error { return nil }
func (c *DeleteInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (c *DeleteInstance) Cancel(ctx core.ExecutionContext) error { return nil }
