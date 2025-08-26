package eventsources

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DeleteEventSource(ctx context.Context, canvasID string, idOrName string) (*pb.DeleteEventSourceResponse, error) {
	source, err := findEventSource(canvasID, idOrName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "event source not found")
		}

		log.Errorf("Error finding event source %s in canvas %s: %v", idOrName, canvasID, err)
		return nil, err
	}

	err = source.Delete()
	if err != nil {
		log.Errorf("Error deleting event source %s in canvas %s: %v", idOrName, canvasID, err)
		return nil, status.Error(codes.Internal, "failed to delete event source")
	}

	return &pb.DeleteEventSourceResponse{}, nil
}
