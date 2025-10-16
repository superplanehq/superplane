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

func ListWorkflowEvents(ctx context.Context, registry *registry.Registry, workflowID string, nodeID string, limit uint32, before *timestamppb.Timestamp) (*pb.ListWorkflowEventsResponse, error) {
	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, err
	}

	limit = getLimit(limit)
	beforeTime := getBefore(before)
	events, err := models.ListWorkflowEventsForNode(workflowUUID, nodeID, int(limit), beforeTime)
	if err != nil {
		return nil, err
	}

	count, err := models.CountWorkflowEventsForNode(workflowUUID, nodeID)
	if err != nil {
		return nil, err
	}

	serialized, err := SerializeWorkflowInitialEvents(events)
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

func SerializeWorkflowInitialEvents(events []models.WorkflowEvent) ([]*pb.WorkflowEvent, error) {
	result := make([]*pb.WorkflowEvent, 0, len(events))

	for _, event := range events {
		data, ok := event.Data.Data().(map[string]any)
		if !ok {
			return nil, fmt.Errorf("event data is not a map[string]any")
		}

		s, err := structpb.NewStruct(data)
		if err != nil {
			return nil, err
		}

		result = append(result, &pb.WorkflowEvent{
			Id:         event.ID.String(),
			WorkflowId: event.WorkflowID.String(),
			NodeId:     event.NodeID,
			Channel:    event.Channel,
			Data:       s,
			CreatedAt:  timestamppb.New(*event.CreatedAt),
		})
	}

	return result, nil
}

func getLastEventTimestamp(events []models.WorkflowEvent) *timestamppb.Timestamp {
	if len(events) > 0 {
		return timestamppb.New(*events[len(events)-1].CreatedAt)
	}
	return nil
}
