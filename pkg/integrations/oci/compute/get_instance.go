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

type GetInstance struct{}

func (c *GetInstance) Name() string        { return "getInstance" }
func (c *GetInstance) Label() string       { return "Get Instance" }
func (c *GetInstance) Description() string { return "Gets details of a Compute instance in OCI" }
func (c *GetInstance) Icon() string        { return "oci" }
func (c *GetInstance) Color() string       { return "#f30000" }
func (c *GetInstance) Documentation() string {
	return "Retrieves the details and current state of an OCI Compute instance."
}

func (c *GetInstance) ExampleOutput() map[string]any {
	return map[string]any{
		"type": "instance",
		"data": map[string]any{
			"id":          "ocid1.instance.oc1...",
			"displayName": "my-instance",
			"state":       "RUNNING",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

func (c *GetInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instanceId",
			Label:       "Instance OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the instance",
		},
	}
}

func (c *GetInstance) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *GetInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return nil, nil
}

func (c *GetInstance) Execute(ctx core.ExecutionContext) error {
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

	req := ocicore.GetInstanceRequest{
		InstanceId: &input.InstanceID,
	}

	resp, err := client.GetInstance(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	var id, displayName, region, shape string
	if resp.Instance.Id != nil {
		id = *resp.Instance.Id
	}
	if resp.Instance.DisplayName != nil {
		displayName = *resp.Instance.DisplayName
	}
	if resp.Instance.Region != nil {
		region = *resp.Instance.Region
	}
	if resp.Instance.Shape != nil {
		shape = *resp.Instance.Shape
	}

	output := map[string]any{
		"id":          id,
		"displayName": displayName,
		"state":       resp.Instance.LifecycleState,
		"region":      region,
		"shape":       shape,
	}

	return ctx.ExecutionState.Emit("default", "instance", []any{output})
}

func (c *GetInstance) Actions() []core.Action {
	return nil
}

func (c *GetInstance) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusNotImplemented, nil, errors.New("not implemented")
}

func (c *GetInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}
