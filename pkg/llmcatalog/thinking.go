package llmcatalog

import (
	"errors"
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
}

func LoadThinkingCatalog(tx *gorm.DB) (*ThinkingCatalog, error) {
	capabilities, err := models.ListLLMModelCapabilities(tx, models.UsageProviderOpenRouter)
	if err != nil {
		return nil, err
	}
	catalog := &ThinkingCatalog{
		openRouterHostedSource: llm.DefaultOpenRouterBaseURL,
		openRouter:             make(map[string]map[string][]string),
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
	return catalog, nil
}

func (c *ThinkingCatalog) ThinkingLevels(provider, model, fundingSource string) []string {
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
