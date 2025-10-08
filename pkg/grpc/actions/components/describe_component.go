package components

import (
	"context"

	pb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/registry"
)

func DescribeComponent(ctx context.Context, registry *registry.Registry, name string) (*pb.DescribeComponentResponse, error) {
	component, err := registry.GetComponent(name)
	if err != nil {
		return nil, err
	}

	outputs := component.OutputBranches(nil)
	branches := make([]*pb.OutputBranch, len(outputs))
	for i, output := range outputs {
		branches[i] = &pb.OutputBranch{
			Name: output,
		}
	}

	configFields := component.Configuration()
	configuration := make([]*pb.ConfigurationField, len(configFields))
	for i, field := range configFields {
		configuration[i] = &pb.ConfigurationField{
			Name:        field.Name,
			Type:        field.Type,
			Description: field.Description,
			Required:    field.Required,
		}
	}

	return &pb.DescribeComponentResponse{
		Component: &pb.Component{
			Name:          component.Name(),
			Branches:      branches,
			Configuration: configuration,
		},
	}, nil
}
