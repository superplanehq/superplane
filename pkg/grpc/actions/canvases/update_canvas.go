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
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	lockedCanvas, err := lockCanvasForUpdate(tx, organizationUUID, canvasID)
	if err != nil {
		return err
	}

	liveVersion, err := models.FindLiveCanvasVersionInTransaction(tx, canvasID)
	if err != nil {
		return err
	}

	changed, err := applyCanvasLiveVersionUpdates(
		tx,
		organizationUUID,
		canvasID,
		liveVersion,
		name,
		description,
	)
	if err != nil {
		return err
	}

	if !changed {
		return nil
	}

	return saveCanvasMetadataUpdate(tx, lockedCanvas, liveVersion)
}

func lockCanvasForUpdate(tx *gorm.DB, organizationUUID, canvasID uuid.UUID) (*models.Canvas, error) {
	lockedCanvas := &models.Canvas{}
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		// This locks workflows directly, so select only columns that physically
		// exist on workflows; metadata fields are projected from live versions.
		Select(
			"id",
			"organization_id",
			"live_version_id",
			"folder_id",
			"name",
			"created_by",
			"created_at",
			"updated_at",
			"deleted_at",
		).
		Where("organization_id = ?", organizationUUID).
		Where("id = ?", canvasID).
		First(lockedCanvas).
		Error
	if err != nil {
		return nil, err
	}

	return lockedCanvas, nil
}

func applyCanvasLiveVersionUpdates(
	tx *gorm.DB,
	organizationUUID uuid.UUID,
	canvasID uuid.UUID,
	liveVersion *models.CanvasVersion,
	name *string,
	description *string,
) (bool, error) {
	nameChanged, err := applyCanvasNameUpdate(tx, organizationUUID, canvasID, liveVersion, name)
	if err != nil {
		return false, err
	}

	descriptionChanged := applyCanvasDescriptionUpdate(liveVersion, description)

	return nameChanged || descriptionChanged, nil
}

func applyCanvasNameUpdate(
	tx *gorm.DB,
	organizationUUID uuid.UUID,
	canvasID uuid.UUID,
	liveVersion *models.CanvasVersion,
	name *string,
) (bool, error) {
	if name == nil {
		return false, nil
	}

	nextName := strings.TrimSpace(*name)
	if nextName == "" {
		return false, grpcerrors.InvalidArgument(nil, "canvas name is required")
	}

	nameErr := ensureCanvasNameAvailableInTransaction(tx, organizationUUID, canvasID, nextName)
	if errors.Is(nameErr, models.ErrCanvasNameAlreadyExists) {
		return false, grpcerrors.AlreadyExists(nil, "Canvas with the same name already exists")
	}
	if nameErr != nil {
		return false, nameErr
	}

	if liveVersion.Name == nextName {
		return false, nil
	}

	liveVersion.Name = nextName
	return true, nil
}

func applyCanvasDescriptionUpdate(liveVersion *models.CanvasVersion, description *string) bool {
	if description == nil || liveVersion.Description == *description {
		return false
	}

	liveVersion.Description = *description
	return true
}

func saveCanvasMetadataUpdate(tx *gorm.DB, lockedCanvas *models.Canvas, liveVersion *models.CanvasVersion) error {
	now := time.Now()
	liveVersion.UpdatedAt = &now
	lockedCanvas.Name = liveVersion.Name
	lockedCanvas.UpdatedAt = &now

	if err := tx.Save(liveVersion).Error; err != nil {
		return err
	}

	if err := tx.Save(lockedCanvas).Error; err != nil {
		return mapCanvasNameUniqueConstraintError(err)
	}

	return nil
}
