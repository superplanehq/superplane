package workflows

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListWorkflowEvents(ctx context.Context, registry *registry.Registry, workflowID uuid.UUID, limit uint32, before *timestamppb.Timestamp) (*pb.ListWorkflowEventsResponse, error) {
	limit = getLimit(limit)
	beforeTime := getBefore(before)
	events, err := models.ListRootWorkflowEvents(workflowID, int(limit), beforeTime)
	if err != nil {
		return nil, err
	}

	count, err := models.CountRootWorkflowEvents(workflowID)
	if err != nil {
		return nil, err
	}

	executionsByEventID, childExecutionsByEventID, err := listExecutionsForWorkflowEvents(events)
	if err != nil {
		return nil, err
	}

	serialized, err := SerializeWorkflowEventsWithExecutions(events, executionsByEventID, childExecutionsByEventID)
	if err != nil {
		return nil, err
	}

	return &pb.ListWorkflowEventsResponse{
		Events:        serialized,
		TotalCount:    uint32(count),
		HasNextPage:   hasNextPage(len(events), int(limit), count),
		LastTimestamp: getLastEventTimestamp(events),
	}, nil
}

func SerializeWorkflowEvents(events []models.WorkflowEvent) ([]*pb.WorkflowEvent, error) {
	result := make([]*pb.WorkflowEvent, 0, len(events))

	for _, event := range events {
		serializedEvent, err := SerializeWorkflowEvent(event)
		if err != nil {
			return nil, err
		}
		result = append(result, serializedEvent)
	}

	return result, nil
}

func SerializeWorkflowEventsWithExecutions(events []models.WorkflowEvent, executionsByEventID map[string][]models.WorkflowNodeExecution, childExecutionsByEventID map[string][]models.WorkflowNodeExecution) ([]*pb.WorkflowEventWithExecutions, error) {
	result := make([]*pb.WorkflowEventWithExecutions, 0, len(events))

	for _, event := range events {
		serializedEvent, err := SerializeWorkflowEventWithExecutions(event, executionsByEventID[event.ID.String()], childExecutionsByEventID[event.ID.String()])
		if err != nil {
			return nil, err
		}
		result = append(result, serializedEvent)
	}

	return result, nil
}

func SerializeWorkflowEvent(event models.WorkflowEvent) (*pb.WorkflowEvent, error) {
	data, ok := event.Data.Data().(map[string]any)
	if !ok {
		return nil, fmt.Errorf("event data is not a map[string]any")
	}

	s, err := structpb.NewStruct(data)
	if err != nil {
		return nil, err
	}

	return &pb.WorkflowEvent{
		Id:         event.ID.String(),
		WorkflowId: event.WorkflowID.String(),
		NodeId:     event.NodeID,
		Channel:    event.Channel,
		CustomName: valueOrEmpty(event.CustomName),
		Data:       s,
		CreatedAt:  timestamppb.New(*event.CreatedAt),
	}, nil
}

func SerializeWorkflowEventWithExecutions(event models.WorkflowEvent, executions []models.WorkflowNodeExecution, childExecutions []models.WorkflowNodeExecution) (*pb.WorkflowEventWithExecutions, error) {
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

	return &pb.WorkflowEventWithExecutions{
		Id:         event.ID.String(),
		WorkflowId: event.WorkflowID.String(),
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

func getLastEventTimestamp(events []models.WorkflowEvent) *timestamppb.Timestamp {
	if len(events) > 0 {
		return timestamppb.New(*events[len(events)-1].CreatedAt)
	}
	return nil
}

func listExecutionsForWorkflowEvents(events []models.WorkflowEvent) (map[string][]models.WorkflowNodeExecution, map[string][]models.WorkflowNodeExecution, error) {
	if len(events) == 0 {
		return map[string][]models.WorkflowNodeExecution{}, map[string][]models.WorkflowNodeExecution{}, nil
	}

	eventIDs := make([]uuid.UUID, len(events))
	for i, event := range events {
		eventIDs[i] = event.ID
	}

	executions, err := models.ListNodeExecutionsForRootEvents(eventIDs)
	if err != nil {
		return nil, nil, err
	}

	parentExecutionsByEventID := make(map[string][]models.WorkflowNodeExecution, len(eventIDs))
	parentExecutions := []models.WorkflowNodeExecution{}
	for _, execution := range executions {
		if execution.ParentExecutionID == nil {
			parentExecutions = append(parentExecutions, execution)
			eventID := execution.RootEventID.String()
			parentExecutionsByEventID[eventID] = append(parentExecutionsByEventID[eventID], execution)
		}
	}

	if len(parentExecutions) == 0 {
		return parentExecutionsByEventID, map[string][]models.WorkflowNodeExecution{}, nil
	}

	childExecutions, err := models.FindChildExecutionsForMultiple(executionIDs(parentExecutions))
	if err != nil {
		return nil, nil, err
	}

	childExecutionsByEventID := make(map[string][]models.WorkflowNodeExecution, len(eventIDs))
	for _, execution := range childExecutions {
		eventID := execution.RootEventID.String()
		childExecutionsByEventID[eventID] = append(childExecutionsByEventID[eventID], execution)
	}

	return parentExecutionsByEventID, childExecutionsByEventID, nil
}
