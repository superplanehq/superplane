package connectiongroups

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListConnectionGroups(ctx context.Context, req *pb.ListConnectionGroupsRequest) (*pb.ListConnectionGroupsResponse, error) {
	err := actions.ValidateUUIDs(req.CanvasIdOrName)
	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "canvas not found")
	}

	connectionGroups, err := canvas.ListConnectionGroups()
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
