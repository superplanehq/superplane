package contexts

import (
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func SkipPausedIntakeFeed(tx *gorm.DB, canvasID uuid.UUID) (bool, error) {
	intake, err := models.FindFactoryIntakeByCanvasID(tx, canvasID)
	if err != nil {
		if errors.Is(err, models.ErrFactoryIntakeNotFound) {
			return false, nil
		}
		return false, err
	}
	return intake.Paused(), nil
}
