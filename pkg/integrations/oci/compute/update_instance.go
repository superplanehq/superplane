package compute

import (
	"context"
	"fmt"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateInstance struct{}

type UpdateInstanceSpec struct {
	InstanceID  string `mapstructure:"instanceId"`
	DisplayName string `mapstructure:"displayName"`
}

func (c *UpdateInstance) Name() string {
	return "updateInstance"
}

func (c *UpdateInstance) Label() string {
	return "Update Instance"
}

func (c *UpdateInstance) Description() string {
	return "Update an OCI Compute instance"
}

func (c *UpdateInstance) Setup() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instanceId",
			Label:       "Instance OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the instance to update",
		},
		{
			Name:        "displayName",
			Label:       "Display Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The new display name for the instance",
		},
	}
}

func (c *UpdateInstance) Run(ctx core.ExecutionContext) (any, error) {
	var spec UpdateInstanceSpec
	if err := ctx.GetSpec(&spec); err != nil {
		return nil, err
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	req := ocicore.UpdateInstanceRequest{
		InstanceId: &spec.InstanceID,
		UpdateInstanceDetails: ocicore.UpdateInstanceDetails{
			DisplayName: &spec.DisplayName,
		},
	}

	resp, err := client.UpdateInstance(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to update OCI instance: %w", err)
	}

	return resp.Instance, nil
}
