package canvases

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/canvas/changerequests"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func RefreshOpenCanvasChangeRequestsInTransaction(
	tx *gorm.DB,
	organizationID uuid.UUID,
	canvasID uuid.UUID,
	skipRequestID uuid.UUID,
) error {
	return changerequests.RefreshOpenCanvasChangeRequestsInTransaction(tx, organizationID, canvasID, skipRequestID)
}

func refreshCanvasChangeRequestDiffInTransaction(
	tx *gorm.DB,
	canvas *models.Canvas,
	version *models.CanvasVersion,
	request *models.CanvasChangeRequest,
) error {
	return changerequests.RefreshCanvasChangeRequestDiffInTransaction(tx, canvas, version, request)
}
