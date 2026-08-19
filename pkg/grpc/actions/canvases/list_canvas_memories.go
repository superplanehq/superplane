package canvases

import (
	"context"

	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func ListCanvasMemories(ctx context.Context, db *gorm.DB, canvas *models.Canvas) (*pb.ListCanvasMemoriesResponse, error) {
	records, err := listCanvasMemories(ctx, db, canvas)
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

func listCanvasMemories(ctx context.Context, db *gorm.DB, canvas *models.Canvas) (records []models.CanvasMemory, err error) {
	ctx, done := telemetry.Span(ctx, "memories.list")
	defer done(&err)

	return models.ListCanvasMemoriesInTransaction(db, canvas.ID)
}

func serializeCanvasMemories(ctx context.Context, records []models.CanvasMemory) (items []*pb.CanvasMemory, err error) {
	ctx, done := telemetry.Span(ctx, "memories.serialize")
	defer done(&err)

	items = make([]*pb.CanvasMemory, 0, len(records))
	for _, record := range records {
		item, itemErr := canvasMemoryToProto(record)
		if itemErr != nil {
			return nil, itemErr
		}

		items = append(items, item)
	}

	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.SetAttributes(attribute.Int("memories.count", len(records)))
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
