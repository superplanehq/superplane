package connectiongroups

import (
	"context"
	"errors"
	"fmt"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DescribeConnectionGroup(ctx context.Context, req *pb.DescribeConnectionGroupRequest) (*pb.DescribeConnectionGroupResponse, error) {
	// Find canvas
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

	// Find connection group
	logger := logging.ForCanvas(canvas)
	err = actions.ValidateUUIDs(req.IdOrName)
	var connectionGroup *models.ConnectionGroup
	if err != nil {
		connectionGroup, err = canvas.FindConnectionGroupByName(req.IdOrName)
	} else {
		connectionGroup, err = canvas.FindConnectionGroupByID(uuid.MustParse(req.IdOrName))
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "connection group not found")
		}

		logger.Errorf("Error describing connection group. Request: %v. Error: %v", req, err)
		return nil, err
	}

	//
	// Connection group exists, serialize it
	//
	connections, err := models.ListConnections(connectionGroup.ID, models.ConnectionTargetTypeConnectionGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to list connections for connection group: %w", err)
	}

	serialized, err := serializeConnectionGroup(*connectionGroup, connections)
	if err != nil {
		return nil, err
	}

	response := &pb.DescribeConnectionGroupResponse{
		ConnectionGroup: serialized,
	}

	return response, nil
}
