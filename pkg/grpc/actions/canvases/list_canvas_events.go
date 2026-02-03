package canvases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListCanvasEvents(ctx context.Context, registry *registry.Registry, canvasID uuid.UUID, limit uint32, before *timestamppb.Timestamp) (*pb.ListCanvasEventsResponse, error) {
	limit = getLimit(limit)
	beforeTime := getBefore(before)
	events, err := models.ListRootCanvasEvents(canvasID, int(limit), beforeTime)
	if err != nil {
		return nil, err
	}

	count, err := models.CountRootCanvasEvents(canvasID)
	if err != nil {
		return nil, err
	}

	executionsByEventID, childExecutionsByEventID, err := listExecutionsForCanvasEvents(events)
	if err != nil {
		return nil, err
	}

	serialized, err := SerializeCanvasEventsWithExecutions(events, executionsByEventID, childExecutionsByEventID)
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

func SerializeCanvasEventsWithExecutions(events []models.CanvasEvent, executionsByEventID map[string][]models.CanvasNodeExecution, childExecutionsByEventID map[string][]models.CanvasNodeExecution) ([]*pb.CanvasEventWithExecutions, error) {
	result := make([]*pb.CanvasEventWithExecutions, 0, len(events))

	for _, event := range events {
		serializedEvent, err := SerializeCanvasEventWithExecutions(event, executionsByEventID[event.ID.String()], childExecutionsByEventID[event.ID.String()])
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

	s, err := structpb.NewStruct(data)
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
	}, nil
}

func SerializeCanvasEventWithExecutions(event models.CanvasEvent, executions []models.CanvasNodeExecution, childExecutions []models.CanvasNodeExecution) (*pb.CanvasEventWithExecutions, error) {
	data, ok := event.Data.Data().(map[string]any)
	if !ok {
		return nil, fmt.Errorf("event data is not a map[string]any")
	}

	s, err := structpb.NewStruct(data)
	if err != nil {
		return nil, err
	}

	serializedExecutions, err := SerializeNodeExecutions(executions, childExecutions)
	if err != nil {
		return nil, err
	}

	return &pb.CanvasEventWithExecutions{
		Id:         event.ID.String(),
		CanvasId:   event.WorkflowID.String(),
		NodeId:     event.NodeID,
		Channel:    event.Channel,
		CustomName: valueOrEmpty(event.CustomName),
		Data:       s,
		CreatedAt:  timestamppb.New(*event.CreatedAt),
		Executions: serializedExecutions,
	}, nil
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

func listExecutionsForCanvasEvents(events []models.CanvasEvent) (map[string][]models.CanvasNodeExecution, map[string][]models.CanvasNodeExecution, error) {
	if len(events) == 0 {
		return map[string][]models.CanvasNodeExecution{}, map[string][]models.CanvasNodeExecution{}, nil
	}

	eventIDs := make([]uuid.UUID, len(events))
	for i, event := range events {
		eventIDs[i] = event.ID
	}

	executions, err := models.ListNodeExecutionsForRootEvents(eventIDs)
	if err != nil {
		return nil, nil, err
	}

	parentExecutionsByEventID := make(map[string][]models.CanvasNodeExecution, len(eventIDs))
	parentExecutions := []models.CanvasNodeExecution{}
	for _, execution := range executions {
		if execution.ParentExecutionID == nil {
			parentExecutions = append(parentExecutions, execution)
			eventID := execution.RootEventID.String()
			parentExecutionsByEventID[eventID] = append(parentExecutionsByEventID[eventID], execution)
		}
	}

	if len(parentExecutions) == 0 {
		return parentExecutionsByEventID, map[string][]models.CanvasNodeExecution{}, nil
	}

	childExecutions, err := models.FindChildExecutionsForMultiple(executionIDs(parentExecutions))
	if err != nil {
		return nil, nil, err
	}

	childExecutionsByEventID := make(map[string][]models.CanvasNodeExecution, len(eventIDs))
	for _, execution := range childExecutions {
		eventID := execution.RootEventID.String()
		childExecutionsByEventID[eventID] = append(childExecutionsByEventID[eventID], execution)
	}

	return parentExecutionsByEventID, childExecutionsByEventID, nil
}
