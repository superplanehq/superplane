package connectiongroups

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DeleteConnectionGroup(ctx context.Context, canvasID, idOrName string) (*pb.DeleteConnectionGroupResponse, error) {
	err := actions.ValidateUUIDs(idOrName)
	var connectionGroup *models.ConnectionGroup
	if err != nil {
		connectionGroup, err = models.FindConnectionGroupByName(canvasID, idOrName)
	} else {
		connectionGroup, err = models.FindConnectionGroupByID(canvasID, idOrName)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "connection group not found")
		}

		log.Errorf("Error finding connection group %s in canvas %s: %v", idOrName, canvasID, err)
		return nil, status.Error(codes.Internal, "failed to find connection group")
	}

	err = connectionGroup.Delete()
	if err != nil {
		log.Errorf("Error deleting connection group %s in canvas %s: %v", idOrName, canvasID, err)
		return nil, status.Error(codes.Internal, "failed to delete connection group")
	}

	return &pb.DeleteConnectionGroupResponse{}, nil
}
