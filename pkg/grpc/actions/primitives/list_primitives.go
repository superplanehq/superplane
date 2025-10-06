package primitives

import (
	"context"

	"github.com/superplanehq/superplane/pkg/primitives"
	pb "github.com/superplanehq/superplane/pkg/protos/primitives"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListPrimitives(ctx context.Context, registry *registry.Registry) (*pb.ListPrimitivesResponse, error) {
	return &pb.ListPrimitivesResponse{
		Primitives: serializePrimitives(registry.ListPrimitives()),
	}, nil
}

func serializePrimitives(in []primitives.Primitive) []*pb.Primitive {
	out := make([]*pb.Primitive, len(in))
	for i, primitive := range in {
		outputs := primitive.Outputs(nil)
		branches := make([]*pb.OutputBranch, len(outputs))
		for j, output := range outputs {
			branches[j] = &pb.OutputBranch{
				Name: output,
			}
		}

		configFields := primitive.Configuration()
		configuration := make([]*pb.ConfigurationField, len(configFields))
		for j, field := range configFields {
			configuration[j] = &pb.ConfigurationField{
				Name:        field.Name,
				Type:        field.Type,
				Description: field.Description,
				Required:    field.Required,
			}
		}

		out[i] = &pb.Primitive{
			Name:          primitive.Name(),
			Description:   primitive.Description(),
			Branches:      branches,
			Configuration: configuration,
		}
	}
	return out
}
