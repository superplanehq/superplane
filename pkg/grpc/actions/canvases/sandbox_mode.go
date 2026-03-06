package canvases

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func isCanvasSandboxModeEnabled(organizationID string) (bool, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return false, err
	}

	return models.IsCanvasSandboxModeEnabled(orgID)
}

func isCanvasSandboxModeEnabledInTransaction(tx *gorm.DB, organizationID string) (bool, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return false, err
	}

	return models.IsCanvasSandboxModeEnabledInTransaction(tx, orgID)
}
