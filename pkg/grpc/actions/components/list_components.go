package components

import (
	"context"

	"github.com/superplanehq/superplane/pkg/components"
	pb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListComponents(ctx context.Context, registry *registry.Registry) (*pb.ListComponentsResponse, error) {
	return &pb.ListComponentsResponse{
		Components: serializeComponents(registry.ListComponents()),
	}, nil
}

func serializeComponents(in []components.Component) []*pb.Component {
	out := make([]*pb.Component, len(in))
	for i, component := range in {
		outputs := component.Outputs(nil)
		branches := make([]*pb.OutputBranch, len(outputs))
		for j, output := range outputs {
			branches[j] = &pb.OutputBranch{
				Name: output,
			}
		}

		configFields := component.Configuration()
		configuration := make([]*pb.ConfigurationField, len(configFields))
		for j, field := range configFields {
			configuration[j] = &pb.ConfigurationField{
				Name:        field.Name,
				Type:        field.Type,
				Description: field.Description,
				Required:    field.Required,
			}
		}

		out[i] = &pb.Component{
			Name:          component.Name(),
			Description:   component.Description(),
			Branches:      branches,
			Configuration: configuration,
		}
	}
	return out
}
