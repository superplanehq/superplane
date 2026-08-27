package models

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var hostedLLMProviders = []string{
	UsageProviderAnthropic,
	UsageProviderOpenAI,
	UsageProviderOpenRouter,
}

var (
	ErrHostedLLMProviderNotFound = errors.New("hosted llm provider is not configured")
	ErrHostedLLMProviderDisabled = errors.New("hosted llm provider is disabled")
	ErrHostedLLMProviderNoKey    = errors.New("hosted llm provider has no API key")
)

// HostedLLMProvider is one SuperPlane-held provider key and model allowlist.
type HostedLLMProvider struct {
	ID            uuid.UUID
	Provider      string
	Enabled       bool
	APIKey        []byte
	BaseURL       string
	AllowedModels datatypes.JSONSlice[string]
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (HostedLLMProvider) TableName() string {
	return "hosted_llm_providers"
}

func (p HostedLLMProvider) HasAPIKey() bool {
	return len(p.APIKey) > 0
}

func (p HostedLLMProvider) HasAllowedModel() bool {
	for _, model := range p.AllowedModels {
		if strings.TrimSpace(model) != "" {
			return true
		}
	}
	return false
}

// OffersHostedModels is true when SuperPlane can resolve hosted credentials.
// A saved API key and allowlist are enough, even if the admin enable switch is off.
func (p HostedLLMProvider) OffersHostedModels() bool {
	if !p.HasAPIKey() {
		return false
	}
	return p.Enabled || p.HasAllowedModel()
}

func (p HostedLLMProvider) AllowsModel(model string) bool {
	normalized := strings.TrimSpace(model)
	if normalized == "" {
		return false
	}
	return slices.Contains(p.AllowedModels, normalized)
}

func ListHostedLLMProviders(tx *gorm.DB) ([]HostedLLMProvider, error) {
	var providers []HostedLLMProvider
	err := tx.Order("provider ASC").Find(&providers).Error
	if err != nil {
		return nil, err
	}
	return providers, nil
}

// HasOfferedHostedLLMProvider is true when SuperPlane can resolve hosted
// credentials and at least one allowlisted model for a coding agent.
func HasOfferedHostedLLMProvider(tx *gorm.DB) (bool, error) {
	providers, err := ListHostedLLMProviders(tx)
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(providers, func(provider HostedLLMProvider) bool {
		return provider.OffersHostedModels() && provider.HasAllowedModel()
	}), nil
}

func FindHostedLLMProvider(tx *gorm.DB, provider string) (*HostedLLMProvider, error) {
	normalized, err := NormalizeHostedLLMProvider(provider)
	if err != nil {
		return nil, err
	}

	var row HostedLLMProvider
	err = tx.Where("provider = ?", normalized).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrHostedLLMProviderNotFound
		}
		return nil, err
	}
	return &row, nil
}

func RequireEnabledHostedLLMProvider(tx *gorm.DB, provider string) (*HostedLLMProvider, error) {
	row, err := FindHostedLLMProvider(tx, provider)
	if err != nil {
		return nil, err
	}
	if !row.OffersHostedModels() {
		if !row.HasAPIKey() {
			return nil, fmt.Errorf("%w: %s", ErrHostedLLMProviderNoKey, provider)
		}
		return nil, fmt.Errorf("%w: %s", ErrHostedLLMProviderDisabled, provider)
	}
	return row, nil
}

func KnownHostedLLMProviders() []string {
	return slices.Clone(hostedLLMProviders)
}

func UpsertHostedLLMProvider(tx *gorm.DB, provider HostedLLMProvider) (*HostedLLMProvider, error) {
	normalized, err := NormalizeHostedLLMProvider(provider.Provider)
	if err != nil {
		return nil, err
	}
	if err := validateAllowedModels(provider.AllowedModels); err != nil {
		return nil, err
	}

	now := time.Now()
	provider.Provider = normalized
	provider.BaseURL = strings.TrimSpace(provider.BaseURL)
	provider.UpdatedAt = now
	if provider.ID == uuid.Nil {
		provider.ID = uuid.New()
	}
	if provider.CreatedAt.IsZero() {
		provider.CreatedAt = now
	}

	err = tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "provider"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"enabled",
			"api_key",
			"base_url",
			"allowed_models",
			"updated_at",
		}),
	}).Create(&provider).Error
	if err != nil {
		return nil, err
	}

	return FindHostedLLMProvider(tx, normalized)
}

func NormalizeHostedLLMProvider(provider string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(provider))
	if !slices.Contains(hostedLLMProviders, normalized) {
		return "", fmt.Errorf("unsupported hosted llm provider: %s", provider)
	}
	return normalized, nil
}

func validateAllowedModels(models datatypes.JSONSlice[string]) error {
	seen := map[string]struct{}{}
	for _, model := range models {
		id := strings.TrimSpace(model)
		if id == "" {
			return errors.New("allowed model id cannot be empty")
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate allowed model: %s", id)
		}
		seen[id] = struct{}{}
	}
	return nil
}
