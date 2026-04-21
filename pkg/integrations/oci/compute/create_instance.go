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

type CreateInstance struct{}

func (c *CreateInstance) Name() string        { return "createInstance" }
func (c *CreateInstance) Label() string       { return "Create Instance" }
func (c *CreateInstance) Description() string { return "Creates a new Compute instance in OCI" }
func (c *CreateInstance) Icon() string        { return "oci" }
func (c *CreateInstance) Color() string       { return "#f30000" }
func (c *CreateInstance) Documentation() string {
	return "Creates a new OCI Compute instance. Requires Compartment OCID, Availability Domain, and Shape."
}

func (c *CreateInstance) ExampleOutput() map[string]any {
	return map[string]any{
		"type": "instance",
		"data": map[string]any{
			"id":          "ocid1.instance.oc1...",
			"displayName": "my-instance",
			"state":       "PROVISIONING",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

func (c *CreateInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "compartmentId",
			Label:       "Compartment OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the compartment",
		},
		{
			Name:        "availabilityDomain",
			Label:       "Availability Domain",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "e.g. Uocm:PHX-AD-1",
		},
		{
			Name:        "shape",
			Label:       "Shape",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "e.g. VM.Standard.E4.Flex",
		},
		{
			Name:     "displayName",
			Label:    "Display Name",
			Type:     configuration.FieldTypeString,
			Required: false,
		},
	}
}

func (c *CreateInstance) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return nil, nil
}

func (c *CreateInstance) Execute(ctx core.ExecutionContext) error {
	var input struct {
		CompartmentID      string `mapstructure:"compartmentId"`
		AvailabilityDomain string `mapstructure:"availabilityDomain"`
		Shape              string `mapstructure:"shape"`
		DisplayName        string `mapstructure:"displayName"`
	}

	if err := mapstructure.Decode(ctx.Data, &input); err != nil {
		return err
	}

	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	req := ocicore.LaunchInstanceRequest{
		LaunchInstanceDetails: ocicore.LaunchInstanceDetails{
			CompartmentId:      &input.CompartmentID,
			AvailabilityDomain: &input.AvailabilityDomain,
			Shape:              &input.Shape,
		},
	}

	if input.DisplayName != "" {
		req.DisplayName = &input.DisplayName
	}

	resp, err := client.LaunchInstance(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
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
		"state":       resp.Instance.LifecycleState,
	}

	return ctx.ExecutionState.Emit("default", "instance", []any{output})
}

func (c *CreateInstance) Actions() []core.Action {
	return nil
}

func (c *CreateInstance) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusNotImplemented, nil, errors.New("not implemented")
}

func (c *CreateInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}
