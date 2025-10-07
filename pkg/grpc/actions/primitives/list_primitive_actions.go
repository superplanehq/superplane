package primitives

import (
	"context"
	"fmt"

	pb "github.com/superplanehq/superplane/pkg/protos/primitives"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListPrimitiveActions(ctx context.Context, registry *registry.Registry, name string) (*pb.ListPrimitiveActionsResponse, error) {
	primitive, err := registry.GetPrimitive(name)
	if err != nil {
		return nil, fmt.Errorf("primitive not found: %w", err)
	}

	actions := primitive.Actions()
	pbActions := make([]*pb.PrimitiveAction, len(actions))
	for i, action := range actions {
		// Convert ConfigurationFields to protobuf format
		parameters := make([]*pb.ConfigurationField, len(action.Parameters))
		for j, param := range action.Parameters {
			parameters[j] = &pb.ConfigurationField{
				Name:        param.Name,
				Type:        param.Type,
				Description: param.Description,
				Required:    param.Required,
			}
		}

		pbActions[i] = &pb.PrimitiveAction{
			Name:        action.Name,
			Description: action.Description,
			Parameters:  parameters,
		}
	}

	return &pb.ListPrimitiveActionsResponse{
		Actions: pbActions,
	}, nil
}
