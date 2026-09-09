package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	installationLLMSettingsID = 1

	DefaultWelcomeGrantCents   int64 = 5000
	DefaultMarkupBPS                 = 2000
	DefaultWarningThresholdBPS       = 2000
	MarkupBaseBPS                    = 10000
	MicrosPerCent              int64 = 10_000
)

const (
	DefaultWelcomeGrantTTLDays = 14
	DefaultWelcomeGrantTTL     = DefaultWelcomeGrantTTLDays * 24 * time.Hour
)

// InstallationLLMSettings is the singleton hosted-LLM catalog policy.
type InstallationLLMSettings struct {
	ID                    int `gorm:"primary_key"`
	WelcomeGrantCents     int64
	WelcomeGrantTTLDays   int
	MarkupBPS             int
	WarningThresholdBPS   int
	DefaultHostedProvider *string
	DefaultHostedModel    *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (s *InstallationLLMSettings) WelcomeGrantTTL() time.Duration {
	days := DefaultWelcomeGrantTTLDays
	if s != nil && s.WelcomeGrantTTLDays > 0 {
		days = s.WelcomeGrantTTLDays
	}
	return time.Duration(days) * 24 * time.Hour
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

	if settings.WelcomeGrantCents < 0 {
		return nil, errors.New("welcome grant cannot be negative")
	}
	ttlDays, err := resolveWelcomeGrantTTLDays(settings.WelcomeGrantTTLDays, current.WelcomeGrantTTLDays)
	if err != nil {
		return nil, err
	}
	if settings.MarkupBPS < 0 {
		return nil, errors.New("markup cannot be negative")
	}
	if settings.WarningThresholdBPS < 0 || settings.WarningThresholdBPS > MarkupBaseBPS {
		return nil, errors.New("warning threshold must be between 0 and 10000 basis points")
	}

	defaultModel, err := NormalizeDefaultHostedLLMModel(
		stringValue(settings.DefaultHostedProvider),
		stringValue(settings.DefaultHostedModel),
	)
	if err != nil {
		return nil, err
	}
	if err := AssertDefaultHostedLLMModelAllowed(tx, defaultModel); err != nil {
		return nil, err
	}

	now := time.Now()
	provider := stringPointer(defaultModel.Provider)
	model := stringPointer(defaultModel.Model)
	err = tx.Model(&InstallationLLMSettings{}).
		Where("id = ?", installationLLMSettingsID).
		Select(
			"welcome_grant_cents",
			"welcome_grant_ttl_days",
			"markup_bps",
			"warning_threshold_bps",
			"default_hosted_provider",
			"default_hosted_model",
			"updated_at",
		).
		Updates(map[string]any{
			"welcome_grant_cents":     settings.WelcomeGrantCents,
			"welcome_grant_ttl_days":  ttlDays,
			"markup_bps":              settings.MarkupBPS,
			"warning_threshold_bps":   settings.WarningThresholdBPS,
			"default_hosted_provider": provider,
			"default_hosted_model":    model,
			"updated_at":              now,
		}).Error
	if err != nil {
		return nil, err
	}

	current.WelcomeGrantCents = settings.WelcomeGrantCents
	current.WelcomeGrantTTLDays = ttlDays
	current.MarkupBPS = settings.MarkupBPS
	current.WarningThresholdBPS = settings.WarningThresholdBPS
	current.DefaultHostedProvider = provider
	current.DefaultHostedModel = model
	current.UpdatedAt = now
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

	now := time.Now()
	settings = InstallationLLMSettings{
		ID:                  installationLLMSettingsID,
		WelcomeGrantCents:   DefaultWelcomeGrantCents,
		WelcomeGrantTTLDays: DefaultWelcomeGrantTTLDays,
		MarkupBPS:           DefaultMarkupBPS,
		WarningThresholdBPS: DefaultWarningThresholdBPS,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&settings).Error; err != nil {
		return nil, err
	}

	if err := tx.Where("id = ?", installationLLMSettingsID).First(&settings).Error; err != nil {
		return nil, err
	}
	return &settings, nil
}

func resolveWelcomeGrantTTLDays(requested, current int) (int, error) {
	if requested < 0 {
		return 0, errors.New("welcome grant duration must be at least 1 day")
	}
	if requested == 0 {
		if current > 0 {
			return current, nil
		}
		return DefaultWelcomeGrantTTLDays, nil
	}
	return requested, nil
}

// ApplyMarkupMicros returns billed micros for SuperPlane-hosted credit.
func ApplyMarkupMicros(providerMicros int64, markupBPS int) int64 {
	if providerMicros <= 0 {
		return 0
	}
	if markupBPS <= 0 {
		return providerMicros
	}
	return providerMicros * int64(MarkupBaseBPS+markupBPS) / int64(MarkupBaseBPS)
}

func CentsToMicros(cents int64) int64 {
	if cents <= 0 {
		return 0
	}
	return cents * MicrosPerCent
}

// SignedMicrosToCents converts millionths of a dollar to whole cents and
// keeps the sign. Refunds store negative amounts.
func SignedMicrosToCents(micros int64) int64 {
	return micros / MicrosPerCent
}
