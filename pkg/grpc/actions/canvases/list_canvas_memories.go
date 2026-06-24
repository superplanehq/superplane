package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func ListCanvasMemories(ctx context.Context, registry *registry.Registry, organizationID, canvasID string) (*pb.ListCanvasMemoriesResponse, error) {
	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid canvas_id")
	}

	err = checkCanvasExistence(ctx, database.DB(ctx), orgUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "canvas not found")
		}
		return nil, grpcerrors.Internal(err, "failed to load canvas")
	}

	var records []models.CanvasMemory
	err = telemetry.RunSpan(ctx, "memories.list", func(ctx context.Context) error {
		var listErr error
		records, listErr = models.ListCanvasMemoriesInTransaction(database.DB(ctx), canvasUUID)
		return listErr
	})
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list canvas memories")
	}

	items, err := serializeCanvasMemories(ctx, records)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to serialize canvas memory")
	}

	return &pb.ListCanvasMemoriesResponse{
		Items: items,
	}, nil
}

func serializeCanvasMemories(ctx context.Context, records []models.CanvasMemory) ([]*pb.CanvasMemory, error) {
	var items []*pb.CanvasMemory
	err := telemetry.RunSpan(ctx, "memories.serialize", func(ctx context.Context) error {
		items = make([]*pb.CanvasMemory, 0, len(records))
		for _, record := range records {
			item, itemErr := canvasMemoryToProto(record)
			if itemErr != nil {
				return itemErr
			}

			items = append(items, item)
		}

		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.SetAttributes(attribute.Int("memories.count", len(records)))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return items, nil
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
		CreatedAt: timestamppb.New(record.CreatedAt),
		UpdatedAt: timestamppb.New(record.UpdatedAt),
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
