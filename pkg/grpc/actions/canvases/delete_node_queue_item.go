package canvases

import (
	"context"
	goerrors "errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
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

	var runID uuid.UUID
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

		runID = item.RunID
		return nil
	})
	if err != nil {
		return nil, err
	}

	if runID != uuid.Nil {
		message := messages.NewCanvasQueueItemDeletedMessage(wfID.String(), qID.String(), nodeID, runID.String())
		if err := message.PublishDeleted(); err != nil {
			log.Errorf("failed to publish queue item deleted RabbitMQ message: %v", err)
		}
	}

	return &pb.DeleteNodeQueueItemResponse{}, nil
}
