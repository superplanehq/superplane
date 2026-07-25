package canvases

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

func DeleteCanvasMemory(ctx context.Context, db *gorm.DB, canvas *models.Canvas, memoryID string) (*pb.DeleteCanvasMemoryResponse, error) {
	entryUUID, err := uuid.Parse(memoryID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid memory_id")
	}

	if err := models.DeleteCanvasMemory(canvas.ID, entryUUID); err != nil {
		return nil, grpcerrors.Internal(err, "failed to delete canvas memory")
	}

	if err := messages.NewCanvasMemoryUpdatedMessage(canvas.ID.String()).PublishMemoryUpdated(); err != nil {
		log.Errorf("failed to publish canvas memory updated RabbitMQ message: %v", err)
	}

	return &pb.DeleteCanvasMemoryResponse{}, nil
}
