package encryption

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

const RegistryVersion = 1

var errUnreadableValue = errors.New("encrypted value cannot be read")

const migrationValueError = "Some values could not be migrated. Check worker logs and start a new migration."

type Migrator struct {
	job       *models.ApplicationEncryptionMigration
	encryptor *crypto.VersionedEncryptor
	registry  *registry.Registry
	locations []models.EncryptedStorageLocation
}

func NewMigrator(job *models.ApplicationEncryptionMigration, encryptor *crypto.VersionedEncryptor, registry *registry.Registry, locations []models.EncryptedStorageLocation) *Migrator {
	return &Migrator{job: job, encryptor: encryptor, registry: registry, locations: locations}
}

func (m *Migrator) ProcessBatch(ctx context.Context, tx *gorm.DB) ([]error, error) {
	page, locationIndex, err := readMigrationPage(tx, m.job, m.locations)
	if err != nil {
		return nil, err
	}

	var unreadable []error
	for _, row := range page {
		err := m.migrateRow(ctx, tx, m.locations[locationIndex], row)
		if err != nil {
			if !errors.Is(err, errUnreadableValue) {
				return nil, fmt.Errorf("%s row %s: %w", m.job.StorageLocation, row.Cursor, err)
			}
			m.job.ErrorMessage = migrationValueError
			unreadable = append(unreadable, fmt.Errorf("%s row %s: %w", m.job.StorageLocation, row.Cursor, err))
		}
		m.job.Cursor = row.Cursor
	}

	if m.job.State == "pending" {
		m.job.State = "running"
	}
	if len(page) < migrationBatchSize {
		m.job.Cursor = ""
		if locationIndex+1 < len(m.locations) {
			m.job.StorageLocation = m.locations[locationIndex+1].Name
		} else if m.job.ErrorMessage != "" {
			m.job.State = "failed"
		} else {
			m.job.State = "auditing"
			m.job.StorageLocation = m.locations[0].Name
		}
	}
	return unreadable, nil
}
func (m *Migrator) migrateRow(ctx context.Context, tx *gorm.DB, location models.EncryptedStorageLocation, row models.EncryptedStorageRow) error {
	switch location.Format {
	case models.EncryptedStorageIntegrationJSON:
		return m.migrateIntegrationConfiguration(ctx, tx, location, row)
	default:
		if len(row.Value) == 0 {
			return nil
		}
		return m.migrateCiphertext(ctx, tx, location, row)
	}
}

func (m *Migrator) migrateCiphertext(ctx context.Context, tx *gorm.DB, location models.EncryptedStorageLocation, row models.EncryptedStorageRow) error {
	id, versioned, err := crypto.EncryptedDataKeyID(row.Value)
	if err != nil {
		return fmt.Errorf("%w: parse encrypted value: %w", errUnreadableValue, err)
	}
	if versioned && id == m.job.TargetKeyID {
		return nil
	}
	plain, err := m.encryptor.Decrypt(ctx, row.Value, []byte(row.AssociatedData))
	if err != nil {
		return fmt.Errorf("%w: decrypt value: %w", errUnreadableValue, err)
	}
	replacement, err := m.encryptor.Encrypt(ctx, plain, []byte(row.AssociatedData))
	if err != nil {
		return err
	}
	actual, _, err := crypto.EncryptedDataKeyID(replacement)
	if err != nil || actual != m.job.TargetKeyID {
		return errors.New("active data key changed during migration")
	}
	_, err = location.Replace(tx, row, replacement)
	return err
}

func (m *Migrator) migrateIntegrationConfiguration(ctx context.Context, tx *gorm.DB, location models.EncryptedStorageLocation, row models.EncryptedStorageRow) error {
	var configuration map[string]any
	decoder := json.NewDecoder(bytes.NewReader(row.Value))
	decoder.UseNumber()
	if err := decoder.Decode(&configuration); err != nil {
		return fmt.Errorf("%w: decode integration configuration: %w", errUnreadableValue, err)
	}
	fields, err := sensitiveConfigurationFields(m.registry, row.AppName)
	if err != nil {
		return fmt.Errorf("%w: %w", errUnreadableValue, err)
	}
	changed := false
	var unreadable []error
	for _, name := range fields {
		value, exists := configuration[name]
		if !exists {
			continue
		}
		encoded, ok := value.(string)
		if !ok {
			unreadable = append(unreadable, fmt.Errorf("%w: sensitive field %s is not a string", errUnreadableValue, name))
			continue
		}
		ciphertext, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			unreadable = append(unreadable, fmt.Errorf("%w: decode sensitive field %s: %w", errUnreadableValue, name, err))
			continue
		}
		id, versioned, err := crypto.EncryptedDataKeyID(ciphertext)
		if err != nil {
			unreadable = append(unreadable, fmt.Errorf("%w: parse sensitive field %s: %w", errUnreadableValue, name, err))
			continue
		}
		if versioned && id == m.job.TargetKeyID {
			continue
		}
		plain, err := m.encryptor.Decrypt(ctx, ciphertext, []byte(row.AssociatedData))
		if err != nil {
			unreadable = append(unreadable, fmt.Errorf("%w: decrypt sensitive field %s: %w", errUnreadableValue, name, err))
			continue
		}
		replacement, err := m.encryptor.Encrypt(ctx, plain, []byte(row.AssociatedData))
		if err != nil {
			return err
		}
		configuration[name] = base64.StdEncoding.EncodeToString(replacement)
		changed = true
	}
	if !changed {
		return errors.Join(unreadable...)
	}
	replacement, err := json.Marshal(configuration)
	if err != nil {
		return err
	}
	_, err = location.Replace(tx, row, replacement)
	if err != nil {
		return err
	}
	return errors.Join(unreadable...)
}
