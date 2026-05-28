package canvases

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func BeginCanvasRepositoryProvisioning(canvasID uuid.UUID) (*models.CanvasRepository, error) {
	var locked *models.CanvasRepository

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		repository, err := models.LockPendingCanvasRepository(tx, canvasID)
		if err != nil {
			return err
		}

		if err := repository.MarkProvisioning(tx); err != nil {
			return err
		}

		locked = repository
		return nil
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return locked, nil
}

func CompleteCanvasRepositoryProvisioning(repository *models.CanvasRepository) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		return repository.MarkReady(tx)
	})
}

func FailCanvasRepositoryProvisioning(repository *models.CanvasRepository, provisionErr error) error {
	txErr := database.Conn().Transaction(func(tx *gorm.DB) error {
		return repository.MarkError(tx)
	})
	if txErr != nil {
		return fmt.Errorf("%w (failed to update repository status: %v)", provisionErr, txErr)
	}

	return provisionErr
}
