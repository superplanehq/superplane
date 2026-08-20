package workers

import (
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// rollUpFactoryUsage copies ledger totals into the factory step cache.
// Child runs created by ctx.Runs.Create keep a different run ID than the
// factory line step, so lookup walks ancestor runs.
func rollUpFactoryUsage(tx *gorm.DB, runID uuid.UUID) error {
	execution, err := models.FindWorkOrderExecutionForRun(tx, runID)
	if err != nil {
		if errors.Is(err, models.ErrFactoryWorkOrderExecutionNotFound) {
			return nil
		}
		return err
	}

	return execution.RollupUsage(tx)
}

func rollUpFactoryUsageBestEffort(logger *log.Entry, tx *gorm.DB, runID uuid.UUID) {
	if err := rollUpFactoryUsage(tx, runID); err != nil {
		logger.WithError(err).WithField("run_id", runID).Error("failed to roll up factory usage")
	}
}
