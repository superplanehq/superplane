package components

import (
	"context"
	"fmt"

	pb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListComponentActions(ctx context.Context, registry *registry.Registry, name string) (*pb.ListComponentActionsResponse, error) {
	component, err := registry.GetComponent(name)
	if err != nil {
		return nil, fmt.Errorf("component not found: %w", err)
	}

	actions := component.Actions()
	pbActions := make([]*pb.ComponentAction, len(actions))
	for i, action := range actions {
		// Convert ConfigurationFields to protobuf format
		parameters := make([]*pb.ConfigurationField, len(action.Parameters))
		for j, param := range action.Parameters {
			parameters[j] = ConfigurationFieldToProto(param)
		}

		pbActions[i] = &pb.ComponentAction{
			Name:        action.Name,
			Description: action.Description,
			Parameters:  parameters,
		}
	}

	return &pb.ListComponentActionsResponse{
		Actions: pbActions,
	}, nil
}
