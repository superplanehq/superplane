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

type ManageInstancePower struct{}

func (c *ManageInstancePower) Name() string {
	return "manageInstancePower"
}

func (c *ManageInstancePower) Label() string {
	return "Manage Instance Power"
}

func (c *ManageInstancePower) Description() string {
	return "Manage the power state of an OCI Compute instance (Start, Stop, Reset)"
}

func (c *ManageInstancePower) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "id",
			Label:       "Instance OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the instance to manage",
		},
		{
			Name:        "action",
			Label:       "Action",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The power action to perform (START, STOP, RESET, SOFTSTOP, SOFTRESET)",
			Options: []configuration.FieldOption{
				{Label: "Start", Value: "START"},
				{Label: "Stop", Value: "STOP"},
				{Label: "Reset", Value: "RESET"},
				{Label: "Soft Stop", Value: "SOFTSTOP"},
				{Label: "Soft Reset", Value: "SOFTRESET"},
			},
		},
	}
}

func (c *ManageInstancePower) Execute(ctx core.ExecutionContext) error {
	var input struct {
		ID     string `mapstructure:"id"`
		Action string `mapstructure:"action"`
	}

	if err := mapstructure.Decode(ctx.Configuration, &input); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	client, err := getClient(ctx)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get OCI client: %v", err))
	}

	_, err = client.InstanceAction(ctx.Context, ocicore.InstanceActionRequest{
		InstanceId: common.String(input.ID),
		Action:     common.String(input.Action),
	})
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to perform power action on OCI instance: %v", err))
	}

	return ctx.ExecutionState.Emit("success", "oci.instance.action", []any{map[string]interface{}{
		"id":     input.ID,
		"action": input.Action,
		"status": "requested",
	}})
}

func (c *ManageInstancePower) Setup(ctx core.SetupContext) error { return nil }
func (c *ManageInstancePower) Cleanup(ctx core.SetupContext) error { return nil }
func (c *ManageInstancePower) Actions() []core.Action { return nil }
func (c *ManageInstancePower) HandleAction(ctx core.ActionContext) error { return nil }
func (c *ManageInstancePower) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (c *ManageInstancePower) Cancel(ctx core.ExecutionContext) error { return nil }
