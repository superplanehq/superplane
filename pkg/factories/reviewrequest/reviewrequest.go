// Package reviewrequest owns the Review request automation policy. Shared
// canvas infrastructure reaches it only through the lifecycle operations.
package reviewrequest

import (
	"errors"

	"github.com/superplanehq/superplane/pkg/factories/lifecycle"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	TemplateID         = "review-request"
	triggerComponent   = "onBroadcast"
	selectionNodeID    = "select-reviewers"
	requestNodeID      = "request-reviewers"
	broadcastEventType = "app.broadcast"
)

func init() {
	lifecycle.Register(lifecycle.Automation{
		Name:                "review request",
		ValidatePublication: validatePublication,
		AdmitEvent:          admitEvent,
		LinkRun:             linkRun,
		LoadAction:          loadAction,
		PrepareAction:       prepareAction,
		PrepareOutput:       prepareOutput,
		RetryEventData:      retryEventData,
	})
}

func installed(tx *gorm.DB, canvas *models.Canvas) (bool, error) {
	return models.CanvasHasFactoryAppTemplate(tx, canvas.ID, TemplateID)
}

// isSourceRejection reports source errors that a retry cannot fix.
func isSourceRejection(err error) bool {
	return errors.Is(err, models.ErrReviewRequestSourceInvalid) || errors.Is(err, models.ErrReviewRequestSuperseded)
}

func linkRun(tx *gorm.DB, canvas *models.Canvas, event *models.CanvasEvent, run *models.CanvasRun) error {
	err := models.MaybeLinkReviewRequestRun(tx, canvas, event, run)
	if isSourceRejection(err) {
		return lifecycle.Reject(err)
	}
	return err
}
