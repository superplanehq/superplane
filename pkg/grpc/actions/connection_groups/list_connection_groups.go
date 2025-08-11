package connectiongroups

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func ListConnectionGroups(ctx context.Context, canvasID string) (*pb.ListConnectionGroupsResponse, error) {
	connectionGroups, err := models.ListConnectionGroups(canvasID)
	if err != nil {
		return nil, fmt.Errorf("failed to list stages for canvas: %w", err)
	}

	serialized, err := serializeConnectionGroups(connectionGroups)
	if err != nil {
		return nil, err
	}

	response := &pb.ListConnectionGroupsResponse{
		ConnectionGroups: serialized,
	}

	return response, nil
}

func serializeConnectionGroups(in []models.ConnectionGroup) ([]*pb.ConnectionGroup, error) {
	out := make([]*pb.ConnectionGroup, len(in))
	for i, group := range in {
		connections, err := models.ListConnections(group.ID, models.ConnectionTargetTypeConnectionGroup)
		if err != nil {
			return nil, err
		}

		serialized, err := serializeConnectionGroup(group, connections)
		if err != nil {
			return nil, err
		}

		out[i] = serialized
	}

	return out, nil
}
