package connectiongroups

import (
	"context"
	"errors"
	"fmt"

	uuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DescribeConnectionGroup(ctx context.Context, canvasID string, idOrName string) (*pb.DescribeConnectionGroupResponse, error) {
	err := actions.ValidateUUIDs(idOrName)
	var connectionGroup *models.ConnectionGroup
	if err != nil {
		connectionGroup, err = models.FindConnectionGroupByName(canvasID, idOrName)
	} else {
		connectionGroup, err = models.FindConnectionGroupByID(canvasID, uuid.MustParse(idOrName))
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "connection group not found")
		}

		log.Errorf("Error describing connection group %s in canvas %s. Error: %v", idOrName, canvasID, err)
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
