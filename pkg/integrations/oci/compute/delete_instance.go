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

type DeleteInstance struct{}
func (c *DeleteInstance) Name() string { return "deleteInstance" }
func (c *DeleteInstance) Label() string { return "Delete Instance" }
func (c *DeleteInstance) Description() string { return "Terminate an OCI Compute instance" }
func (c *DeleteInstance) Icon() string { return "oci" }
func (c *DeleteInstance) Color() string { return "#f30000" }
func (c *DeleteInstance) Documentation() string { return "Terminate an OCI Compute instance" }
func (c *DeleteInstance) ExampleOutput() map[string]any {
	return map[string]any{"id": "ocid1.instance.oc1...", "status": "termination_requested"}
}
func (c *DeleteInstance) OutputChannels(any) []core.OutputChannel { return nil }

func (c *DeleteInstance) Configuration() []configuration.Field {
	return []configuration.Field{{Name: "id", Label: "Instance OCID", Type: configuration.FieldTypeString, Required: true}}
}
func (c *DeleteInstance) Execute(ctx core.ExecutionContext) error {
	var input struct { ID string `mapstructure:"id"` }
	if err := mapstructure.Decode(ctx.Configuration, &input); err != nil { return ctx.ExecutionState.Fail("error", err.Error()) }
	client, err := getClient(ctx)
	if err != nil { return ctx.ExecutionState.Fail("error", err.Error()) }
	_, err = client.TerminateInstance(ctx.Context, ocicore.TerminateInstanceRequest{InstanceId: ocicommon.String(input.ID)})
	if err != nil { return ctx.ExecutionState.Fail("error", err.Error()) }
	return ctx.ExecutionState.Emit("success", "oci.instance.deleted", []any{map[string]interface{}{"id": input.ID}})
}
func (c *DeleteInstance) Setup(ctx core.SetupContext) error { return nil }
func (c *DeleteInstance) Cleanup(ctx core.SetupContext) error { return nil }
func (c *DeleteInstance) Actions() []core.Action { return nil }
func (c *DeleteInstance) HandleAction(ctx core.ActionContext) error { return nil }
func (c *DeleteInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) { return 200, nil, nil }
func (c *DeleteInstance) Cancel(ctx core.ExecutionContext) error { return nil }
