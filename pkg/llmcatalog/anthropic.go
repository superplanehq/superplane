package llmcatalog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/llm/modelid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *Service) RefreshAnthropicIntegration(ctx context.Context, tx *gorm.DB, integration core.IntegrationContext) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tx = tx.WithContext(ctx)
	apiKey, err := integration.GetConfig("apiKey")
	if err != nil {
		return fmt.Errorf("read Claude integration API key: %w", err)
	}
	return s.refreshAnthropic(ctx, tx, llm.Credentials{APIKey: string(apiKey)})
}

func (s *Service) refreshAnthropic(ctx context.Context, tx *gorm.DB, credentials llm.Credentials) error {
	client, err := llm.New(s.http, llm.ProviderAnthropic, credentials)
	if err != nil {
		return err
	}
	listed, err := client.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("fetch Anthropic model capabilities: %w", err)
	}
	if len(listed) == 0 {
		return errors.New("Anthropic model catalog is empty")
	}
	return StoreAnthropicCapabilities(tx, credentials.BaseURL, listed)
}

// StoreAnthropicCapabilities preserves the last successful metadata when a
// response omits capabilities, scoped to the fetched source. An explicit
// empty list disables effort choices.
func StoreAnthropicCapabilities(tx *gorm.DB, baseURL string, listed []llm.Model) error {
	baseURL = catalogBaseURL(baseURL, llm.DefaultAnthropicBaseURL)
	refreshedAt := time.Now()
	capabilities := make([]models.LLMModelCapability, 0, len(listed))
	seen := map[string]bool{}
	for _, model := range listed {
		id := strings.ToLower(modelid.CatalogModelID(model.ID))
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		capabilities = append(capabilities, models.LLMModelCapability{
			Provider:             models.UsageProviderAnthropic,
			BaseURL:              baseURL,
			Model:                id,
			ThinkingLevels:       datatypes.NewJSONSlice(model.ThinkingLevels),
			DefaultThinkingLevel: model.DefaultThinkingLevel,
			ReasoningMandatory:   model.ReasoningMandatory,
			RefreshedAt:          refreshedAt,
		})
	}
	if err := models.UpsertLLMModelCapabilities(tx, models.UsageProviderAnthropic, baseURL, capabilities); err != nil {
		return fmt.Errorf("store Anthropic model capabilities: %w", err)
	}
	return nil
}
