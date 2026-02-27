package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DeleteCanvasMemory(ctx context.Context, registry *registry.Registry, organizationID, canvasID, memoryID string) (*pb.DeleteCanvasMemoryResponse, error) {
	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas_id")
	}

	entryUUID, err := uuid.Parse(memoryID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid memory_id")
	}

	_, err = models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas not found")
		}
		return nil, status.Error(codes.Internal, "failed to load canvas")
	}

	if err := models.DeleteCanvasMemory(canvasUUID, entryUUID); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete canvas memory")
	}

	return &pb.DeleteCanvasMemoryResponse{}, nil
}
