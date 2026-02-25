package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const OrganizationAgentCredentialProviderOpenAI = "openai"

type OrganizationAgentCredential struct {
	ID               uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	OrganizationID   uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	Provider         string
	APIKeyCiphertext []byte
	EncryptionKeyID  string
	KeyLast4         string
	CreatedBy        *uuid.UUID `gorm:"type:uuid"`
	UpdatedBy        *uuid.UUID `gorm:"type:uuid"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (c *OrganizationAgentCredential) TableName() string {
	return "organization_agent_credentials"
}

func FindOrganizationAgentCredentialByOrganizationID(organizationID string) (*OrganizationAgentCredential, error) {
	return FindOrganizationAgentCredentialByOrganizationIDInTransaction(database.Conn(), organizationID)
}

func FindOrganizationAgentCredentialByOrganizationIDInTransaction(
	tx *gorm.DB,
	organizationID string,
) (*OrganizationAgentCredential, error) {
	var credential OrganizationAgentCredential

	err := tx.
		Where("organization_id = ?", organizationID).
		First(&credential).
		Error
	if err != nil {
		return nil, err
	}

	return &credential, nil
}

func UpsertOrganizationAgentCredential(credential *OrganizationAgentCredential) error {
	return UpsertOrganizationAgentCredentialInTransaction(database.Conn(), credential)
}

func UpsertOrganizationAgentCredentialInTransaction(tx *gorm.DB, credential *OrganizationAgentCredential) error {
	return tx.
		Clauses(
			clause.OnConflict{
				Columns: []clause.Column{{Name: "organization_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"provider",
					"api_key_ciphertext",
					"encryption_key_id",
					"key_last4",
					"updated_by",
					"updated_at",
				}),
			},
		).
		Create(credential).
		Error
}

func DeleteOrganizationAgentCredentialByOrganizationID(organizationID string) error {
	return DeleteOrganizationAgentCredentialByOrganizationIDInTransaction(database.Conn(), organizationID)
}

func DeleteOrganizationAgentCredentialByOrganizationIDInTransaction(tx *gorm.DB, organizationID string) error {
	return tx.
		Where("organization_id = ?", organizationID).
		Delete(&OrganizationAgentCredential{}).
		Error
}
