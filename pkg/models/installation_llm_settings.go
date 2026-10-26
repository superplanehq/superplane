package models

import (
	"errors"
	"time"

	"github.com/superplanehq/superplane/pkg/billing"
	"gorm.io/gorm"
)

const installationLLMSettingsID = 1

// InstallationLLMSettings is the singleton hosted-provider selection policy.
// Billing policy shares its table and owns separate columns.
type InstallationLLMSettings struct {
	ID                    int `gorm:"primary_key"`
	DefaultHostedProvider *string
	DefaultHostedModel    *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (InstallationLLMSettings) TableName() string {
	return "installation_llm_settings"
}

func GetInstallationLLMSettings(tx *gorm.DB) (*InstallationLLMSettings, error) {
	return findOrCreateInstallationLLMSettings(tx)
}

func UpdateInstallationLLMSettings(tx *gorm.DB, settings InstallationLLMSettings) (*InstallationLLMSettings, error) {
	current, err := findOrCreateInstallationLLMSettings(tx)
	if err != nil {
		return nil, err
	}
	defaultModel, err := NormalizeDefaultHostedLLMModel(stringValue(settings.DefaultHostedProvider), stringValue(settings.DefaultHostedModel))
	if err != nil {
		return nil, err
	}
	if err := AssertDefaultHostedLLMModelAllowed(tx, defaultModel); err != nil {
		return nil, err
	}
	now := time.Now()
	provider, model := stringPointer(defaultModel.Provider), stringPointer(defaultModel.Model)
	err = tx.Model(&InstallationLLMSettings{}).Where("id = ?", installationLLMSettingsID).
		Updates(map[string]any{"default_hosted_provider": provider, "default_hosted_model": model, "updated_at": now}).Error
	if err != nil {
		return nil, err
	}
	current.DefaultHostedProvider, current.DefaultHostedModel, current.UpdatedAt = provider, model, now
	return current, nil
}

func findOrCreateInstallationLLMSettings(tx *gorm.DB) (*InstallationLLMSettings, error) {
	var settings InstallationLLMSettings
	err := tx.Where("id = ?", installationLLMSettingsID).First(&settings).Error
	if err == nil {
		return &settings, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	// Billing creates the shared row with the existing singleton defaults.
	// Database defaults can differ from the application defaults on older installs.
	if _, err := billing.GetInstallationSettings(tx); err != nil {
		return nil, err
	}
	if err := tx.Where("id = ?", installationLLMSettingsID).First(&settings).Error; err != nil {
		return nil, err
	}
	return &settings, nil
}
