package compute

import (
	"context"
	"fmt"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetInstance struct{}

type GetInstanceSpec struct {
	InstanceID string `mapstructure:"instanceId"`
}

func (c *GetInstance) Name() string {
	return "getInstance"
}

func (c *GetInstance) Label() string {
	return "Get Instance"
}

func (c *GetInstance) Description() string {
	return "Retrieve details of an OCI Compute instance"
}

func (c *GetInstance) Setup() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instanceId",
			Label:       "Instance OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the instance to retrieve",
		},
	}
}

func (c *GetInstance) Run(ctx core.ExecutionContext) (any, error) {
	var spec GetInstanceSpec
	if err := ctx.GetSpec(&spec); err != nil {
		return nil, err
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	req := ocicore.GetInstanceRequest{
		InstanceId: &spec.InstanceID,
	}

	resp, err := client.GetInstance(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to get OCI instance: %w", err)
	}

	return resp.Instance, nil
}
