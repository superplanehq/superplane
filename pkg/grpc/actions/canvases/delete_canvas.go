package canvases

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

func DeleteCanvas(ctx context.Context, db *gorm.DB, canvas *models.Canvas) (*pb.DeleteCanvasResponse, error) {
	// Perform soft delete on the canvas with name suffix
	// The cleanup worker will handle the actual deletion of nodes and related data
	err := db.Transaction(func(tx *gorm.DB) error {
		current, err := models.LockCanvasForUpdate(tx, canvas.OrganizationID, canvas.ID)
		if err != nil {
			return err
		}
		version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, current)
		if err != nil {
			return err
		}
		if models.FactoryAppTemplateID(version.Nodes) == models.FactoryAppTemplateHumanReviewID {
			return grpcerrors.FailedPrecondition(nil, "Disable Request review in the automation settings")
		}
		return current.SoftDeleteInTransaction(tx)
	})
	if err != nil {
		if _, _, ok := grpcerrors.HandlerStatus(err); ok {
			return nil, err
		}
		log.Errorf("failed to delete canvas %s: %v", canvas.ID.String(), err)
		return nil, grpcerrors.Internal(err, "failed to delete canvas")
	}

	if err := messages.NewCanvasDeletedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishDeleted(); err != nil {
		log.Errorf("failed to publish canvas deleted RabbitMQ message: %v", err)
	}

	return &pb.DeleteCanvasResponse{}, nil
}
