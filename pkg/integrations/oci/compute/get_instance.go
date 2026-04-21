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

type GetInstance struct{}
func (c *GetInstance) Name() string { return "getInstance" }
func (c *GetInstance) Label() string { return "Get Instance" }
func (c *GetInstance) Description() string { return "Retrieve details of an OCI Compute instance" }
func (c *GetInstance) Icon() string { return "oci" }
func (c *GetInstance) Color() string { return "#f30000" }
func (c *GetInstance) Documentation() string { return "Retrieve details of an OCI Compute instance" }
func (c *GetInstance) ExampleOutput() map[string]any {
	return map[string]any{"id": "ocid1.instance.oc1...", "state": "RUNNING"}
}
func (c *GetInstance) OutputChannels(any) []core.OutputChannel { return nil }

func (c *GetInstance) Configuration() []configuration.Field {
	return []configuration.Field{{Name: "id", Label: "Instance OCID", Type: configuration.FieldTypeString, Required: true}}
}
func (c *GetInstance) Execute(ctx core.ExecutionContext) error {
	var input struct { ID string `mapstructure:"id"` }
	if err := mapstructure.Decode(ctx.Configuration, &input); err != nil { return ctx.ExecutionState.Fail("error", err.Error()) }
	client, err := getClient(ctx)
	if err != nil { return ctx.ExecutionState.Fail("error", err.Error()) }
	resp, err := client.GetInstance(ctx.Context, ocicore.GetInstanceRequest{InstanceId: ocicommon.String(input.ID)})
	if err != nil { return ctx.ExecutionState.Fail("error", err.Error()) }
	return ctx.ExecutionState.Emit("success", "oci.instance", []any{map[string]interface{}{"id": *resp.Instance.Id, "state": string(resp.Instance.LifecycleState)}})
}
func (c *GetInstance) Setup(ctx core.SetupContext) error { return nil }
func (c *GetInstance) Cleanup(ctx core.SetupContext) error { return nil }
func (c *GetInstance) Actions() []core.Action { return nil }
func (c *GetInstance) HandleAction(ctx core.ActionContext) error { return nil }
func (c *GetInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) { return 200, nil, nil }
func (c *GetInstance) Cancel(ctx core.ExecutionContext) error { return nil }
