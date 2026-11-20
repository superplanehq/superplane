package mergeconfidence

import (
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// LinkRun records task authority before agent admission, including owner and model checks.
func LinkRun(tx *gorm.DB, canvas *models.Canvas, event *models.CanvasEvent, run *models.CanvasRun) error {
	return models.LinkFactoryColumnAppRun(tx, canvas, event, run, "risk-score", "Merge confidence")
}
