package llmcatalog

import (
	"errors"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/llm/modelid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// ThinkingCatalog resolves effort choices from the catalog of the model's source.
type ThinkingCatalog struct {
	openRouterHostedSource string
	openRouter             map[string]map[string][]string
	anthropic              map[string][]string
}

func LoadThinkingCatalog(tx *gorm.DB) (*ThinkingCatalog, error) {
	capabilities, err := models.ListLLMModelCapabilities(tx, models.UsageProviderOpenRouter)
	if err != nil {
		return nil, err
	}
	catalog := &ThinkingCatalog{
		openRouterHostedSource: llm.DefaultOpenRouterBaseURL,
		openRouter:             make(map[string]map[string][]string),
		anthropic:              make(map[string][]string),
	}
	provider, err := models.FindHostedLLMProvider(tx, models.UsageProviderOpenRouter)
	if err != nil && !errors.Is(err, models.ErrHostedLLMProviderNotFound) {
		return nil, err
	}
	if provider != nil {
		catalog.openRouterHostedSource = CatalogBaseURL(provider.BaseURL)
	}
	for _, capability := range capabilities {
		source := CatalogBaseURL(capability.BaseURL)
		if catalog.openRouter[source] == nil {
			catalog.openRouter[source] = make(map[string][]string)
		}
		id := strings.ToLower(modelid.CatalogModelID(capability.Model))
		catalog.openRouter[source][id] = runner.SupportedThinkingLevels(capability.ThinkingLevels)
	}
	anthropic, err := models.ListLLMModelCapabilities(tx, models.UsageProviderAnthropic)
	if err != nil {
		return nil, err
	}
	for _, capability := range anthropic {
		if capability.BaseURL != llm.DefaultAnthropicBaseURL {
			continue
		}
		id := strings.ToLower(modelid.CatalogModelID(capability.Model))
		levels := runner.SupportedThinkingLevels(capability.ThinkingLevels)
		catalog.anthropic[id] = slices.DeleteFunc(levels, func(level string) bool {
			return !slices.Contains(runner.ClaudeCodeThinkingLevels, level)
		})
	}
	return catalog, nil
}

func (c *ThinkingCatalog) ThinkingLevels(provider, model, fundingSource string) []string {
	if provider == models.UsageProviderAnthropic && fundingSource == models.UsageFundingSourceBYOK {
		id := strings.ToLower(modelid.CatalogModelID(model))
		if levels, found := c.anthropic[id]; found {
			return slices.Clone(levels)
		}
	}
	if provider != models.UsageProviderOpenRouter {
		return runner.ModelThinkingLevels(provider, model)
	}
	source := llm.DefaultOpenRouterBaseURL
	if fundingSource == models.UsageFundingSourceHosted {
		source = c.openRouterHostedSource
	}
	id := strings.ToLower(modelid.CatalogModelID(model))
	return c.openRouter[source][id]
}
