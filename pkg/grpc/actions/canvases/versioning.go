package canvases

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func isCanvasVersioningEnabled(organizationID string) (bool, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return false, err
	}

	return models.IsCanvasVersioningEnabled(orgID)
}

func isCanvasVersioningEnabledInTransaction(tx *gorm.DB, organizationID string) (bool, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return false, err
	}

	return models.IsCanvasVersioningEnabledInTransaction(tx, orgID)
}
