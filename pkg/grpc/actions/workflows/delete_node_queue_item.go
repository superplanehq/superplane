package workflows

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
)

// DeleteNodeQueueItem deletes a single queue item for a node within a workflow.
func DeleteNodeQueueItem(ctx context.Context, registry *registry.Registry, workflowID, nodeID, itemID string) (*pb.DeleteNodeQueueItemResponse, error) {
	wfID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, err
	}
	qID, err := uuid.Parse(itemID)
	if err != nil {
		return nil, err
	}

	// Ensure we only delete the item that belongs to this workflow and node
	if err := database.Conn().Where("id = ? AND workflow_id = ? AND node_id = ?", qID, wfID, nodeID).Delete(&models.WorkflowNodeQueueItem{}).Error; err != nil {
		return nil, err
	}

	return &pb.DeleteNodeQueueItemResponse{}, nil
}
