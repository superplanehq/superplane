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

type CreateInstance struct{}

func (c *CreateInstance) Name() string { return "createInstance" }
func (c *CreateInstance) Label() string { return "Create Instance" }
func (c *CreateInstance) Description() string { return "Provision a new OCI Compute instance" }

func (c *CreateInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{Name: "compartmentId", Label: "Compartment OCID", Type: configuration.FieldTypeString, Required: true},
		{Name: "displayName", Label: "Display Name", Type: configuration.FieldTypeString, Required: true},
		{Name: "shape", Label: "Shape", Type: configuration.FieldTypeString, Required: true},
		{Name: "imageId", Label: "Image OCID", Type: configuration.FieldTypeString, Required: true},
		{Name: "subnetId", Label: "Subnet OCID", Type: configuration.FieldTypeString, Required: true},
	}
}

func (c *CreateInstance) Execute(ctx core.ExecutionContext) error {
	var input struct {
		CompartmentID string `mapstructure:"compartmentId"`
		DisplayName   string `mapstructure:"displayName"`
		Shape         string `mapstructure:"shape"`
		ImageID       string `mapstructure:"imageId"`
		SubnetID      string `mapstructure:"subnetId"`
	}

	if err := mapstructure.Decode(ctx.Configuration, &input); err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	client, err := getClient(ctx)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	_, err = client.LaunchInstance(ctx.Context, ocicore.LaunchInstanceRequest{
		LaunchInstanceDetails: ocicore.LaunchInstanceDetails{
			CompartmentId: ocicommon.String(input.CompartmentID),
			DisplayName:   ocicommon.String(input.DisplayName),
			Shape:         ocicommon.String(input.Shape),
			SourceDetails: ocicore.InstanceSourceViaImageDetails{
				ImageId: ocicommon.String(input.ImageID),
			},
			CreateVnicDetails: &ocicore.CreateVnicDetails{
				SubnetId: ocicommon.String(input.SubnetID),
			},
		},
	})
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	return ctx.ExecutionState.Emit("success", "oci.instance", []any{map[string]interface{}{"status": "provisioning"}})
}

func (c *CreateInstance) Setup(ctx core.SetupContext) error { return nil }
func (c *CreateInstance) Cleanup(ctx core.SetupContext) error { return nil }
func (c *CreateInstance) Actions() []core.Action { return nil }
func (c *CreateInstance) HandleAction(ctx core.ActionContext) error { return nil }
func (c *CreateInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) { return 200, nil, nil }
func (c *CreateInstance) Cancel(ctx core.ExecutionContext) error { return nil }
