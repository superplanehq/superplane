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

type UpdateInstance struct{}

func (c *UpdateInstance) Name() string        { return "oci.updateInstance" }
func (c *UpdateInstance) Label() string       { return "Update Instance" }
func (c *UpdateInstance) Description() string { return "Updates a Compute instance in OCI" }
func (c *UpdateInstance) Icon() string        { return "oci" }
func (c *UpdateInstance) Color() string       { return "#f30000" }
func (c *UpdateInstance) Documentation() string {
	return "Updates the display name or other attributes of an OCI Compute instance."
}

func (c *UpdateInstance) ExampleOutput() map[string]any {
	return map[string]any{
		"type": "instance",
		"data": map[string]any{
			"id":          "ocid1.instance.oc1...",
			"displayName": "new-name",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

func (c *UpdateInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instanceId",
			Label:       "Instance OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the instance",
		},
		{
			Name:     "displayName",
			Label:    "Display Name",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
	}
}

func (c *UpdateInstance) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return nil, nil
}

func (c *UpdateInstance) Execute(ctx core.ExecutionContext) error {
	var input struct {
		InstanceID  string `mapstructure:"instanceId"`
		DisplayName string `mapstructure:"displayName"`
	}
	if err := mapstructure.Decode(ctx.Data, &input); err != nil {
		return err
	}

	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	req := ocicore.UpdateInstanceRequest{
		InstanceId: &input.InstanceID,
		UpdateInstanceDetails: ocicore.UpdateInstanceDetails{
			DisplayName: &input.DisplayName,
		},
	}

	resp, err := client.UpdateInstance(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	var id, displayName string
	if resp.Instance.Id != nil {
		id = *resp.Instance.Id
	}
	if resp.Instance.DisplayName != nil {
		displayName = *resp.Instance.DisplayName
	}

	output := map[string]any{
		"id":          id,
		"displayName": displayName,
	}

	return ctx.ExecutionState.Emit("default", "instance", []any{output})
}

func (c *UpdateInstance) Actions() []core.Action {
	return nil
}

func (c *UpdateInstance) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusNotImplemented, nil, errors.New("not implemented")
}

func (c *UpdateInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}
