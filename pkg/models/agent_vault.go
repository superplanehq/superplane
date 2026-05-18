package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

type AgentVault struct {
	ID                uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	UserID            uuid.UUID `gorm:"index:idx_agent_vaults_user_org,priority:1"`
	OrganizationID    uuid.UUID `gorm:"index:idx_agent_vaults_user_org,priority:2"`
	ProviderVaultID   string    `gorm:"not null"` // e.g., Anthropic vault ID
	ProviderName      string    `gorm:"not null"` // e.g., "anthropic"
	CredentialID      string    // Provider's credential ID within the vault
	MCPServerURL      string    `gorm:"not null"` // URL the credential is bound to
	CreatedAt         *time.Time
	UpdatedAt         *time.Time
}

func (AgentVault) TableName() string { return "agent_vaults" }

// FindOrCreateAgentVault returns the existing vault for user+org+provider, or creates one if not found.
func FindOrCreateAgentVault(tx *gorm.DB, userID, organizationID uuid.UUID, providerName, providerVaultID, credentialID, mcpServerURL string) (*AgentVault, error) {
	var vault AgentVault
	err := tx.Where("user_id = ? AND organization_id = ? AND provider_name = ?", userID, organizationID, providerName).
		First(&vault).Error

	if err == nil {
		// Found existing vault — update if needed
		if vault.ProviderVaultID != providerVaultID || vault.CredentialID != credentialID || vault.MCPServerURL != mcpServerURL {
			vault.ProviderVaultID = providerVaultID
			vault.CredentialID = credentialID
			vault.MCPServerURL = mcpServerURL
			if err := tx.Save(&vault).Error; err != nil {
				return nil, err
			}
		}
		return &vault, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Create new vault record
	vault = AgentVault{
		UserID:          userID,
		OrganizationID:  organizationID,
		ProviderName:    providerName,
		ProviderVaultID: providerVaultID,
		CredentialID:    credentialID,
		MCPServerURL:    mcpServerURL,
	}
	if err := tx.Create(&vault).Error; err != nil {
		return nil, err
	}
	return &vault, nil
}

// FindAgentVaultForUser returns the vault for the given user+org+provider.
func FindAgentVaultForUser(userID, organizationID uuid.UUID, providerName string) (*AgentVault, error) {
	var vault AgentVault
	err := database.Conn().
		Where("user_id = ? AND organization_id = ? AND provider_name = ?", userID, organizationID, providerName).
		First(&vault).Error
	if err != nil {
		return nil, err
	}
	return &vault, nil
}
