package canvases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

func UpdateCanvas(
	ctx context.Context,
	organizationID string,
	id string,
	name *string,
	description *string,
) (*pb.UpdateCanvasResponse, error) {
	canvasID, err := uuid.Parse(id)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	organizationUUID := uuid.MustParse(organizationID)

	canvas, err := models.FindCanvas(organizationUUID, canvasID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "canvas not found")
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		return updateCanvasInTransaction(tx, organizationUUID, canvasID, name, description)
	})

	if err != nil {
		if _, _, ok := grpcerrors.HandlerStatus(err); ok {
			return nil, err
		}
		return nil, err
	}

	if publishErr := messages.NewCanvasUpdatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishUpdated(); publishErr != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", publishErr)
	}

	refreshedCanvas, err := models.FindCanvas(organizationUUID, canvasID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load updated canvas")
	}

	var user *models.User
	if refreshedCanvas.CreatedBy != nil {
		user, err = models.FindMaybeDeletedUserByID(refreshedCanvas.OrganizationID.String(), refreshedCanvas.CreatedBy.String())
		if err != nil {
			return nil, err
		}
	}

	liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(ctx), refreshedCanvas)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load canvas spec")
	}

	serializedCanvas, err := SerializeCanvas(refreshedCanvas, liveVersion, user, nil)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to serialize canvas")
	}

	return &pb.UpdateCanvasResponse{Canvas: serializedCanvas}, nil
}

func updateCanvasInTransaction(
	tx *gorm.DB,
	organizationUUID uuid.UUID,
	canvasID uuid.UUID,
	name *string,
	description *string,
) error {
	canvas, err := models.FindCanvasInTransaction(tx, organizationUUID, canvasID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return grpcerrors.NotFound(err, "canvas not found")
		}
		return err
	}

	changed, err := applyCanvasMetadataUpdates(canvas, name, description)
	if err != nil {
		return err
	}

	if !changed {
		return nil
	}

	return saveCanvasMetadataUpdate(tx, canvas)
}

func applyCanvasMetadataUpdates(
	canvas *models.Canvas,
	name *string,
	description *string,
) (bool, error) {
	nameChanged, err := applyCanvasNameUpdate(canvas, name)
	if err != nil {
		return false, err
	}

	descriptionChanged := applyCanvasDescriptionUpdate(canvas, description)

	return nameChanged || descriptionChanged, nil
}

func applyCanvasNameUpdate(canvas *models.Canvas, name *string) (bool, error) {
	if name == nil {
		return false, nil
	}

	nextName := strings.TrimSpace(*name)
	if nextName == "" {
		return false, grpcerrors.InvalidArgument(nil, "canvas name is required")
	}

	if canvas.Name == nextName {
		return false, nil
	}

	canvas.Name = nextName
	return true, nil
}

func applyCanvasDescriptionUpdate(canvas *models.Canvas, description *string) bool {
	if description == nil || canvas.Description == *description {
		return false
	}

	canvas.Description = *description
	return true
}

func saveCanvasMetadataUpdate(tx *gorm.DB, canvas *models.Canvas) error {
	now := time.Now()
	canvas.UpdatedAt = &now

	if err := tx.Save(canvas).Error; err != nil {
		return mapCanvasNameUniqueConstraintError(err)
	}

	return nil
}
