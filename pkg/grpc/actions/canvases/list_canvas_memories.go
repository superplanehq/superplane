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

func ListCanvasMemories(ctx context.Context, registry *registry.Registry, organizationID, canvasID string) (*pb.ListCanvasMemoriesResponse, error) {
	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas_id")
	}

	_, err = models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas not found")
		}
		return nil, status.Error(codes.Internal, "failed to load canvas")
	}

	records, err := models.ListCanvasMemories(canvasUUID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list canvas memories")
	}

	items := make([]*pb.CanvasMemory, 0, len(records))
	for _, record := range records {
		item, err := canvasMemoryToProto(record)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to serialize canvas memory")
		}

		items = append(items, item)
	}

	return &pb.ListCanvasMemoriesResponse{
		Items: items,
	}, nil
}

func canvasMemoryToProto(record models.CanvasMemory) (*pb.CanvasMemory, error) {
	values, err := newStructpbValue(record.Values.Data())
	if err != nil {
		return nil, err
	}

	return &pb.CanvasMemory{
		Id:        record.ID.String(),
		Namespace: record.Namespace,
		Values:    values,
		Source:    canvasMemorySourceToProto(record.Source),
	}, nil
}

func canvasMemorySourceToProto(source string) pb.CanvasMemory_Source {
	switch source {
	case models.CanvasMemorySourceNode:
		return pb.CanvasMemory_SOURCE_NODE
	case models.CanvasMemorySourceManual:
		return pb.CanvasMemory_SOURCE_MANUAL
	default:
		return pb.CanvasMemory_SOURCE_UNKNOWN
	}
}
