package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	OrganizationAgentOpenAIKeyStatusNotConfigured = "not_configured"
	OrganizationAgentOpenAIKeyStatusValid         = "valid"
	OrganizationAgentOpenAIKeyStatusInvalid       = "invalid"
	OrganizationAgentOpenAIKeyStatusUnchecked     = "unchecked"
)

type OrganizationAgentSettings struct {
	ID                       uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	OrganizationID           uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	AgentModeEnabled         bool
	OpenAIApiKeyCiphertext   []byte     `gorm:"column:openai_api_key_ciphertext"`
	OpenAIKeyEncryptionKeyID *string    `gorm:"column:openai_key_encryption_key_id"`
	OpenAIKeyLast4           *string    `gorm:"column:openai_key_last4"`
	OpenAIKeyStatus          string     `gorm:"column:openai_key_status"`
	OpenAIKeyValidatedAt     *time.Time `gorm:"column:openai_key_validated_at"`
	OpenAIKeyValidationError *string    `gorm:"column:openai_key_validation_error"`
	UpdatedBy                *uuid.UUID `gorm:"type:uuid"`
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func (s *OrganizationAgentSettings) TableName() string {
	return "organization_agent_settings"
}

func FindOrganizationAgentSettingsByOrganizationID(organizationID string) (*OrganizationAgentSettings, error) {
	return FindOrganizationAgentSettingsByOrganizationIDInTransaction(database.Conn(), organizationID)
}

func FindOrganizationAgentSettingsByOrganizationIDInTransaction(
	tx *gorm.DB,
	organizationID string,
) (*OrganizationAgentSettings, error) {
	var settings OrganizationAgentSettings

	err := tx.
		Where("organization_id = ?", organizationID).
		First(&settings).
		Error
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

func UpsertOrganizationAgentSettings(settings *OrganizationAgentSettings) error {
	return UpsertOrganizationAgentSettingsInTransaction(database.Conn(), settings)
}

func UpsertOrganizationAgentSettingsInTransaction(tx *gorm.DB, settings *OrganizationAgentSettings) error {
	return tx.
		Clauses(
			clause.OnConflict{
				Columns: []clause.Column{{Name: "organization_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"agent_mode_enabled",
					"openai_api_key_ciphertext",
					"openai_key_encryption_key_id",
					"openai_key_last4",
					"openai_key_status",
					"openai_key_validated_at",
					"openai_key_validation_error",
					"updated_by",
					"updated_at",
				}),
			},
		).
		Create(settings).
		Error
}
