package encryption

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

const auditValueError = "Some values do not use the target data key. Check worker logs and start a new migration."

var errNonTargetValue = errors.New("value does not use the target data key")

type Auditor struct {
	job       *models.ApplicationEncryptionMigration
	registry  *registry.Registry
	locations []models.EncryptedStorageLocation
}

func NewAuditor(job *models.ApplicationEncryptionMigration, registry *registry.Registry, locations []models.EncryptedStorageLocation) *Auditor {
	return &Auditor{job: job, registry: registry, locations: locations}
}

func (a *Auditor) ProcessBatch(tx *gorm.DB) ([]error, error) {
	page, locationIndex, err := readMigrationPage(tx, a.job, a.locations)
	if err != nil {
		return nil, err
	}

	var mismatches []error
	for _, row := range page {
		if err := a.auditRow(a.locations[locationIndex], row); err != nil {
			a.job.ErrorMessage = auditValueError
			mismatches = append(mismatches, fmt.Errorf("%s row %s: %w", a.job.StorageLocation, row.Cursor, err))
		}
		a.job.Cursor = row.Cursor
	}

	if len(page) < migrationBatchSize {
		a.job.Cursor = ""
		if locationIndex+1 < len(a.locations) {
			a.job.StorageLocation = a.locations[locationIndex+1].Name
		} else if a.job.ErrorMessage != "" {
			a.job.State = "failed"
		} else {
			a.job.State = "completed"
			now := time.Now()
			a.job.CompletedAt = &now
		}
	}
	return mismatches, nil
}

func (a *Auditor) auditRow(location models.EncryptedStorageLocation, row models.EncryptedStorageRow) error {
	if location.Format != models.EncryptedStorageIntegrationJSON {
		if len(row.Value) == 0 {
			return nil
		}
		return auditCiphertext(row.Value, a.job.TargetKeyID)
	}

	var configuration map[string]any
	if err := json.Unmarshal(row.Value, &configuration); err != nil {
		return fmt.Errorf("decode integration configuration: %w", err)
	}
	fields, err := sensitiveConfigurationFields(a.registry, row.AppName)
	if err != nil {
		return err
	}
	var mismatches []error
	for _, name := range fields {
		value, exists := configuration[name]
		if !exists {
			continue
		}
		encoded, ok := value.(string)
		if !ok {
			mismatches = append(mismatches, fmt.Errorf("sensitive field %s is not a string", name))
			continue
		}
		ciphertext, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			mismatches = append(mismatches, fmt.Errorf("decode sensitive field %s: %w", name, err))
			continue
		}
		if err := auditCiphertext(ciphertext, a.job.TargetKeyID); err != nil {
			mismatches = append(mismatches, fmt.Errorf("sensitive field %s: %w", name, err))
		}
	}
	return errors.Join(mismatches...)
}

func auditCiphertext(ciphertext []byte, target uuid.UUID) error {
	id, versioned, err := crypto.EncryptedDataKeyID(ciphertext)
	if err != nil {
		return fmt.Errorf("parse encrypted value: %w", err)
	}
	if !versioned || id != target {
		return errNonTargetValue
	}
	return nil
}
