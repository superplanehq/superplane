package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// LLMModelCapability stores provider model metadata independently of billing rates.
type LLMModelCapability struct {
	Provider             string `gorm:"primaryKey"`
	BaseURL              string `gorm:"primaryKey"`
	Model                string `gorm:"primaryKey"`
	ThinkingLevels       datatypes.JSONSlice[string]
	DefaultThinkingLevel string
	ReasoningMandatory   bool
	RefreshedAt          time.Time
}

func (LLMModelCapability) TableName() string {
	return "llm_model_capabilities"
}

func ListLLMModelCapabilities(tx *gorm.DB, provider string) ([]LLMModelCapability, error) {
	var capabilities []LLMModelCapability
	err := tx.Where("provider = ?", provider).Find(&capabilities).Error
	return capabilities, err
}

// ReplaceLLMModelCapabilities publishes one successfully fetched source catalog.
// The transaction leaves the previous catalog intact if replacement fails.
func ReplaceLLMModelCapabilities(tx *gorm.DB, provider, baseURL string, capabilities []LLMModelCapability) error {
	return tx.Transaction(func(inner *gorm.DB) error {
		// API saves and background refreshes can publish the same source at once.
		if err := inner.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", provider+":"+baseURL).Error; err != nil {
			return err
		}
		if err := inner.Where("provider = ? AND base_url = ?", provider, baseURL).Delete(&LLMModelCapability{}).Error; err != nil {
			return err
		}
		if len(capabilities) == 0 {
			return nil
		}
		for i := range capabilities {
			capabilities[i].Provider = provider
			capabilities[i].BaseURL = baseURL
			if capabilities[i].ThinkingLevels == nil {
				capabilities[i].ThinkingLevels = datatypes.NewJSONSlice([]string{})
			}
		}
		return inner.CreateInBatches(capabilities, 250).Error
	})
}
