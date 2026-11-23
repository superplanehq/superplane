package canvases

import (
	"context"
	"errors"

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
		if canvas.FactoryID != nil {
			if _, err := models.LockAutomationFactory(tx, canvas.OrganizationID, *canvas.FactoryID); err != nil {
				return err
			}
			source, err := models.FindAutomationSourceByArtifact(tx, canvas.OrganizationID, canvas.ID.String())
			if err == nil {
				if err := source.Delete(tx); err != nil {
					return err
				}
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		return canvas.SoftDeleteInTransaction(tx)
	})
	if err != nil {
		log.Errorf("failed to delete canvas %s: %v", canvas.ID.String(), err)
		return nil, grpcerrors.Internal(err, "failed to delete canvas")
	}

	PublishCanvasDeleted(canvas)
	return &pb.DeleteCanvasResponse{}, nil
}

func PublishCanvasDeleted(canvas *models.Canvas) {
	if err := messages.NewCanvasDeletedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishDeleted(); err != nil {
		log.Errorf("failed to publish canvas deleted RabbitMQ message: %v", err)
	}
}
