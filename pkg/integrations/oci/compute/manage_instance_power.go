package compute

import (
	"context"
	"fmt"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ManageInstancePower struct{}

type ManageInstancePowerSpec struct {
	InstanceID string `mapstructure:"instanceId"`
	Action     string `mapstructure:"action"`
}

func (c *ManageInstancePower) Name() string {
	return "manageInstancePower"
}

func (c *ManageInstancePower) Label() string {
	return "Manage Instance Power"
}

func (c *ManageInstancePower) Description() string {
	return "Start, stop, or reset an OCI Compute instance"
}

func (c *ManageInstancePower) Setup() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instanceId",
			Label:       "Instance OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the instance to manage",
		},
		{
			Name:        "action",
			Label:       "Action",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "The power action to perform",
			Options: []configuration.FieldOption{
				{Label: "Start", Value: "START"},
				{Label: "Stop (Force)", Value: "STOP"},
				{Label: "Stop (Soft)", Value: "SOFTSTOP"},
				{Label: "Reset (Force)", Value: "RESET"},
				{Label: "Reset (Soft)", Value: "SOFTRESET"},
			},
		},
	}
}

func (c *ManageInstancePower) Run(ctx core.ExecutionContext) (any, error) {
	var spec ManageInstancePowerSpec
	if err := ctx.GetSpec(&spec); err != nil {
		return nil, err
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	req := ocicore.InstanceActionRequest{
		InstanceId: &spec.InstanceID,
		Action:     &spec.Action,
	}

	resp, err := client.InstanceAction(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform power action on OCI instance: %w", err)
	}

	return resp.Instance, nil
}
