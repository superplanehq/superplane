package workflows

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListWorkflowEvents(ctx context.Context, registry *registry.Registry, workflowID string, limit uint32, before *timestamppb.Timestamp) (*pb.ListWorkflowEventsResponse, error) {
	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, err
	}

	limit = getLimit(limit)
	beforeTime := getBefore(before)

	var events []models.WorkflowInitialEvent
	query := database.Conn().
		Where("workflow_id = ?", workflowUUID).
		Order("created_at DESC").
		Limit(int(limit))

	if beforeTime != nil {
		query = query.Where("created_at < ?", beforeTime)
	}

	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}

	var totalCount int64
	countQuery := database.Conn().
		Model(&models.WorkflowInitialEvent{}).
		Where("workflow_id = ?", workflowUUID)

	if err := countQuery.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	serialized, err := SerializeWorkflowInitialEvents(events)
	if err != nil {
		return nil, err
	}

	return &pb.ListWorkflowEventsResponse{
		Events:        serialized,
		TotalCount:    uint32(totalCount),
		HasNextPage:   hasNextPage(len(events), int(limit), totalCount),
		LastTimestamp: getLastEventTimestamp(events),
	}, nil
}

func SerializeWorkflowInitialEvents(events []models.WorkflowInitialEvent) ([]*pb.WorkflowInitialEvent, error) {
	result := make([]*pb.WorkflowInitialEvent, 0, len(events))

	for _, event := range events {
		data, err := structpb.NewStruct(event.Data.Data())
		if err != nil {
			return nil, err
		}

		result = append(result, &pb.WorkflowInitialEvent{
			Id:         event.ID.String(),
			WorkflowId: event.WorkflowID.String(),
			Data:       data,
			CreatedAt:  timestamppb.New(*event.CreatedAt),
		})
	}

	return result, nil
}

func getLastEventTimestamp(events []models.WorkflowInitialEvent) *timestamppb.Timestamp {
	if len(events) > 0 {
		return timestamppb.New(*events[len(events)-1].CreatedAt)
	}
	return nil
}
