package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func UpdateCanvasPause(ctx context.Context, registry *registry.Registry, canvasID string, paused bool) (*pb.UpdateCanvasPauseResponse, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas_id")
	}

	var canvas *models.Canvas
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		canvas, err = lockCanvasForUpdate(tx, canvasUUID)
		if err != nil {
			return err
		}

		if canvas.Paused == paused {
			return nil
		}

		canvas.Paused = paused
		if err := tx.Model(canvas).Update("paused", paused).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas not found")
		}
		return nil, err
	}

	serialized, err := SerializeCanvas(canvas, false, nil)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateCanvasPauseResponse{
		Canvas: serialized,
	}, nil
}

func lockCanvasForUpdate(tx *gorm.DB, canvasID uuid.UUID) (*models.Canvas, error) {
	lockedCanvas := &models.Canvas{}
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select(
			"id",
			"organization_id",
			"live_version_id",
			"is_template",
			"name",
			"paused",
			"created_by",
			"created_at",
			"updated_at",
			"deleted_at",
		).
		Where("id = ?", canvasID).
		First(lockedCanvas).
		Error
	if err != nil {
		return nil, err
	}

	if lockedCanvas.LiveVersionID != nil {
		liveVersion, err := models.FindCanvasVersionInTransaction(tx, lockedCanvas.ID, *lockedCanvas.LiveVersionID)
		if err != nil {
			return nil, err
		}

		lockedCanvas.Name = liveVersion.Name
		lockedCanvas.Description = liveVersion.Description
		lockedCanvas.ChangeManagementEnabled = liveVersion.ChangeManagementEnabled
		lockedCanvas.ChangeRequestApprovers = datatypes.NewJSONSlice(liveVersion.EffectiveChangeRequestApprovers())
	}

	return lockedCanvas, nil
}
