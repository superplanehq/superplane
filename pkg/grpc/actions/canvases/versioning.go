package canvases

import (
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func isChangeManagementEnabledForCanvas(canvas *models.Canvas) (bool, error) {
	return isChangeManagementEnabledForCanvasInTransaction(database.Conn(), canvas)
}

func isChangeManagementEnabledForCanvasInTransaction(tx *gorm.DB, canvas *models.Canvas) (bool, error) {
	if canvas == nil {
		return false, nil
	}

	// Template canvases are read-only and never use change management.
	if canvas.IsTemplate {
		return false, nil
	}

	organizationChangeManagementEnabled, err := models.IsChangeManagementEnabledInTransaction(tx, canvas.OrganizationID)
	if err != nil {
		return false, err
	}
	if organizationChangeManagementEnabled {
		return true, nil
	}

	return canvas.ChangeManagementEnabled, nil
}
