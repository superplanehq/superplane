package canvases

import (
	"context"
	goerrors "errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

func DeleteNodeQueueItem(ctx context.Context, registry *registry.Registry, workflowID, nodeID, itemID string) (*pb.DeleteNodeQueueItemResponse, error) {
	wfID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}
	qID, err := uuid.Parse(itemID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid item id")
	}

	var finishedRunIDs []uuid.UUID
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var item models.CanvasNodeQueueItem
		err := tx.
			Where("id = ? AND workflow_id = ? AND node_id = ?", qID, wfID, nodeID).
			First(&item).
			Error
		if goerrors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}

		if err != nil {
			return err
		}

		if err := tx.Delete(&item).Error; err != nil {
			return err
		}

		finishedRunIDs, err = models.FinishCanvasRunsWithNoOpenWork(tx, wfID, []uuid.UUID{item.RunID})
		return err
	})
	if err != nil {
		return nil, err
	}

	publishFinishedRunMessages(wfID, finishedRunIDs)

	return &pb.DeleteNodeQueueItemResponse{}, nil
}
