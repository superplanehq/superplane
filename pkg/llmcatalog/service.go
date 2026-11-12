package llmcatalog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/llm/modelid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Service refreshes model capabilities independently of billing price books.
type Service struct {
	encryptor crypto.Encryptor
	http      core.HTTPContext
}

func New(encryptor crypto.Encryptor, httpClient core.HTTPContext) *Service {
	return &Service{encryptor: encryptor, http: httpClient}
}

// CatalogBaseURL identifies the configured source of an OpenRouter model catalog.
func CatalogBaseURL(configured string) string {
	return catalogBaseURL(configured, llm.DefaultOpenRouterBaseURL)
}

func catalogBaseURL(configured, fallback string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(configured), "/")
	if baseURL == "" {
		return fallback
	}
	return baseURL
}

// Refresh stores capabilities after a successful, nonempty fetch.
func (s *Service) Refresh(ctx context.Context, tx *gorm.DB, provider models.HostedLLMProvider) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tx = tx.WithContext(ctx)
	if provider.Provider != models.UsageProviderOpenRouter && provider.Provider != models.UsageProviderAnthropic {
		return fmt.Errorf("model catalog is unavailable for provider %s", provider.Provider)
	}
	baseURL := CatalogBaseURL(provider.BaseURL)
	if provider.Provider == models.UsageProviderAnthropic {
		baseURL = catalogBaseURL(provider.BaseURL, llm.DefaultAnthropicBaseURL)
	}
	if err := llm.ValidateBaseURL(baseURL); err != nil {
		return err
	}
	credentials := llm.Credentials{BaseURL: baseURL}
	if provider.HasAPIKey() {
		apiKey, err := llm.DecryptAPIKey(ctx, s.encryptor, provider.Provider, provider.APIKey)
		if err != nil {
			return err
		}
		credentials.APIKey = apiKey
	}
	if provider.Provider == models.UsageProviderAnthropic {
		return s.refreshAnthropic(ctx, tx, credentials)
	}
	listed, err := llm.ListOpenRouterModels(ctx, s.http, credentials)
	if err != nil {
		return fmt.Errorf("fetch OpenRouter model capabilities: %w", err)
	}
	if len(listed) == 0 {
		return errors.New("OpenRouter model catalog is empty")
	}

	refreshedAt := time.Now()
	capabilities := make([]models.LLMModelCapability, 0, len(listed))
	for _, model := range listed {
		capabilities = append(capabilities, models.LLMModelCapability{
			Provider:                provider.Provider,
			BaseURL:                 baseURL,
			Model:                   strings.ToLower(modelid.CatalogModelID(model.ID)),
			ThinkingLevels:          datatypes.NewJSONSlice(model.ThinkingLevels),
			DefaultThinkingLevel:    model.DefaultThinkingLevel,
			ReasoningMandatory:      model.ReasoningMandatory,
			RefreshedAt:             refreshedAt,
			DefaultThinkingLevelSet: model.DefaultThinkingLevelSet,
			ReasoningMandatorySet:   model.ReasoningMandatorySet,
		})
	}
	if err := models.ReplaceLLMModelCapabilities(tx, provider.Provider, baseURL, capabilities); err != nil {
		return fmt.Errorf("store OpenRouter model capabilities: %w", err)
	}
	return nil
}

// RefreshConfigured refreshes hosted sources and the public OpenRouter catalog
// that workspaces use when they connect their own credentials.
func (s *Service) RefreshConfigured(ctx context.Context, tx *gorm.DB) error {
	return errors.Join(s.refreshConfiguredOpenRouter(ctx, tx), s.refreshConfiguredAnthropic(ctx, tx))
}

func (s *Service) refreshConfiguredAnthropic(ctx context.Context, tx *gorm.DB) error {
	provider, err := models.FindHostedLLMProvider(tx, models.UsageProviderAnthropic)
	if errors.Is(err, models.ErrHostedLLMProviderNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !provider.HasAPIKey() {
		return nil
	}
	return s.Refresh(ctx, tx, *provider)
}

func (s *Service) refreshConfiguredOpenRouter(ctx context.Context, tx *gorm.DB) error {
	provider, err := models.FindHostedLLMProvider(tx, models.UsageProviderOpenRouter)
	if err != nil && !errors.Is(err, models.ErrHostedLLMProviderNotFound) {
		return err
	}

	var refreshErrors []error
	refreshPublic := true
	if provider != nil && provider.HasAPIKey() {
		refreshPublic = CatalogBaseURL(provider.BaseURL) != llm.DefaultOpenRouterBaseURL
		if err := s.Refresh(ctx, tx, *provider); err != nil {
			refreshErrors = append(refreshErrors, fmt.Errorf("refresh saved OpenRouter catalog: %w", err))
			refreshPublic = true
		}
	}
	if refreshPublic {
		publicProvider := models.HostedLLMProvider{Provider: models.UsageProviderOpenRouter}
		if err := s.Refresh(ctx, tx, publicProvider); err != nil {
			refreshErrors = append(refreshErrors, fmt.Errorf("refresh public OpenRouter catalog: %w", err))
		}
	}
	return errors.Join(refreshErrors...)
}
