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
	db *gorm.DB,
	canvas *models.Canvas,
	name *string,
	description *string,
	dismissAgentSuggestionID *string,
) (*pb.UpdateCanvasResponse, error) {
	err := db.Transaction(func(tx *gorm.DB) error {
		return updateCanvasInTransaction(tx, canvas.OrganizationID, canvas.ID, name, description, dismissAgentSuggestionID)
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

	refreshedCanvas, err := models.FindCanvasInTransaction(db, canvas.OrganizationID, canvas.ID)
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

	serializedCanvas, err := serializePreparedCanvas(db, refreshedCanvas, liveVersion, user, nil)
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
	dismissAgentSuggestionID *string,
) error {
	canvas, err := models.LockCanvasForUpdate(tx, organizationUUID, canvasID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return grpcerrors.NotFound(err, "canvas not found")
		}
		return err
	}

	updates, err := buildCanvasMetadataUpdates(canvas, name, description)
	if err != nil {
		return err
	}

	if dismissAgentSuggestionID != nil {
		suggestionID := strings.TrimSpace(*dismissAgentSuggestionID)
		if suggestionID == "" {
			return grpcerrors.InvalidArgument(nil, "dismiss_agent_suggestion_id cannot be empty")
		}
		if err := canvas.DismissAgentSuggestion(tx, suggestionID); err != nil {
			return err
		}
	}

	if len(updates) == 0 {
		return nil
	}

	updates["updated_at"] = time.Now()

	return mapCanvasNameUniqueConstraintError(
		tx.Model(&models.Canvas{}).
			Where("organization_id = ? AND id = ?", organizationUUID, canvasID).
			Updates(updates).Error,
	)
}

func buildCanvasMetadataUpdates(canvas *models.Canvas, name *string, description *string) (map[string]any, error) {
	updates := map[string]any{}

	if name != nil {
		nextName := strings.TrimSpace(*name)
		if nextName == "" {
			return nil, grpcerrors.InvalidArgument(nil, "canvas name is required")
		}
		if canvas.Name != nextName {
			updates["name"] = nextName
		}
	}

	if description != nil && canvas.Description != *description {
		updates["description"] = *description
	}

	return updates, nil
}
