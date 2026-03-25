package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteNodeQueueItem(ctx context.Context, registry *registry.Registry, workflowID, nodeID, itemID string) (*pb.DeleteNodeQueueItemResponse, error) {
	wfID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}
	qID, err := uuid.Parse(itemID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item id: %v", err)
	}

	// Ensure we only delete the item that belongs to this workflow and node
	if err := database.Conn().Where("id = ? AND workflow_id = ? AND node_id = ?", qID, wfID, nodeID).Delete(&models.CanvasNodeQueueItem{}).Error; err != nil {
		return nil, err
	}

	return &pb.DeleteNodeQueueItemResponse{}, nil
}
