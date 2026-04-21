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

type UpdateInstance struct{}
func (c *UpdateInstance) Name() string { return "updateInstance" }
func (c *UpdateInstance) Label() string { return "Update Instance" }
func (c *UpdateInstance) Description() string { return "Update an OCI Compute instance" }
func (c *UpdateInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "id", Label: "Instance OCID", Type: configuration.FieldTypeString, Required: true},
		{Name: "displayName", Label: "Display Name", Type: configuration.FieldTypeString, Required: true},
	}
}
func (c *UpdateInstance) Execute(ctx core.ExecutionContext) error {
	var input struct { ID string `mapstructure:"id"`; DisplayName string `mapstructure:"displayName"` }
	if err := mapstructure.Decode(ctx.Configuration, &input); err != nil { return ctx.ExecutionState.Fail("error", err.Error()) }
	client, err := getClient(ctx)
	if err != nil { return ctx.ExecutionState.Fail("error", err.Error()) }
	_, err = client.UpdateInstance(ctx.Context, ocicore.UpdateInstanceRequest{
		InstanceId: ocicommon.String(input.ID),
		UpdateInstanceDetails: ocicore.UpdateInstanceDetails{DisplayName: ocicommon.String(input.DisplayName)},
	})
	if err != nil { return ctx.ExecutionState.Fail("error", err.Error()) }
	return ctx.ExecutionState.Emit("success", "oci.instance.updated", []any{map[string]interface{}{"id": input.ID}})
}
func (c *UpdateInstance) Setup(ctx core.SetupContext) error { return nil }
func (c *UpdateInstance) Cleanup(ctx core.SetupContext) error { return nil }
func (c *UpdateInstance) Actions() []core.Action { return nil }
func (c *UpdateInstance) HandleAction(ctx core.ActionContext) error { return nil }
func (c *UpdateInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) { return 200, nil, nil }
func (c *UpdateInstance) Cancel(ctx core.ExecutionContext) error { return nil }
