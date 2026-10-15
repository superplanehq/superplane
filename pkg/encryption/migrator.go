package encryption

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

const encryptionRegistryVersion = 1
const migrationBatchSize = 100

var ErrMigrationAlreadyRunning = errors.New("an encryption migration is already unfinished")

type Migrator struct {
	encryptor *crypto.VersionedEncryptor
	registry  *registry.Registry
	locations []models.EncryptedStorageLocation
}

func NewMigrator(encryptor *crypto.VersionedEncryptor, registry *registry.Registry, locations []models.EncryptedStorageLocation) *Migrator {
	return &Migrator{encryptor: encryptor, registry: registry, locations: locations}
}

func (m *Migrator) Start(ctx context.Context) (*models.ApplicationEncryptionMigration, error) {
	var job *models.ApplicationEncryptionMigration
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		active, err := models.LockActiveApplicationEncryptionKey(tx)
		if err != nil {
			return err
		}
		unfinished, err := models.HasUnfinishedApplicationEncryptionMigration(tx)
		if err != nil {
			return err
		}
		if unfinished {
			return ErrMigrationAlreadyRunning
		}
		job = &models.ApplicationEncryptionMigration{
			ID: uuid.New(), TargetKeyID: active.ID, State: "pending",
			RegistryVersion: encryptionRegistryVersion,
			StorageLocation: m.locations[0].Name,
			Pass:            1,
		}
		return tx.Create(job).Error
	})
	return job, err
}

func (m *Migrator) Get(ctx context.Context, id uuid.UUID) (*models.ApplicationEncryptionMigration, error) {
	return models.LoadApplicationEncryptionMigration(database.DB(ctx), id)
}

func (m *Migrator) List(ctx context.Context) ([]models.ApplicationEncryptionMigration, error) {
	return models.ListApplicationEncryptionMigrations(database.DB(ctx))
}

func (m *Migrator) Retry(ctx context.Context, id uuid.UUID) (*models.ApplicationEncryptionMigration, error) {
	var job *models.ApplicationEncryptionMigration
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		job, err = models.LoadApplicationEncryptionMigration(tx, id)
		if err != nil {
			return err
		}
		if job.State != "failed" {
			return errors.New("only a failed encryption migration can be retried")
		}
		job.State = "pending"
		job.ErrorMessage = ""
		return tx.Save(job).Error
	})
	return job, err
}

// Run lets every worker pod attempt one short batch at a time. PostgreSQL row
// locking gives each batch to one pod and rolls it back if that pod exits.
func (m *Migrator) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := m.RunBatch(ctx); err != nil && ctx.Err() == nil {
			log.WithError(err).Error("application encryption migration batch failed")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (m *Migrator) RunBatch(ctx context.Context) error {
	return database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		job, err := models.LockNextApplicationEncryptionMigration(tx)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		next := *job
		if err := tx.Transaction(func(batch *gorm.DB) error {
			return m.processBatch(ctx, batch, &next)
		}); err != nil {
			return m.failJob(tx, job, err.Error())
		}
		return tx.Save(&next).Error
	})
}

func (m *Migrator) processBatch(ctx context.Context, tx *gorm.DB, job *models.ApplicationEncryptionMigration) error {
	if job.RegistryVersion != encryptionRegistryVersion {
		return errors.New("migration registry version is not supported")
	}
	locations := m.locations
	locationIndex := -1
	for i, location := range locations {
		if location.Name == job.StorageLocation {
			locationIndex = i
			break
		}
	}
	if locationIndex == -1 {
		return errors.New("migration storage location is not supported")
	}
	page, err := locations[locationIndex].ReadPage(tx, job.Cursor, migrationBatchSize)
	if err != nil {
		return fmt.Errorf("read %s: %w", job.StorageLocation, err)
	}
	for _, row := range page {
		changed, err := m.migrateRow(ctx, tx, locations[locationIndex], row, job.TargetKeyID)
		if err != nil {
			return fmt.Errorf("%s row %s: %w", job.StorageLocation, row.Cursor, err)
		}
		if changed && job.Pass >= 2 && job.Pass%2 == 0 {
			job.Pass++ // odd verification pass means at least one value changed
		}
		job.Cursor = row.Cursor
	}
	job.State = "running"
	if len(page) < migrationBatchSize {
		job.Cursor = ""
		if locationIndex+1 < len(locations) {
			job.StorageLocation = locations[locationIndex+1].Name
		} else if job.Pass == 1 || job.Pass%2 == 1 {
			job.Pass++
			job.StorageLocation = locations[0].Name
		} else {
			job.State = "completed"
			now := time.Now()
			job.CompletedAt = &now
		}
	}
	return nil
}

func (m *Migrator) failJob(tx *gorm.DB, job *models.ApplicationEncryptionMigration, message string) error {
	job.State = "failed"
	job.ErrorMessage = message
	return tx.Save(job).Error
}

func (m *Migrator) migrateRow(ctx context.Context, tx *gorm.DB, location models.EncryptedStorageLocation, row models.EncryptedStorageRow, target uuid.UUID) (bool, error) {
	switch location.Format {
	case models.EncryptedStorageIntegrationJSON:
		return m.migrateIntegrationConfiguration(ctx, tx, location, row, target)
	default:
		if len(row.Value) == 0 {
			return false, nil
		}
		return m.migrateCiphertext(ctx, tx, location, row, row.Value, target, func(value []byte) []byte { return value })
	}
}

func (m *Migrator) migrateCiphertext(ctx context.Context, tx *gorm.DB, location models.EncryptedStorageLocation, row models.EncryptedStorageRow, ciphertext []byte, target uuid.UUID, encode func([]byte) []byte) (bool, error) {
	id, versioned, err := crypto.EncryptedDataKeyID(ciphertext)
	if err != nil {
		return false, err
	}
	if versioned && id == target {
		return false, nil
	}
	plain, err := m.encryptor.Decrypt(ctx, ciphertext, []byte(row.AssociatedData))
	if err != nil {
		return false, err
	}
	replacement, err := m.encryptor.Encrypt(ctx, plain, []byte(row.AssociatedData))
	if err != nil {
		return false, err
	}
	actual, _, err := crypto.EncryptedDataKeyID(replacement)
	if err != nil || actual != target {
		return false, errors.New("active data key changed during migration")
	}
	return location.Replace(tx, row, encode(replacement))
}

func (m *Migrator) migrateIntegrationConfiguration(ctx context.Context, tx *gorm.DB, location models.EncryptedStorageLocation, row models.EncryptedStorageRow, target uuid.UUID) (bool, error) {
	var configuration map[string]any
	decoder := json.NewDecoder(bytes.NewReader(row.Value))
	decoder.UseNumber()
	if err := decoder.Decode(&configuration); err != nil {
		return false, fmt.Errorf("decode integration configuration: %w", err)
	}
	integration, err := m.registry.GetIntegration(row.AppName)
	if err != nil {
		return false, fmt.Errorf("load integration %s: %w", row.AppName, err)
	}
	fields := make([]string, 0)
	for _, field := range integration.Configuration() {
		if field.Sensitive {
			fields = append(fields, field.Name)
		}
	}
	// The factory BYOK path stores this field for OpenRouter even though its
	// regular integration configuration does not declare it.
	if row.AppName == "openrouter" && !slices.Contains(fields, "apiKey") {
		fields = append(fields, "apiKey")
	}
	changed := false
	for _, name := range fields {
		value, exists := configuration[name]
		if !exists {
			continue
		}
		encoded, ok := value.(string)
		if !ok {
			return false, fmt.Errorf("sensitive field %s is not a string", name)
		}
		ciphertext, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return false, fmt.Errorf("decode sensitive field %s: %w", name, err)
		}
		id, versioned, err := crypto.EncryptedDataKeyID(ciphertext)
		if err != nil {
			return false, err
		}
		if versioned && id == target {
			continue
		}
		plain, err := m.encryptor.Decrypt(ctx, ciphertext, []byte(row.AssociatedData))
		if err != nil {
			return false, fmt.Errorf("decrypt sensitive field %s: %w", name, err)
		}
		replacement, err := m.encryptor.Encrypt(ctx, plain, []byte(row.AssociatedData))
		if err != nil {
			return false, err
		}
		configuration[name] = base64.StdEncoding.EncodeToString(replacement)
		changed = true
	}
	if !changed {
		return false, nil
	}
	replacement, err := json.Marshal(configuration)
	if err != nil {
		return false, err
	}
	return location.Replace(tx, row, replacement)
}
