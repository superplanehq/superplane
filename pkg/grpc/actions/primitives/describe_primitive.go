package primitives

import (
	"context"

	pb "github.com/superplanehq/superplane/pkg/protos/primitives"
	"github.com/superplanehq/superplane/pkg/registry"
)

func DescribePrimitive(ctx context.Context, registry *registry.Registry, name string) (*pb.DescribePrimitiveResponse, error) {
	primitive, err := registry.GetPrimitive(name)
	if err != nil {
		return nil, err
	}

	outputs := primitive.Outputs(nil)
	branches := make([]*pb.OutputBranch, len(outputs))
	for i, output := range outputs {
		branches[i] = &pb.OutputBranch{
			Name: output,
		}
	}

	configFields := primitive.Configuration()
	configuration := make([]*pb.ConfigurationField, len(configFields))
	for i, field := range configFields {
		configuration[i] = &pb.ConfigurationField{
			Name:        field.Name,
			Type:        field.Type,
			Description: field.Description,
			Required:    field.Required,
		}
	}

	return &pb.DescribePrimitiveResponse{
		Primitive: &pb.Primitive{
			Name:          primitive.Name(),
			Branches:      branches,
			Configuration: configuration,
		},
	}, nil
}
