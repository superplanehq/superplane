package canvases

import (
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func isCanvasVersioningEnabledForCanvas(canvas *models.Canvas) (bool, error) {
	return isCanvasVersioningEnabledForCanvasInTransaction(database.Conn(), canvas)
}

func isCanvasVersioningEnabledForCanvasInTransaction(tx *gorm.DB, canvas *models.Canvas) (bool, error) {
	if canvas == nil {
		return false, nil
	}

	// Template canvases are not user-editable, but keep the value stable if needed.
	if canvas.IsTemplate {
		return canvas.VersioningEnabled, nil
	}

	organizationVersioningEnabled, err := models.IsCanvasVersioningEnabledInTransaction(tx, canvas.OrganizationID)
	if err != nil {
		return false, err
	}
	if organizationVersioningEnabled {
		return true, nil
	}

	return canvas.VersioningEnabled, nil
}
