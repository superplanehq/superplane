package compute

import (
	"context"
	"fmt"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateInstance struct{}

type CreateInstanceSpec struct {
	CompartmentID      string `mapstructure:"compartmentId"`
	AvailabilityDomain string `mapstructure:"availabilityDomain"`
	DisplayName        string `mapstructure:"displayName"`
	Shape              string `mapstructure:"shape"`
	ImageID            string `mapstructure:"imageId"`
	SubnetID           string `mapstructure:"subnetId"`
}

func (c *CreateInstance) Name() string {
	return "createInstance"
}

func (c *CreateInstance) Label() string {
	return "Create Instance"
}

func (c *CreateInstance) Description() string {
	return "Provision a new Compute instance in OCI"
}

func (c *CreateInstance) Setup() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "compartmentId",
			Label:       "Compartment OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the compartment to create the instance in",
		},
		{
			Name:        "availabilityDomain",
			Label:       "Availability Domain",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The availability domain to create the instance in (e.g. Uocm:US-ASHBURN-AD-1)",
		},
		{
			Name:        "displayName",
			Label:       "Display Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
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

func (c *CreateInstance) Run(ctx core.ExecutionContext) (any, error) {
	var spec CreateInstanceSpec
	if err := ctx.GetSpec(&spec); err != nil {
		return nil, err
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	req := ocicore.LaunchInstanceRequest{
		LaunchInstanceDetails: ocicore.LaunchInstanceDetails{
			CompartmentId:      &spec.CompartmentID,
			AvailabilityDomain: &spec.AvailabilityDomain,
			DisplayName:        &spec.DisplayName,
			Shape:              &spec.Shape,
			SourceDetails: ocicore.InstanceSourceViaImageDetails{
				ImageId: &spec.ImageID,
			},
			CreateVnicDetails: &ocicore.CreateVnicDetails{
				SubnetId: &spec.SubnetID,
			},
		},
	}

	resp, err := client.CreateInstance(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to launch OCI instance: %w", err)
	}

	return resp.Instance, nil
}
