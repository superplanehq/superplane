package canvases

import (
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// nonMapEventDataKey is the field name used to surface event payloads that are
// not encoded as a JSON object. Pre-existing or externally produced events can
// carry scalars, arrays, or null at the top level, and the gRPC `data` field is
// a `google.protobuf.Struct`, which only accepts an object. Wrapping the raw
// value preserves it for clients without forcing a 500.
const nonMapEventDataKey = "value"

func SerializeCanvasEvents(events []models.CanvasEvent) ([]*pb.CanvasEvent, error) {
	result := make([]*pb.CanvasEvent, 0, len(events))

	for _, event := range events {
		serializedEvent, err := SerializeCanvasEvent(event)
		if err != nil {
			return nil, err
		}
		result = append(result, serializedEvent)
	}

	return result, nil
}

func SerializeCanvasEvent(event models.CanvasEvent) (*pb.CanvasEvent, error) {
	data, err := canvasEventDataAsStruct(event.Data.Data())
	if err != nil {
		return nil, fmt.Errorf("failed to serialize event %s data: %w", event.ID, err)
	}

	return &pb.CanvasEvent{
		Id:         event.ID.String(),
		CanvasId:   event.WorkflowID.String(),
		NodeId:     event.NodeID,
		Channel:    event.Channel,
		CustomName: valueOrEmpty(event.CustomName),
		Data:       data,
		CreatedAt:  timestampOrNil(event.CreatedAt),
		Root:       event.ExecutionID == nil,
	}, nil
}

func SerializeNodeExecutionRef(execution models.CanvasNodeExecution) *pb.CanvasNodeExecutionRef {
	return &pb.CanvasNodeExecutionRef{
		Id:            execution.ID.String(),
		NodeId:        execution.NodeID,
		State:         NodeExecutionStateToProto(execution.State),
		Result:        NodeExecutionResultToProto(execution.Result),
		ResultReason:  NodeExecutionResultReasonToProto(execution.ResultReason),
		ResultMessage: execution.ResultMessage,
		CreatedAt:     timestampOrNil(execution.CreatedAt),
		UpdatedAt:     timestampOrNil(execution.UpdatedAt),
	}
}

// canvasEventDataAsStruct converts the persisted JSON value of a CanvasEvent
// into a structpb.Struct suitable for the gRPC `data` field. Events with nil or
// non-object top-level values are normalized so serialization never fails or
// panics — instead the raw value is exposed under `nonMapEventDataKey`.
func canvasEventDataAsStruct(data any) (*structpb.Struct, error) {
	if data == nil {
		return &structpb.Struct{}, nil
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		dataMap = map[string]any{nonMapEventDataKey: data}
	}

	return newStructpbStruct(dataMap)
}

func timestampOrNil(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func getLastEventTimestamp(events []models.CanvasEvent) *timestamppb.Timestamp {
	if len(events) == 0 {
		return nil
	}
	return timestampOrNil(events[len(events)-1].CreatedAt)
}
