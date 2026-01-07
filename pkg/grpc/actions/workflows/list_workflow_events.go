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

	executionsByEventID, err := listExecutionSummariesForWorkflowEvents(events)
	if err != nil {
		return nil, err
	}

	serialized, err := SerializeWorkflowEventsWithExecutions(events, executionsByEventID)
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

func SerializeWorkflowEventsWithExecutions(events []models.WorkflowEvent, executionsByEventID map[string][]models.WorkflowNodeExecutionSummary) ([]*pb.WorkflowEventWithExecutions, error) {
	result := make([]*pb.WorkflowEventWithExecutions, 0, len(events))

	for _, event := range events {
		serializedEvent, err := SerializeWorkflowEventWithExecutions(event, executionsByEventID[event.ID.String()])
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
		Data:       s,
		CreatedAt:  timestamppb.New(*event.CreatedAt),
	}, nil
}

func SerializeWorkflowEventWithExecutions(event models.WorkflowEvent, executions []models.WorkflowNodeExecutionSummary) (*pb.WorkflowEventWithExecutions, error) {
	data, ok := event.Data.Data().(map[string]any)
	if !ok {
		return nil, fmt.Errorf("event data is not a map[string]any")
	}

	s, err := structpb.NewStruct(data)
	if err != nil {
		return nil, err
	}

	serializedExecutions := make([]*pb.WorkflowEventExecution, 0, len(executions))
	for _, execution := range executions {
		serializedExecutions = append(serializedExecutions, &pb.WorkflowEventExecution{
			Id:                  execution.ID.String(),
			WorkflowId:          execution.WorkflowID.String(),
			NodeId:              execution.NodeID,
			ParentExecutionId:   execution.GetParentExecutionID(),
			PreviousExecutionId: execution.GetPreviousExecutionID(),
			State:               NodeExecutionStateToProto(execution.State),
			Result:              NodeExecutionResultToProto(execution.Result),
			ResultReason:        NodeExecutionResultReasonToProto(execution.ResultReason),
			ResultMessage:       execution.ResultMessage,
		})
	}

	return &pb.WorkflowEventWithExecutions{
		Id:         event.ID.String(),
		WorkflowId: event.WorkflowID.String(),
		NodeId:     event.NodeID,
		Channel:    event.Channel,
		Data:       s,
		CreatedAt:  timestamppb.New(*event.CreatedAt),
		Executions: serializedExecutions,
	}, nil
}

func getLastEventTimestamp(events []models.WorkflowEvent) *timestamppb.Timestamp {
	if len(events) > 0 {
		return timestamppb.New(*events[len(events)-1].CreatedAt)
	}
	return nil
}

func listExecutionSummariesForWorkflowEvents(events []models.WorkflowEvent) (map[string][]models.WorkflowNodeExecutionSummary, error) {
	if len(events) == 0 {
		return map[string][]models.WorkflowNodeExecutionSummary{}, nil
	}

	eventIDs := make([]uuid.UUID, len(events))
	for i, event := range events {
		eventIDs[i] = event.ID
	}

	executions, err := models.ListNodeExecutionSummariesForRootEvents(eventIDs)
	if err != nil {
		return nil, err
	}

	executionsByEventID := make(map[string][]models.WorkflowNodeExecutionSummary, len(eventIDs))
	for _, execution := range executions {
		eventID := execution.RootEventID.String()
		executionsByEventID[eventID] = append(executionsByEventID[eventID], execution)
	}

	return executionsByEventID, nil
}
