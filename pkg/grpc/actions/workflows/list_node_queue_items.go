package workflows

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	MinLimit     = 50
	MaxLimit     = 100
	DefaultLimit = 50
)

func ListNodeQueueItems(ctx context.Context, registry *registry.Registry, workflowID, nodeID string, limit uint32, before *timestamppb.Timestamp) (*pb.ListNodeQueueItemsResponse, error) {
	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, err
	}

	limit = getLimit(limit)
	beforeTime := getBefore(before)

	var queueItems []models.WorkflowQueueItem
	query := database.Conn().
		Where("workflow_id = ?", workflowUUID).
		Where("node_id = ?", nodeID).
		Order("created_at DESC").
		Limit(int(limit))

	if beforeTime != nil {
		query = query.Where("created_at < ?", beforeTime)
	}

	if err := query.Find(&queueItems).Error; err != nil {
		return nil, err
	}

	var totalCount int64
	countQuery := database.Conn().
		Model(&models.WorkflowQueueItem{}).
		Where("workflow_id = ?", workflowUUID).
		Where("node_id = ?", nodeID)

	if err := countQuery.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	serialized, err := serializeQueueItems(queueItems)
	if err != nil {
		return nil, err
	}

	return &pb.ListNodeQueueItemsResponse{
		QueueItems:    serialized,
		TotalCount:    uint32(totalCount),
		HasNextPage:   hasNextPage(len(queueItems), int(limit), totalCount),
		LastTimestamp: getLastTimestamp(queueItems),
	}, nil
}

func serializeQueueItems(items []models.WorkflowQueueItem) ([]*pb.WorkflowQueueItem, error) {
	result := make([]*pb.WorkflowQueueItem, 0, len(items))

	for _, item := range items {
		var event models.WorkflowEvent
		if err := database.Conn().First(&event, "id = ?", item.EventID).Error; err != nil {
			return nil, err
		}

		result = append(result, &pb.WorkflowQueueItem{
			EventId:   item.EventID.String(),
			CreatedAt: timestamppb.New(*item.CreatedAt),
			Event:     SerializeWorkflowEvent(&event),
		})
	}

	return result, nil
}

func getLimit(limit uint32) uint32 {
	if limit == 0 {
		return DefaultLimit
	}

	if limit > MaxLimit {
		return MaxLimit
	}

	return limit
}

func getBefore(before *timestamppb.Timestamp) *time.Time {
	if before != nil && before.IsValid() {
		t := before.AsTime()
		return &t
	}

	return nil
}

func hasNextPage(resultCount int, limit int, totalCount int64) bool {
	return resultCount == limit && totalCount > int64(limit)
}

func getLastTimestamp(items []models.WorkflowQueueItem) *timestamppb.Timestamp {
	if len(items) > 0 {
		return timestamppb.New(*items[len(items)-1].CreatedAt)
	}
	return nil
}
