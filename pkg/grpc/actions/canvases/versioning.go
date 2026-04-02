package canvases

import (
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func isCanvasVersioningEnabledForCanvas(canvas *models.Canvas) (bool, error) {
	return isCanvasVersioningEnabledForCanvasInTransaction(nil, canvas)
}

func isCanvasVersioningEnabledForCanvasInTransaction(_ *gorm.DB, canvas *models.Canvas) (bool, error) {
	if canvas == nil {
		return false, nil
	}

	// Template canvases are not user-editable; respect their individual flag.
	if canvas.IsTemplate {
		return canvas.VersioningEnabled, nil
	}

	// Versioning is always enabled for non-template canvases.
	return true, nil
}
