package columnapps

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// LinkRun establishes column-app authority before the event can admit agents.
// This is factory policy; the router owns only persistence and transitions.
func LinkRun(tx *gorm.DB, canvas *models.Canvas, event *models.CanvasEvent, run *models.CanvasRun) error {
	if canvas.FactoryID == nil {
		return nil
	}
	if err := models.MaybeLinkVisualEvidenceRun(tx, canvas, event, run); err != nil {
		return linkError("visual evidence", run, err)
	}
	if err := models.MaybeLinkCustomAutomationRun(tx, canvas, event, run); err != nil {
		return linkError("custom automation", run, err)
	}
	if err := mergeconfidence.LinkRun(tx, canvas, event, run); err != nil {
		return linkError("merge confidence", run, err)
	}
	return nil
}

func linkError(name string, run *models.CanvasRun, err error) error {
	if models.IsPermanentSubscriptionAdmissionError(err) {
		return err
	}
	return fmt.Errorf("link %s run %s: %w", name, run.ID, err)
}
