package compute

import (
	"context"
	"fmt"

	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	ocicommon "github.com/oracle/oci-go-sdk/v65/common"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ManageInstancePower struct{}
func (c *ManageInstancePower) Name() string { return "manageInstancePower" }
func (c *ManageInstancePower) Label() string { return "Manage Instance Power" }
func (c *ManageInstancePower) Description() string { return "Manage the power state of an OCI Compute instance" }
func (c *ManageInstancePower) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "id", Label: "Instance OCID", Type: configuration.FieldTypeString, Required: true},
		{Name: "action", Label: "Action", Type: configuration.FieldTypeString, Required: true},
	}
}
func (c *ManageInstancePower) Execute(ctx core.ExecutionContext) error {
	var input struct { ID string `mapstructure:"id"`; Action string `mapstructure:"action"` }
	if err := mapstructure.Decode(ctx.Configuration, &input); err != nil { return ctx.ExecutionState.Fail("error", err.Error()) }
	client, err := getClient(ctx)
	if err != nil { return ctx.ExecutionState.Fail("error", err.Error()) }
	_, err = client.InstanceAction(ctx.Context, ocicore.InstanceActionRequest{
		InstanceId: ocicommon.String(input.ID),
		Action:     ocicommon.String(input.Action),
	})
	if err != nil { return ctx.ExecutionState.Fail("error", err.Error()) }
	return ctx.ExecutionState.Emit("success", "oci.instance.action", []any{map[string]interface{}{"id": input.ID, "action": input.Action}})
}
func (c *ManageInstancePower) Setup(ctx core.SetupContext) error { return nil }
func (c *ManageInstancePower) Cleanup(ctx core.SetupContext) error { return nil }
func (c *ManageInstancePower) Actions() []core.Action { return nil }
func (c *ManageInstancePower) HandleAction(ctx core.ActionContext) error { return nil }
func (c *ManageInstancePower) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) { return 200, nil, nil }
func (c *ManageInstancePower) Cancel(ctx core.ExecutionContext) error { return nil }
