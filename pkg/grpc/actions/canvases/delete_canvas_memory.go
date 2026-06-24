package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

func DeleteCanvasMemory(ctx context.Context, registry *registry.Registry, organizationID, canvasID, memoryID string) (*pb.DeleteCanvasMemoryResponse, error) {
	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid canvas_id")
	}

	entryUUID, err := uuid.Parse(memoryID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid memory_id")
	}

	_, err = models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "canvas not found")
		}
		return nil, grpcerrors.Internal(err, "failed to load canvas")
	}

	if err := models.DeleteCanvasMemory(canvasUUID, entryUUID); err != nil {
		return nil, grpcerrors.Internal(err, "failed to delete canvas memory")
	}

	if err := messages.NewCanvasMemoryUpdatedMessage(canvasUUID.String()).PublishMemoryUpdated(); err != nil {
		log.Errorf("failed to publish canvas memory updated RabbitMQ message: %v", err)
	}

	return &pb.DeleteCanvasMemoryResponse{}, nil
}
