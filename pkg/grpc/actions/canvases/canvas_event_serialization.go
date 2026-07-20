package canvases

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
	data, ok := event.Data.Data().(map[string]any)
	if !ok {
		return nil, fmt.Errorf("event data is not a map[string]any")
	}

	s, err := newStructpbStruct(data)
	if err != nil {
		return nil, err
	}

	return &pb.CanvasEvent{
		Id:         event.ID.String(),
		CanvasId:   event.WorkflowID.String(),
		NodeId:     event.NodeID,
		Channel:    event.Channel,
		CustomName: valueOrEmpty(event.CustomName),
		Data:       s,
		CreatedAt:  timestamppb.New(*event.CreatedAt),
		Root:       event.ExecutionID == nil,
		RunId:      uuidStringOrEmpty(event.RunID),
	}, nil
}

func SerializeNodeExecutionRef(execution models.CanvasNodeExecution, childRuns []models.CanvasRun) *pb.CanvasNodeExecutionRef {
	return &pb.CanvasNodeExecutionRef{
		Id:            execution.ID.String(),
		NodeId:        execution.NodeID,
		State:         NodeExecutionStateToProto(execution.State),
		Result:        NodeExecutionResultToProto(execution.Result),
		ResultReason:  NodeExecutionResultReasonToProto(execution.ResultReason),
		ResultMessage: execution.ResultMessage,
		CreatedAt:     timestamppb.New(*execution.CreatedAt),
		UpdatedAt:     timestamppb.New(*execution.UpdatedAt),
		Runs:          SerializeCanvasRunRefs(childRuns),
	}
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func uuidStringOrEmpty(value uuid.UUID) string {
	if value == uuid.Nil {
		return ""
	}

	return value.String()
}

func getLastEventTimestamp(events []models.CanvasEvent) *timestamppb.Timestamp {
	if len(events) > 0 {
		return timestamppb.New(*events[len(events)-1].CreatedAt)
	}
	return nil
}
