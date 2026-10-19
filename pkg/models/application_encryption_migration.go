package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ApplicationEncryptionMigration struct {
	ID              uuid.UUID `gorm:"primaryKey"`
	TargetKeyID     uuid.UUID
	State           string
	RegistryVersion int
	StorageLocation string
	Cursor          string
	ErrorMessage    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CompletedAt     *time.Time
}

func (ApplicationEncryptionMigration) TableName() string {
	return "application_encryption_migrations"
}

func (m *ApplicationEncryptionMigration) Start(tx *gorm.DB) error {
	m.State = "pending"
	return tx.Create(m).Error
}

func (m *ApplicationEncryptionMigration) SaveProgress(tx *gorm.DB) error {
	return tx.Save(m).Error
}

func (m *ApplicationEncryptionMigration) Fail(tx *gorm.DB, cause error) error {
	m.State = "failed"
	m.ErrorMessage = cause.Error()
	return tx.Model(&ApplicationEncryptionMigration{}).
		Where("id = ?", m.ID).
		Updates(map[string]any{"state": m.State, "error_message": m.ErrorMessage}).Error
}

func LoadApplicationEncryptionMigration(tx *gorm.DB, id uuid.UUID) (*ApplicationEncryptionMigration, error) {
	var migration ApplicationEncryptionMigration
	if err := tx.First(&migration, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &migration, nil
}

func ListApplicationEncryptionMigrations(tx *gorm.DB) ([]ApplicationEncryptionMigration, error) {
	var migrations []ApplicationEncryptionMigration
	err := tx.Order("created_at DESC, id DESC").Limit(20).Find(&migrations).Error
	return migrations, err
}

func LockNextApplicationEncryptionMigration(tx *gorm.DB) (*ApplicationEncryptionMigration, error) {
	var migration ApplicationEncryptionMigration
	err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("state IN ?", []string{"pending", "running", "auditing"}).
		Order("created_at, id").First(&migration).Error
	if err != nil {
		return nil, err
	}
	return &migration, nil
}

func HasUnfinishedApplicationEncryptionMigration(tx *gorm.DB) (bool, error) {
	var count int64
	err := tx.Table("application_encryption_migrations").
		Where("state IN ?", []string{"pending", "running", "auditing"}).
		Count(&count).Error
	return count > 0, err
}
