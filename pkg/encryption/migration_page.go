package encryption

import (
	"errors"
	"fmt"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const migrationBatchSize = 100

func readMigrationPage(tx *gorm.DB, job *models.ApplicationEncryptionMigration, locations []models.EncryptedStorageLocation) ([]models.EncryptedStorageRow, int, error) {
	if job.RegistryVersion != RegistryVersion {
		return nil, 0, errors.New("migration registry version is not supported")
	}
	for i, location := range locations {
		if location.Name != job.StorageLocation {
			continue
		}
		page, err := location.ReadPage(tx, job.Cursor, migrationBatchSize)
		if err != nil {
			return nil, 0, fmt.Errorf("read %s: %w", job.StorageLocation, err)
		}
		return page, i, nil
	}
	return nil, 0, errors.New("migration storage location is not supported")
}
