package contexts

import (
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func SkipPausedIntakeFeed(tx *gorm.DB, canvasID uuid.UUID) (bool, error) {
	_, skip, err := intakeForFeed(tx, canvasID)
	return skip, err
}

func intakeForFeed(tx *gorm.DB, canvasID uuid.UUID) (*models.FactoryIntake, bool, error) {
	intake, err := models.FindFactoryIntakeByCanvasID(tx, canvasID)
	if err != nil {
		if errors.Is(err, models.ErrFactoryIntakeNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return intake, intake.Paused(), nil
}
