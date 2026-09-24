package pricebooksync

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"gorm.io/gorm"
)

// ErrNoPricedCatalogProvider means no enabled provider publishes catalog prices.
var ErrNoPricedCatalogProvider = errors.New("no enabled provider publishes catalog prices")

// Service fetches provider catalog prices and publishes price book versions.
type Service struct {
	encryptor crypto.Encryptor
	http      core.HTTPContext
}

// New builds a catalog sync service.
func New(encryptor crypto.Encryptor, http core.HTTPContext) *Service {
	return &Service{
		encryptor: encryptor,
		http:      http,
	}
}

// Options controls publish behavior.
type Options struct {
	// SkipWhenUnchanged avoids a new version when merged rates match the current book.
	SkipWhenUnchanged bool
}

// Result is the outcome of one catalog sync.
type Result struct {
	Published        bool
	Book             *models.UsagePriceBook
	UpdatedCount     int
	AddedCount       int
	SkippedProviders []string
}

type collectResult struct {
	before      []models.UsagePriceBookRate
	rates       []models.UsagePriceBookRate
	baseVersion string
	updated     int
	added       int
	skipped     []string
}

// Sync reads provider catalog prices and optionally publishes a new price book.
func (s *Service) Sync(ctx context.Context, tx *gorm.DB, opts Options) (Result, error) {
	collected, err := s.collect(ctx, tx)
	if err != nil {
		return Result{}, err
	}

	out := Result{
		UpdatedCount:     collected.updated,
		AddedCount:       collected.added,
		SkippedProviders: collected.skipped,
	}

	if opts.SkipWhenUnchanged && models.UsagePriceBookRatesEqual(collected.before, collected.rates) {
		current, findErr := models.FindCurrentUsagePriceBook(tx)
		if findErr != nil {
			return Result{}, findErr
		}
		out.Book = current
		return out, nil
	}

	var published *models.UsagePriceBook
	err = tx.Transaction(func(inner *gorm.DB) error {
		book, pubErr := models.PublishUsagePriceBook(inner, collected.rates, collected.baseVersion)
		if pubErr != nil {
			return pubErr
		}
		published = book
		return nil
	})
	if err != nil {
		return Result{}, err
	}

	if err := models.LoadCurrentPriceBook(tx); err != nil {
		log.Errorf("pricebooksync: published version %s but failed to reload in-memory book: %v", published.Version, err)
	}

	out.Published = true
	out.Book = published
	return out, nil
}

func (s *Service) collect(ctx context.Context, tx *gorm.DB) (collectResult, error) {
	current, err := models.FindCurrentUsagePriceBook(tx)
	if err != nil {
		return collectResult{}, err
	}
	rows, err := models.ListUsagePriceBookRates(tx, current.Version)
	if err != nil {
		return collectResult{}, err
	}
	providers, err := models.ListHostedLLMProviders(tx)
	if err != nil {
		return collectResult{}, err
	}

	before := models.CloneUsagePriceBookRates(rows)
	out := collectResult{
		before:      before,
		rates:       models.CloneUsagePriceBookRates(rows),
		baseVersion: current.Version,
		skipped:     make([]string, 0),
	}
	fetchedPricedProvider := false
	for _, provider := range providers {
		if !provider.Enabled || !provider.HasAPIKey() {
			continue
		}
		if provider.Provider != models.UsageProviderOpenRouter {
			out.skipped = append(out.skipped, provider.Provider)
			continue
		}

		apiKey, decryptErr := llm.DecryptAPIKey(ctx, s.encryptor, provider.Provider, provider.APIKey)
		if decryptErr != nil {
			return collectResult{}, decryptErr
		}
		prices, listErr := llm.ListCatalogPrices(
			ctx,
			s.http,
			provider.Provider,
			llm.Credentials{APIKey: apiKey, BaseURL: provider.BaseURL},
		)
		if listErr != nil {
			if errors.Is(listErr, llm.ErrNoCatalogPrices) {
				out.skipped = append(out.skipped, provider.Provider)
				continue
			}
			return collectResult{}, listErr
		}

		fetchedPricedProvider = true
		var updated, added int
		out.rates, updated, added = models.ApplyCatalogPrices(out.rates, FilterCatalogPrices(provider.Provider, prices, provider.AllowedModels))
		out.updated += updated
		out.added += added
	}

	if !fetchedPricedProvider {
		return collectResult{}, ErrNoPricedCatalogProvider
	}
	return out, nil
}

// FilterCatalogPrices keeps catalog prices that match a provider allowlist.
func FilterCatalogPrices(providerName string, prices []llm.CatalogPrice, allowlist []string) []models.CatalogModelPrice {
	provider := models.HostedLLMProvider{AllowedModels: allowlist}
	filtered := make([]models.CatalogModelPrice, 0)
	for _, price := range prices {
		if !catalogPriceAllowed(provider, price.ID) {
			continue
		}
		id := pricebook.CatalogModelID(price.ID)
		if id == "" {
			continue
		}
		filtered = append(filtered, models.CatalogModelPrice{
			Provider: providerName,
			ModelID:  id,
			Rate:     price.Rate,
		})
	}
	return filtered
}

func catalogPriceAllowed(provider models.HostedLLMProvider, id string) bool {
	if provider.AllowsModel(id) {
		return true
	}
	return provider.AllowsModel(pricebook.NormalizeModelID(id))
}
