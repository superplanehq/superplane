package components

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/components"
	configpb "github.com/superplanehq/superplane/pkg/protos/configuration"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListComponentActions(ctx context.Context, registry *registry.Registry, name string) (*pb.ListComponentActionsResponse, error) {
	component, err := registry.GetComponent(name)
	if err != nil {
		return nil, fmt.Errorf("component not found: %w", err)
	}

	componentActions := component.Actions()
	pbActions := []*pb.ComponentAction{}

	for i, action := range componentActions {
		if !action.UserAccessible {
			continue
		}

		parameters := make([]*configpb.Field, len(action.Parameters))
		for j, param := range action.Parameters {
			parameters[j] = actions.ConfigurationFieldToProto(param)
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
