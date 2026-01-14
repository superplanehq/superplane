package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const EmailProviderSMTP = "smtp"

type EmailSettings struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Provider      string    `gorm:"type:varchar(50);uniqueIndex"`
	SMTPHost      string
	SMTPPort      int
	SMTPUsername  string
	SMTPPassword  []byte
	SMTPFromName  string
	SMTPFromEmail string
	SMTPUseTLS    bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func UpsertEmailSettings(settings *EmailSettings) error {
	return UpsertEmailSettingsInTransaction(database.Conn(), settings)
}

func UpsertEmailSettingsInTransaction(tx *gorm.DB, settings *EmailSettings) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "provider"}},
		UpdateAll: true,
	}).Create(settings).Error
}

func FindEmailSettings(provider string) (*EmailSettings, error) {
	return FindEmailSettingsInTransaction(database.Conn(), provider)
}

func FindEmailSettingsInTransaction(tx *gorm.DB, provider string) (*EmailSettings, error) {
	var settings EmailSettings
	err := tx.
		Where("provider = ?", provider).
		First(&settings).
		Error
	if err != nil {
		return nil, err
	}

	return &settings, nil
}
