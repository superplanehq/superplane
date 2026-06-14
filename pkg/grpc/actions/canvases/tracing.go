package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/telemetry"
)

func findCanvas(ctx context.Context, orgID, canvasID uuid.UUID) (*models.Canvas, error) {
	var canvas *models.Canvas
	err := telemetry.RunSpan(ctx, "canvases.find_canvas", func(ctx context.Context) error {
		var findErr error
		canvas, findErr = models.FindCanvasInTransaction(database.DB(ctx), orgID, canvasID)
		return findErr
	})
	if err != nil {
		return nil, err
	}

	return canvas, nil
}
