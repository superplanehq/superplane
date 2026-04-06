package usage

import (
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
)

func ReconcileCanvasCount(orgID string, usageServiceCount int32) {
	reconcileCanvasCount(orgID, usageServiceCount, publishCanvasCreated)
}

func reconcileCanvasCount(orgID string, usageServiceCount int32, publish func(canvasID, orgID string) error) {
	dbCount, err := models.CountCanvasesByOrganization(orgID)
	if err != nil {
		log.Warnf("Failed to count canvases for reconciliation in organization %s: %v", orgID, err)
		return
	}

	if int64(usageServiceCount) == dbCount {
		return
	}

	log.Infof(
		"Canvas count mismatch for organization %s: usage service=%d, database=%d. Re-enqueuing canvases.",
		orgID, usageServiceCount, dbCount,
	)

	canvases, err := models.ListCanvases(orgID, false)
	if err != nil {
		log.Warnf("Failed to list canvases for reconciliation in organization %s: %v", orgID, err)
		return
	}

	for _, canvas := range canvases {
		if err := publish(canvas.ID.String(), orgID); err != nil {
			log.Warnf("Failed to re-enqueue canvas %s for organization %s: %v", canvas.ID, orgID, err)
		}
	}
}

func publishCanvasCreated(canvasID, orgID string) error {
	return messages.NewCanvasCreatedMessage(canvasID, orgID).PublishCreated()
}
