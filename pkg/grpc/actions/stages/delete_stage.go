package stages

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DeleteStage(ctx context.Context, canvasID string, idOrName string) (*pb.DeleteStageResponse, error) {
	stage, err := findStage(canvasID, idOrName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "stage not found")
		}

		log.Errorf("Error finding stage %s in canvas %s: %v", idOrName, canvasID, err)
		return nil, err
	}

	err = stage.Delete()
	if err != nil {
		log.Errorf("Error deleting stage %s in canvas %s: %v", idOrName, canvasID, err)
		return nil, status.Error(codes.Internal, "failed to delete stage")
	}

	return &pb.DeleteStageResponse{}, nil
}