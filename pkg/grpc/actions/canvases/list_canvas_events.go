package canvases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListCanvasEvents(ctx context.Context, registry *registry.Registry, canvasID uuid.UUID, limit uint32, before *timestamppb.Timestamp) (*pb.ListCanvasEventsResponse, error) {
	limit = getLimit(limit)
	beforeTime := getBefore(before)

	var events []models.CanvasEvent
	err := telemetry.RunSpan(ctx, "events.list", func(ctx context.Context) error {
		var listErr error
		events, listErr = models.ListRootCanvasEventsInTransaction(database.DB(ctx), canvasID, int(limit), beforeTime)
		return listErr
	})
	if err != nil {
		return nil, err
	}

	var count int64
	err = telemetry.RunSpan(ctx, "events.count", func(ctx context.Context) error {
		var countErr error
		count, countErr = models.CountRootCanvasEventsInTransaction(database.DB(ctx), canvasID)
		return countErr
	})
	if err != nil {
		return nil, err
	}

	var executionsByEventID map[string][]models.CanvasNodeExecution
	err = telemetry.RunSpan(ctx, "events.load_executions", func(ctx context.Context) error {
		var loadErr error
		executionsByEventID, loadErr = listExecutionsForCanvasEvents(ctx, canvasID, events)
		return loadErr
	})
	if err != nil {
		return nil, err
	}

	serialized, err := SerializeCanvasEventsWithExecutions(events, executionsByEventID)
	if err != nil {
		return nil, err
	}

	return &pb.ListCanvasEventsResponse{
		Events:        serialized,
		TotalCount:    uint32(count),
		HasNextPage:   hasNextPage(len(events), int(limit), count),
		LastTimestamp: getLastEventTimestamp(events),
	}, nil
}

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

func SerializeCanvasEventsWithExecutions(events []models.CanvasEvent, executionsByEventID map[string][]models.CanvasNodeExecution) ([]*pb.CanvasEventWithExecutions, error) {
	result := make([]*pb.CanvasEventWithExecutions, 0, len(events))

	for _, event := range events {
		serializedEvent, err := SerializeCanvasEventWithExecutions(event, executionsByEventID[event.ID.String()])
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
	}, nil
}

func SerializeCanvasEventWithExecutions(event models.CanvasEvent, executions []models.CanvasNodeExecution) (*pb.CanvasEventWithExecutions, error) {
	data, ok := event.Data.Data().(map[string]any)
	if !ok {
		return nil, fmt.Errorf("event data is not a map[string]any")
	}

	s, err := newStructpbStruct(data)
	if err != nil {
		return nil, err
	}

	executionInfos := make([]*pb.CanvasNodeExecutionRef, 0, len(executions))
	for _, execution := range executions {
		executionInfos = append(executionInfos, SerializeNodeExecutionRef(execution))
	}

	return &pb.CanvasEventWithExecutions{
		Id:         event.ID.String(),
		CanvasId:   event.WorkflowID.String(),
		NodeId:     event.NodeID,
		Channel:    event.Channel,
		CustomName: valueOrEmpty(event.CustomName),
		Data:       s,
		CreatedAt:  timestamppb.New(*event.CreatedAt),
		Executions: executionInfos,
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
		CreatedAt:     timestamppb.New(*execution.CreatedAt),
		UpdatedAt:     timestamppb.New(*execution.UpdatedAt),
	}
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func getLastEventTimestamp(events []models.CanvasEvent) *timestamppb.Timestamp {
	if len(events) > 0 {
		return timestamppb.New(*events[len(events)-1].CreatedAt)
	}
	return nil
}

func listExecutionsForCanvasEvents(ctx context.Context, canvasID uuid.UUID, events []models.CanvasEvent) (map[string][]models.CanvasNodeExecution, error) {
	if len(events) == 0 {
		return map[string][]models.CanvasNodeExecution{}, nil
	}

	eventIDs := make([]uuid.UUID, len(events))
	for i, event := range events {
		eventIDs[i] = event.ID
	}

	executions, err := models.ListParentExecutionsForRootEventsInTransaction(database.DB(ctx), canvasID, eventIDs)
	if err != nil {
		return nil, err
	}

	executionsByEventID := make(map[string][]models.CanvasNodeExecution, len(eventIDs))
	for _, execution := range executions {
		eventID := execution.RootEventID.String()
		executionsByEventID[eventID] = append(executionsByEventID[eventID], execution)
	}

	return executionsByEventID, nil
}
