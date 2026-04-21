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

type CreateInstance struct{}

func (c *CreateInstance) Name() string {
	return "createInstance"
}

func (c *CreateInstance) Label() string {
	return "Create Instance"
}

func (c *CreateInstance) Description() string {
	return "Provision a new OCI Compute instance"
}

func (c *CreateInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "compartmentId",
			Label:       "Compartment OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the compartment to create the instance in",
		},
		{
			Name:        "displayName",
			Label:       "Display Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "A user-friendly name for the instance",
		},
		{
			Name:        "shape",
			Label:       "Shape",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The shape of the instance (e.g. VM.Standard.E4.Flex)",
		},
		{
			Name:        "imageId",
			Label:       "Image OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the image to use for the instance",
		},
		{
			Name:        "subnetId",
			Label:       "Subnet OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the subnet to create the instance in",
		},
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
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	client, err := getClient(ctx)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get OCI client: %v", err))
	}

	resp, err := client.LaunchInstance(ctx.Context, ocicore.LaunchInstanceRequest{
		LaunchInstanceDetails: ocicore.LaunchInstanceDetails{
			CompartmentId: common.String(input.CompartmentID),
			DisplayName:   common.String(input.DisplayName),
			Shape:         common.String(input.Shape),
			SourceDetails: ocicore.InstanceSourceViaImageDetails{
				ImageId: common.String(input.ImageID),
			},
			CreateVnicDetails: &ocicore.CreateVnicDetails{
				SubnetId: common.String(input.SubnetID),
			},
		},
	})
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to launch OCI instance: %v", err))
	}

	payload := map[string]interface{}{
		"id":           *resp.Instance.Id,
		"displayName":  *resp.Instance.DisplayName,
		"state":        string(resp.Instance.LifecycleState),
		"shape":        *resp.Instance.Shape,
		"region":       *resp.Instance.Region,
		"timeCreated":  resp.Instance.TimeCreated.String(),
	}

	return ctx.ExecutionState.Emit("success", "oci.instance", []any{payload})
}

func (c *CreateInstance) Setup(ctx core.SetupContext) error { return nil }
func (c *CreateInstance) Cleanup(ctx core.SetupContext) error { return nil }
func (c *CreateInstance) Actions() []core.Action { return nil }
func (c *CreateInstance) HandleAction(ctx core.ActionContext) error { return nil }
func (c *CreateInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (c *CreateInstance) Cancel(ctx core.ExecutionContext) error { return nil }
