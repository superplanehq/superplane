package connectiongroups

import (
	"context"

	uuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateConnectionGroup(ctx context.Context, req *pb.UpdateConnectionGroupRequest) (*pb.UpdateConnectionGroupResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	err := actions.ValidateUUIDs(req.CanvasIdOrName)
	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	err = actions.ValidateUUIDs(req.IdOrName)
	var connectionGroup *models.ConnectionGroup
	if err != nil {
		connectionGroup, err = canvas.FindConnectionGroupByName(req.IdOrName)
	} else {
		connectionGroup, err = canvas.FindConnectionGroupByID(uuid.MustParse(req.IdOrName))
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "connection group not found")
	}

	connections, err := actions.ValidateConnections(canvas, req.ConnectionGroup.Spec.Connections)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	spec, err := validateSpec(req.ConnectionGroup.Spec)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = canvas.UpdateConnectionGroup(connectionGroup.ID.String(), userID, connections, *spec)
	if err != nil {
		log.Errorf("Error updating connection group. Request: %v. Error: %v", req, err)
		return nil, err
	}

	connectionGroup, _ = canvas.FindConnectionGroupByID(connectionGroup.ID)
	group, err := serializeConnectionGroup(*connectionGroup, connections)
	if err != nil {
		return nil, err
	}

	response := &pb.UpdateConnectionGroupResponse{
		ConnectionGroup: group,
	}

	return response, nil
}
