package compute

import (
	"context"
	"fmt"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteInstance struct{}

type DeleteInstanceSpec struct {
	InstanceID string `mapstructure:"instanceId"`
}

func (c *DeleteInstance) Name() string {
	return "deleteInstance"
}

func (c *DeleteInstance) Label() string {
	return "Delete Instance"
}

func (c *DeleteInstance) Description() string {
	return "Terminate an OCI Compute instance"
}

func (c *DeleteInstance) Setup() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instanceId",
			Label:       "Instance OCID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The OCID of the instance to terminate",
		},
	}
}

func (c *DeleteInstance) Run(ctx core.ExecutionContext) (any, error) {
	var spec DeleteInstanceSpec
	if err := ctx.GetSpec(&spec); err != nil {
		return nil, err
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	req := ocicore.TerminateInstanceRequest{
		InstanceId: &spec.InstanceID,
	}

	_, err = client.TerminateInstance(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to terminate OCI instance: %w", err)
	}

	return map[string]string{"status": "terminating"}, nil
}
