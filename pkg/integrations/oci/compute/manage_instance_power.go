package compute

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ManageInstancePower struct{}

func (c *ManageInstancePower) Name() string        { return "manageInstancePower" }
func (c *ManageInstancePower) Label() string       { return "Manage Instance Power" }
func (c *ManageInstancePower) Description() string { return "START, STOP, or RESET an OCI instance" }
func (c *ManageInstancePower) Icon() string        { return "oci" }
func (c *ManageInstancePower) Color() string       { return "#f30000" }
func (c *ManageInstancePower) Documentation() string {
	return "Performs power actions on an OCI Compute instance (START, STOP, RESET, SOFTSTOP, SOFTRESET)."
}

func (c *ManageInstancePower) ExampleOutput() map[string]any {
	return map[string]any{
		"id":    "ocid1.instance.oc1...",
		"state": "STARTING",
	}
}

func (c *ManageInstancePower) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ManageInstancePower) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instanceId",
			Label:       "Instance OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the instance",
		},
		{
			Name:        "action",
			Label:       "Action",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "START, STOP, RESET, SOFTSTOP, SOFTRESET",
		},
	}
}

func (c *ManageInstancePower) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *ManageInstancePower) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return nil, nil
}

func (c *ManageInstancePower) Execute(ctx core.ExecutionContext) error {
	var input struct {
		InstanceID string `mapstructure:"instanceId"`
		Action     string `mapstructure:"action"`
	}
	if err := mapstructure.Decode(ctx.Data, &input); err != nil {
		return err
	}

	client, err := clientFactory(ctx)
	if err != nil {
		return err
	}

	req := ocicore.InstanceActionRequest{
		InstanceId: &input.InstanceID,
		Action:     ocicore.InstanceActionActionEnum(input.Action),
	}

	resp, err := client.InstanceAction(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to perform power action %s: %w", input.Action, err)
	}

	output := map[string]any{
		"id":    *resp.Instance.Id,
		"state": resp.Instance.LifecycleState,
	}

	return ctx.ExecutionState.Emit("default", "instance", []any{output})
}

func (c *ManageInstancePower) Actions() []core.Action {
	return nil
}

func (c *ManageInstancePower) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *ManageInstancePower) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusNotImplemented, nil, errors.New("not implemented")
}

func (c *ManageInstancePower) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ManageInstancePower) Cleanup(ctx core.SetupContext) error {
	return nil
}
