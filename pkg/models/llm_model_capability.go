package models

import (
	"errors"
	"slices"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LLMModelCapability stores provider model metadata independently of billing rates.
type LLMModelCapability struct {
	Provider string `gorm:"primaryKey"`
	BaseURL  string `gorm:"primaryKey"`
	Model    string `gorm:"primaryKey"`
	// Nil means unavailable metadata; an empty list explicitly disables efforts.
	ThinkingLevels       datatypes.JSONSlice[string]
	DefaultThinkingLevel string
	ReasoningMandatory   bool
	RefreshedAt          time.Time
	// Presence flags apply only to incoming metadata, not stored rows.
	DefaultThinkingLevelSet bool `gorm:"-"`
	ReasoningMandatorySet   bool `gorm:"-"`
}

func (LLMModelCapability) TableName() string {
	return "llm_model_capabilities"
}

func ListLLMModelCapabilities(tx *gorm.DB, provider string) ([]LLMModelCapability, error) {
	var capabilities []LLMModelCapability
	err := tx.Where("provider = ?", provider).Find(&capabilities).Error
	return capabilities, err
}

// UpsertLLMModelCapabilities keeps models absent from a restricted API key's
// catalog. Capability metadata is shared; model access remains organization scoped.
func UpsertLLMModelCapabilities(tx *gorm.DB, provider, baseURL string, capabilities []LLMModelCapability) error {
	return publishLLMModelCapabilities(tx, provider, baseURL, capabilities, false)
}

// ReplaceLLMModelCapabilities publishes one successfully fetched source catalog.
// Omitted effort metadata keeps the last successful values for each model.
// The transaction leaves the previous catalog intact if replacement fails.
func ReplaceLLMModelCapabilities(tx *gorm.DB, provider, baseURL string, capabilities []LLMModelCapability) error {
	return publishLLMModelCapabilities(tx, provider, baseURL, capabilities, true)
}

func publishLLMModelCapabilities(tx *gorm.DB, provider, baseURL string, capabilities []LLMModelCapability, replace bool) error {
	if len(capabilities) == 0 {
		return errors.New("model catalog is empty")
	}
	return tx.Transaction(func(inner *gorm.DB) error {
		// API saves and background refreshes can publish the same source at once.
		if err := inner.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", provider+":"+baseURL).Error; err != nil {
			return err
		}
		var previous []LLMModelCapability
		if err := inner.Where("provider = ? AND base_url = ?", provider, baseURL).Find(&previous).Error; err != nil {
			return err
		}
		previousByModel := make(map[string]LLMModelCapability, len(previous))
		for _, capability := range previous {
			previousByModel[capability.Model] = capability
		}
		if replace {
			if err := inner.Where("provider = ? AND base_url = ?", provider, baseURL).Delete(&LLMModelCapability{}).Error; err != nil {
				return err
			}
		}
		for i := range capabilities {
			capabilities[i].Provider = provider
			capabilities[i].BaseURL = baseURL
			if capabilities[i].ThinkingLevels == nil {
				cached := previousByModel[capabilities[i].Model]
				capabilities[i].ThinkingLevels = datatypes.NewJSONSlice(slices.Clone(cached.ThinkingLevels))
				if !capabilities[i].DefaultThinkingLevelSet && capabilities[i].DefaultThinkingLevel == "" {
					capabilities[i].DefaultThinkingLevel = cached.DefaultThinkingLevel
				}
				if !capabilities[i].ReasoningMandatorySet && !capabilities[i].ReasoningMandatory {
					capabilities[i].ReasoningMandatory = cached.ReasoningMandatory
				}
			}
		}
		if replace {
			return inner.CreateInBatches(capabilities, 250).Error
		}
		return inner.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "provider"}, {Name: "base_url"}, {Name: "model"}},
			DoUpdates: clause.AssignmentColumns([]string{"thinking_levels", "default_thinking_level", "reasoning_mandatory", "refreshed_at"}),
		}).CreateInBatches(capabilities, 250).Error
	})
}
