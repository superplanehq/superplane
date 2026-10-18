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
