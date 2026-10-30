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

// CustomLLMAppName is the integration app for an organization custom provider.
const CustomLLMAppName = "customLlm"

var byokIntegrationAppNames = map[string]string{
	UsageProviderAnthropic:  "claude",
	UsageProviderOpenAI:     "openai",
	UsageProviderOpenRouter: "openrouter",
	UsageProviderCustom:     CustomLLMAppName,
}

var ErrModelNotInParentList = errors.New("model is not in the parent selected-model list")
var ErrInheritedModelSelection = errors.New("inherited model policy cannot include selected models")

// OrganizationBYOKModelAllowlist is the org-selected BYOK model list for one provider.
type OrganizationBYOKModelAllowlist struct {
	OrganizationID uuid.UUID `gorm:"primaryKey"`
	Provider       string    `gorm:"primaryKey"`
	AllowedModels  datatypes.JSONSlice[string]
	UpdatedAt      time.Time
}

func (OrganizationBYOKModelAllowlist) TableName() string {
	return "organization_byok_model_allowlists"
}

// FactoryLLMModelAllowlist is a workspace subset of parent models.
// A nil InheritParent preserves legacy behavior: an empty list inherits.
type FactoryLLMModelAllowlist struct {
	FactoryID     uuid.UUID `gorm:"primaryKey"`
	Provider      string    `gorm:"primaryKey"`
	FundingSource string    `gorm:"primaryKey"`
	AllowedModels datatypes.JSONSlice[string]
	InheritParent *bool
	UpdatedAt     time.Time
}

func (FactoryLLMModelAllowlist) TableName() string {
	return "factory_llm_model_allowlists"
}

func (f *FactoryLLMModelAllowlist) InheritsParent() bool {
	if f == nil {
		return true
	}
	if f.InheritParent != nil {
		return *f.InheritParent
	}
	return !hasAllowedModel(f.AllowedModels)
}

func NormalizeBYOKLLMProvider(provider string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(provider))
	if normalized == UsageProviderCustom || slices.Contains(hostedLLMProviders, normalized) {
		return normalized, nil
	}
	return "", fmt.Errorf("unsupported byok llm provider: %s", provider)
}

func BYOKIntegrationAppName(provider string) (string, error) {
	normalized, err := NormalizeBYOKLLMProvider(provider)
	if err != nil {
		return "", err
	}
	appName, ok := byokIntegrationAppNames[normalized]
	if !ok {
		return "", fmt.Errorf("unsupported byok llm provider: %s", provider)
	}
	return appName, nil
}

func FindOrganizationBYOKModelAllowlist(tx *gorm.DB, orgID uuid.UUID, provider string) (*OrganizationBYOKModelAllowlist, error) {
	normalized, err := NormalizeBYOKLLMProvider(provider)
	if err != nil {
		return nil, err
	}

	var row OrganizationBYOKModelAllowlist
	err = tx.Where("organization_id = ? AND provider = ?", orgID, normalized).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &OrganizationBYOKModelAllowlist{
				OrganizationID: orgID,
				Provider:       normalized,
			}, nil
		}
		return nil, err
	}
	return &row, nil
}

// OrganizationBYOKModelAllowlistExists is false until the organization saves
// a model list for the provider. An empty saved list still counts as saved.
func OrganizationBYOKModelAllowlistExists(tx *gorm.DB, orgID uuid.UUID, provider string) (bool, error) {
	normalized, err := NormalizeBYOKLLMProvider(provider)
	if err != nil {
		return false, err
	}

	var count int64
	err = tx.Model(&OrganizationBYOKModelAllowlist{}).
		Where("organization_id = ? AND provider = ?", orgID, normalized).
		Count(&count).
		Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateOrganizationBYOKModelAllowlistIfAbsent inserts the allowlist only when
// the organization has not saved one yet. An existing row, including an empty
// list, stays unchanged.
func CreateOrganizationBYOKModelAllowlistIfAbsent(
	tx *gorm.DB,
	orgID uuid.UUID,
	provider string,
	models datatypes.JSONSlice[string],
) error {
	normalized, err := NormalizeBYOKLLMProvider(provider)
	if err != nil {
		return err
	}
	normalizedModels, err := normalizeAllowedModels(models)
	if err != nil {
		return err
	}
	if len(normalizedModels) == 0 {
		return nil
	}

	row := OrganizationBYOKModelAllowlist{
		OrganizationID: orgID,
		Provider:       normalized,
		AllowedModels:  normalizedModels,
		UpdatedAt:      time.Now(),
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "organization_id"},
			{Name: "provider"},
		},
		DoNothing: true,
	}).Create(&row).Error
}

func UpsertOrganizationBYOKModelAllowlist(tx *gorm.DB, orgID uuid.UUID, provider string, models datatypes.JSONSlice[string]) (*OrganizationBYOKModelAllowlist, error) {
	normalized, err := NormalizeBYOKLLMProvider(provider)
	if err != nil {
		return nil, err
	}
	normalizedModels, err := normalizeAllowedModels(models)
	if err != nil {
		return nil, err
	}

	row := OrganizationBYOKModelAllowlist{
		OrganizationID: orgID,
		Provider:       normalized,
		AllowedModels:  normalizedModels,
		UpdatedAt:      time.Now(),
	}
	err = tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "organization_id"},
			{Name: "provider"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"allowed_models", "updated_at"}),
	}).Create(&row).Error
	if err != nil {
		return nil, err
	}
	return FindOrganizationBYOKModelAllowlist(tx, orgID, normalized)
}

func FindFactoryLLMModelAllowlist(tx *gorm.DB, factoryID uuid.UUID, provider, fundingSource string) (*FactoryLLMModelAllowlist, error) {
	normalized, err := NormalizeBYOKLLMProvider(provider)
	if err != nil {
		return nil, err
	}
	source, err := normalizeLLMFundingSource(fundingSource)
	if err != nil {
		return nil, err
	}
	if source == UsageFundingSourceSubscription && normalized != UsageProviderOpenAI {
		return nil, errors.New("subscription model policies require provider openai")
	}

	var row FactoryLLMModelAllowlist
	err = tx.Where("factory_id = ? AND provider = ? AND funding_source = ?", factoryID, normalized, source).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func UpsertFactoryLLMModelAllowlist(tx *gorm.DB, orgID, factoryID uuid.UUID, provider, fundingSource string, models datatypes.JSONSlice[string], inheritance ...*bool) (*FactoryLLMModelAllowlist, error) {
	if _, err := FindFactory(tx, orgID, factoryID); err != nil {
		return nil, err
	}
	normalized, err := NormalizeBYOKLLMProvider(provider)
	if err != nil {
		return nil, err
	}
	source, err := normalizeLLMFundingSource(fundingSource)
	if err != nil {
		return nil, err
	}
	normalizedModels, err := normalizeAllowedModels(models)
	if err != nil {
		return nil, err
	}
	var inheritParent *bool
	if len(inheritance) > 0 {
		inheritParent = inheritance[0]
	}
	if inheritParent != nil && *inheritParent && len(normalizedModels) > 0 {
		return nil, ErrInheritedModelSelection
	}
	if source == UsageFundingSourceSubscription && normalized != UsageProviderOpenAI {
		return nil, errors.New("subscription model policies require provider openai")
	}
	if source == UsageFundingSourceBYOK {
		if err := RequireOrganizationBYOKAccess(tx, orgID, normalized); err != nil {
			return nil, err
		}
	}
	if source != UsageFundingSourceSubscription {
		// Saved IDs that the parent list no longer offers stay in the policy.
		// Only new IDs must be in the parent list.
		previous, err := FindFactoryLLMModelAllowlist(tx, factoryID, normalized, source)
		if err != nil {
			return nil, err
		}
		var added datatypes.JSONSlice[string]
		for _, model := range normalizedModels {
			if previous == nil || !slices.Contains(previous.AllowedModels, model) {
				added = append(added, model)
			}
		}
		if err := assertFactoryModelsInParent(tx, orgID, normalized, source, added); err != nil {
			return nil, err
		}
	}

	row := FactoryLLMModelAllowlist{
		FactoryID:     factoryID,
		Provider:      normalized,
		FundingSource: source,
		AllowedModels: normalizedModels,
		InheritParent: inheritParent,
		UpdatedAt:     time.Now(),
	}
	err = tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "factory_id"},
			{Name: "provider"},
			{Name: "funding_source"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"allowed_models", "inherit_parent", "updated_at"}),
	}).Create(&row).Error
	if err != nil {
		return nil, err
	}
	return FindFactoryLLMModelAllowlist(tx, factoryID, normalized, source)
}

func assertFactoryModelsInParent(tx *gorm.DB, orgID uuid.UUID, provider, fundingSource string, models datatypes.JSONSlice[string]) error {
	if !hasAllowedModel(models) {
		return nil
	}
	parent, err := parentSelectableLLMModels(tx, orgID, provider, fundingSource)
	if err != nil {
		return err
	}
	allowed := make(map[string]struct{}, len(parent))
	for _, model := range parent {
		allowed[model] = struct{}{}
	}
	for _, model := range models {
		if _, ok := allowed[model]; !ok {
			return fmt.Errorf("%w: %s", ErrModelNotInParentList, model)
		}
	}
	return nil
}

// ResolveSelectableLLMModels returns the models a picker and run gate may use.
// Missing policies inherit. Explicit policies can disable every model.
func ResolveSelectableLLMModels(tx *gorm.DB, orgID uuid.UUID, factoryID *uuid.UUID, provider, fundingSource string) ([]string, error) {
	source, err := normalizeLLMFundingSource(fundingSource)
	if err != nil {
		return nil, err
	}

	parent, err := parentSelectableLLMModels(tx, orgID, provider, source)
	if err != nil {
		return nil, err
	}
	if factoryID == nil || *factoryID == uuid.Nil {
		return parent, nil
	}

	return ApplyFactoryLLMModelPolicy(tx, *factoryID, provider, source, parent)
}

// ApplyFactoryLLMModelPolicy applies workspace policy to the supplied catalog.
// Subscription callers supply the task owner's native catalog.
func ApplyFactoryLLMModelPolicy(tx *gorm.DB, factoryID uuid.UUID, provider, fundingSource string, parent []string) ([]string, error) {
	subset, err := FindFactoryLLMModelAllowlist(tx, factoryID, provider, fundingSource)
	if err != nil {
		return nil, err
	}
	if subset.InheritsParent() {
		return parent, nil
	}
	return IntersectModelIDs(parent, subset.AllowedModels), nil
}

func ModelIsSelectable(tx *gorm.DB, orgID uuid.UUID, factoryID *uuid.UUID, provider, fundingSource, model string) (bool, error) {
	allowed, err := ResolveSelectableLLMModels(tx, orgID, factoryID, provider, fundingSource)
	if err != nil {
		return false, err
	}
	return slices.Contains(allowed, strings.TrimSpace(model)), nil
}

func parentSelectableLLMModels(tx *gorm.DB, orgID uuid.UUID, provider, fundingSource string) ([]string, error) {
	if fundingSource == UsageFundingSourceSubscription {
		return nil, errors.New("subscription models require a personal catalog")
	}
	if fundingSource == UsageFundingSourceBYOK {
		enabled, err := OrganizationBYOKAccess(tx, orgID, provider)
		if err != nil {
			return nil, err
		}
		if !enabled {
			return nil, nil
		}
	}

	if fundingSource == UsageFundingSourceHosted {
		row, err := FindHostedLLMProvider(tx, provider)
		if err != nil {
			if errors.Is(err, ErrHostedLLMProviderNotFound) {
				return nil, nil
			}
			return nil, err
		}
		if !row.OffersHostedModels() {
			return nil, nil
		}
		selection, err := FindOrganizationHostedModelAllowlist(tx, orgID, provider)
		if err != nil {
			return nil, err
		}
		if selection == nil {
			return compactModelIDs(row.AllowedModels), nil
		}
		return IntersectModelIDs(row.AllowedModels, selection.AllowedModels), nil
	}

	row, err := FindOrganizationBYOKModelAllowlist(tx, orgID, provider)
	if err != nil {
		return nil, err
	}
	return compactModelIDs(row.AllowedModels), nil
}

func FindReadyBYOKIntegration(tx *gorm.DB, orgID uuid.UUID, provider string) (*Integration, error) {
	appName, err := BYOKIntegrationAppName(provider)
	if err != nil {
		return nil, err
	}

	var integrations []Integration
	err = tx.Where("organization_id = ? AND app_name = ? AND state = ?", orgID, appName, IntegrationStateReady).
		Order("created_at ASC").
		Find(&integrations).Error
	if err != nil {
		return nil, err
	}
	if len(integrations) == 0 {
		return nil, nil
	}
	return &integrations[0], nil
}

func normalizeLLMFundingSource(source string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(source))
	if normalized == UsageFundingSourceHosted || normalized == UsageFundingSourceBYOK || normalized == UsageFundingSourceSubscription {
		return normalized, nil
	}
	return "", fmt.Errorf("unsupported llm funding source: %s", source)
}

func normalizeAllowedModels(models datatypes.JSONSlice[string]) (datatypes.JSONSlice[string], error) {
	if err := validateAllowedModels(models); err != nil {
		return nil, err
	}
	return compactModelIDs(models), nil
}

func compactModelIDs(models datatypes.JSONSlice[string]) datatypes.JSONSlice[string] {
	return datatypes.JSONSlice[string](CompactModelIDs([]string(models)))
}

func CompactModelIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if _, dup := seen[trimmed]; dup {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func IntersectModelIDs(parent, subset []string) []string {
	allowed := make(map[string]struct{}, len(parent))
	for _, model := range parent {
		allowed[model] = struct{}{}
	}
	out := make([]string, 0, len(subset))
	seen := map[string]struct{}{}
	for _, model := range subset {
		id := strings.TrimSpace(model)
		if id == "" {
			continue
		}
		if _, ok := allowed[id]; !ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func hasAllowedModel(models datatypes.JSONSlice[string]) bool {
	for _, model := range models {
		if strings.TrimSpace(model) != "" {
			return true
		}
	}
	return false
}
