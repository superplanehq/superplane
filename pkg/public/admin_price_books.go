package public

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"gorm.io/gorm"
)

type adminPriceBookVersion struct {
	Version     string `json:"version"`
	EffectiveAt string `json:"effective_at"`
	CreatedAt   string `json:"created_at"`
}

type adminPriceBookModelRate struct {
	Provider                  string `json:"provider"`
	MatchKey                  string `json:"match_key"`
	MatchMode                 string `json:"match_mode"`
	InputCentsPerMillion      int64  `json:"input_cents_per_million"`
	OutputCentsPerMillion     int64  `json:"output_cents_per_million"`
	CacheReadCentsPerMillion  int64  `json:"cache_read_cents_per_million"`
	CacheWriteCentsPerMillion int64  `json:"cache_write_cents_per_million"`
	ReasoningCentsPerMillion  int64  `json:"reasoning_cents_per_million"`
	Selected                  bool   `json:"selected"`
}

type adminPriceBookVMRate struct {
	MatchKey        string `json:"match_key"`
	MatchMode       string `json:"match_mode"`
	MicrosPerSecond int64  `json:"micros_per_second"`
}

type adminPriceBooksResponse struct {
	CurrentVersion string                    `json:"current_version"`
	Version        string                    `json:"version"`
	EffectiveAt    string                    `json:"effective_at"`
	CreatedAt      string                    `json:"created_at"`
	Versions       []adminPriceBookVersion   `json:"versions"`
	Models         []adminPriceBookModelRate `json:"models"`
	VMs            []adminPriceBookVMRate    `json:"vms"`
}

type adminPriceBooksSaveRequest struct {
	BaseVersion string                    `json:"base_version"`
	Models      []adminPriceBookModelRate `json:"models"`
	VMs         []adminPriceBookVMRate    `json:"vms"`
}

type adminPriceBookCurrentRequest struct {
	Version string `json:"version"`
}

type adminPriceBookSyncResponse struct {
	adminPriceBooksResponse
	UpdatedCount     int      `json:"updated_count"`
	AddedCount       int      `json:"added_count"`
	SkippedProviders []string `json:"skipped_providers"`
}

func (s *Server) adminGetPriceBooks(w http.ResponseWriter, r *http.Request) {
	payload, status, message := loadAdminPriceBooks(database.DB(r.Context()), strings.TrimSpace(r.URL.Query().Get("version")))
	if status != http.StatusOK {
		http.Error(w, message, status)
		return
	}
	respondJSON(w, payload)
}

func (s *Server) adminSavePriceBooks(w http.ResponseWriter, r *http.Request) {
	var req adminPriceBooksSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.BaseVersion) == "" {
		http.Error(w, "Base version is required", http.StatusBadRequest)
		return
	}

	rates := make([]models.UsagePriceBookRate, 0, len(req.Models)+len(req.VMs))
	for _, model := range req.Models {
		mode := model.MatchMode
		if strings.TrimSpace(mode) == "" {
			mode = models.UsagePriceBookMatchExact
		}
		rates = append(rates, models.UsagePriceBookRate{
			UsageKind:                 models.UsageKindModel,
			Provider:                  model.Provider,
			MatchKey:                  model.MatchKey,
			MatchMode:                 mode,
			InputCentsPerMillion:      model.InputCentsPerMillion,
			OutputCentsPerMillion:     model.OutputCentsPerMillion,
			CacheReadCentsPerMillion:  model.CacheReadCentsPerMillion,
			CacheWriteCentsPerMillion: model.CacheWriteCentsPerMillion,
			ReasoningCentsPerMillion:  model.ReasoningCentsPerMillion,
		})
	}
	for _, vm := range req.VMs {
		mode := vm.MatchMode
		if strings.TrimSpace(mode) == "" {
			mode = models.UsagePriceBookMatchExact
		}
		rates = append(rates, models.UsagePriceBookRate{
			UsageKind:       models.UsageKindCompute,
			MatchKey:        vm.MatchKey,
			MatchMode:       mode,
			MicrosPerSecond: vm.MicrosPerSecond,
		})
	}

	var published *models.UsagePriceBook
	err := database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		book, pubErr := models.PublishUsagePriceBook(tx, rates, req.BaseVersion)
		if pubErr != nil {
			return pubErr
		}
		published = book
		return nil
	})
	if err != nil {
		writePublishPriceBookError(w, err)
		return
	}
	reloadCurrentPriceBook(r.Context())

	payload, status, message := loadAdminPriceBooks(database.DB(r.Context()), published.Version)
	if status != http.StatusOK {
		log.Errorf("admin: failed to load published price book %s: %s", published.Version, message)
		http.Error(w, "Failed to load price books", http.StatusInternalServerError)
		return
	}
	respondJSON(w, payload)
}

func (s *Server) adminActivatePriceBook(w http.ResponseWriter, r *http.Request) {
	var req adminPriceBookCurrentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Version) == "" {
		http.Error(w, "Version is required", http.StatusBadRequest)
		return
	}

	err := database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		return models.ActivateUsagePriceBook(tx, req.Version)
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Price book not found", http.StatusNotFound)
			return
		}
		log.Errorf("admin: failed to activate price book %s: %v", req.Version, err)
		http.Error(w, "Failed to activate price book", http.StatusInternalServerError)
		return
	}
	reloadCurrentPriceBook(r.Context())

	payload, status, message := loadAdminPriceBooks(database.DB(r.Context()), strings.TrimSpace(req.Version))
	if status != http.StatusOK {
		http.Error(w, message, status)
		return
	}
	respondJSON(w, payload)
}

func (s *Server) adminDeletePriceBook(w http.ResponseWriter, r *http.Request) {
	version := strings.TrimSpace(r.URL.Query().Get("version"))
	if version == "" {
		http.Error(w, "Version is required", http.StatusBadRequest)
		return
	}

	err := database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		return models.DeleteUsagePriceBook(tx, version)
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Price book not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, models.ErrUsagePriceBookCurrent) {
			http.Error(w, "You cannot delete the current price book. Switch to another version first.", http.StatusConflict)
			return
		}
		if errors.Is(err, models.ErrUsagePriceBookLast) {
			http.Error(w, "You cannot delete the last price book.", http.StatusConflict)
			return
		}
		log.Errorf("admin: failed to delete price book %s: %v", version, err)
		http.Error(w, "Failed to delete price book", http.StatusInternalServerError)
		return
	}

	payload, status, message := loadAdminPriceBooks(database.DB(r.Context()), "")
	if status != http.StatusOK {
		http.Error(w, message, status)
		return
	}
	respondJSON(w, payload)
}

func (s *Server) adminSyncPriceBooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	provider := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("provider")))
	if provider == "" {
		provider = models.UsageProviderOpenRouter
	}
	if provider != models.UsageProviderOpenRouter {
		http.Error(w, catalogPricesUnavailableMessage(provider), http.StatusBadRequest)
		return
	}
	sync, err := s.collectCatalogRates(ctx, database.DB(ctx), provider)
	if err != nil {
		if errors.Is(err, errNoPricedCatalogProvider) {
			http.Error(w, "No enabled provider publishes catalog prices", http.StatusBadRequest)
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Price book not found", http.StatusNotFound)
			return
		}
		log.Errorf("admin: failed to read provider catalog prices: %v", err)
		http.Error(w, "Unable to update model rates from the provider", http.StatusBadGateway)
		return
	}

	var published *models.UsagePriceBook
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		book, pubErr := models.PublishUsagePriceBook(tx, sync.rates, sync.baseVersion)
		if pubErr != nil {
			return pubErr
		}
		published = book
		return nil
	})
	if err != nil {
		if errors.Is(err, models.ErrUsagePriceBookConflict) {
			writePublishPriceBookError(w, err)
			return
		}
		log.Errorf("admin: failed to publish synced price book: %v", err)
		http.Error(w, "Failed to save price books", http.StatusInternalServerError)
		return
	}
	reloadCurrentPriceBook(ctx)

	payload, status, message := loadAdminPriceBooks(database.DB(ctx), published.Version)
	if status != http.StatusOK {
		log.Errorf("admin: failed to load synced price book %s: %s", published.Version, message)
		http.Error(w, "Failed to load price books", http.StatusInternalServerError)
		return
	}
	respondJSON(w, adminPriceBookSyncResponse{
		adminPriceBooksResponse: payload,
		UpdatedCount:            sync.updated,
		AddedCount:              sync.added,
		SkippedProviders:        sync.skipped,
	})
}

// reloadCurrentPriceBook installs the committed catalog into the in-memory
// book. The process keeps the previous book when the reload fails.
func reloadCurrentPriceBook(ctx context.Context) {
	if err := models.LoadCurrentPriceBook(database.DB(ctx)); err != nil {
		log.Errorf("admin: failed to reload the current price book: %v", err)
	}
}

var errNoPricedCatalogProvider = errors.New("no enabled provider publishes catalog prices")

const adminPriceBookConflictMessage = "The current price book changed. Load the latest version and try again."

func catalogPricesUnavailableMessage(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case models.UsageProviderAnthropic:
		return "The Anthropic API does not publish prices."
	case models.UsageProviderOpenAI:
		return "The OpenAI API does not publish prices."
	default:
		return "This provider does not publish catalog prices."
	}
}

func writePublishPriceBookError(w http.ResponseWriter, err error) {
	if errors.Is(err, models.ErrUsagePriceBookConflict) {
		http.Error(w, adminPriceBookConflictMessage, http.StatusConflict)
		return
	}
	http.Error(w, err.Error(), http.StatusBadRequest)
}

// catalogSync is the price book that provider catalogs produce, before it is published.
type catalogSync struct {
	rates       []models.UsagePriceBookRate
	baseVersion string
	updated     int
	added       int
	skipped     []string
}

// collectCatalogRates reads the current rates and merges provider catalog
// prices into them. It runs outside a transaction because it calls provider
// HTTP APIs.
func (s *Server) collectCatalogRates(ctx context.Context, tx *gorm.DB, targetProvider string) (catalogSync, error) {
	current, err := models.FindCurrentUsagePriceBook(tx)
	if err != nil {
		return catalogSync{}, err
	}
	rows, err := models.ListUsagePriceBookRates(tx, current.Version)
	if err != nil {
		return catalogSync{}, err
	}
	providers, err := models.ListHostedLLMProviders(tx)
	if err != nil {
		return catalogSync{}, err
	}

	sync := catalogSync{
		rates:       models.CloneUsagePriceBookRates(rows),
		baseVersion: current.Version,
		skipped:     make([]string, 0),
	}
	for _, provider := range providers {
		if provider.Provider != targetProvider {
			continue
		}
		if !provider.Enabled || !provider.HasAPIKey() {
			return catalogSync{}, errNoPricedCatalogProvider
		}

		apiKey, decryptErr := llm.DecryptAPIKey(ctx, s.encryptor, provider.Provider, provider.APIKey)
		if decryptErr != nil {
			return catalogSync{}, decryptErr
		}
		prices, listErr := llm.ListCatalogPrices(
			ctx,
			s.registry.HTTPContext(),
			provider.Provider,
			llm.Credentials{APIKey: apiKey, BaseURL: provider.BaseURL},
		)
		if listErr != nil {
			if errors.Is(listErr, llm.ErrNoCatalogPrices) {
				return catalogSync{}, errNoPricedCatalogProvider
			}
			return catalogSync{}, listErr
		}

		var updated, added int
		sync.rates, updated, added = models.ApplyCatalogPrices(sync.rates, catalogModelPrices(provider.Provider, prices))
		sync.updated += updated
		sync.added += added
		return sync, nil
	}

	return catalogSync{}, errNoPricedCatalogProvider
}

func catalogModelPrices(provider string, prices []llm.CatalogPrice) []models.CatalogModelPrice {
	mapped := make([]models.CatalogModelPrice, 0, len(prices))
	for _, price := range prices {
		id := pricebook.CatalogModelID(price.ID)
		if id == "" {
			continue
		}
		mapped = append(mapped, models.CatalogModelPrice{
			Provider: provider,
			ModelID:  id,
			Rate:     price.Rate,
		})
	}
	return mapped
}

func loadAdminPriceBooks(tx *gorm.DB, requestedVersion string) (adminPriceBooksResponse, int, string) {
	books, err := models.ListUsagePriceBooks(tx)
	if err != nil {
		log.Errorf("admin: failed to list price books: %v", err)
		return adminPriceBooksResponse{}, http.StatusInternalServerError, "Failed to load price books"
	}

	if len(books) == 0 {
		if requestedVersion != "" {
			return adminPriceBooksResponse{}, http.StatusNotFound, "Price book not found"
		}
		return emptyAdminPriceBooksResponse(), http.StatusOK, ""
	}

	current, err := models.FindCurrentUsagePriceBook(tx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return adminPriceBooksResponse{}, http.StatusNotFound, "Price book not found"
		}
		log.Errorf("admin: failed to load current price book: %v", err)
		return adminPriceBooksResponse{}, http.StatusInternalServerError, "Failed to load price books"
	}

	selected := *current
	if requestedVersion != "" {
		found, findErr := models.FindUsagePriceBook(tx, requestedVersion)
		if findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return adminPriceBooksResponse{}, http.StatusNotFound, "Price book not found"
			}
			log.Errorf("admin: failed to load price book %s: %v", requestedVersion, findErr)
			return adminPriceBooksResponse{}, http.StatusInternalServerError, "Failed to load price books"
		}
		selected = *found
	}

	rows, err := models.ListUsagePriceBookRates(tx, selected.Version)
	if err != nil {
		log.Errorf("admin: failed to list price book rates for %s: %v", selected.Version, err)
		return adminPriceBooksResponse{}, http.StatusInternalServerError, "Failed to load price books"
	}

	providers, err := models.ListHostedLLMProviders(tx)
	if err != nil {
		log.Errorf("admin: failed to list providers: %v", err)
		return adminPriceBooksResponse{}, http.StatusInternalServerError, "Failed to load providers"
	}

	return buildAdminPriceBooksResponse(current.Version, selected, books, rows, providers), http.StatusOK, ""
}

func emptyAdminPriceBooksResponse() adminPriceBooksResponse {
	return adminPriceBooksResponse{
		Versions: []adminPriceBookVersion{},
		Models:   []adminPriceBookModelRate{},
		VMs:      []adminPriceBookVMRate{},
	}
}

func buildAdminPriceBooksResponse(
	currentVersion string,
	selected models.UsagePriceBook,
	books []models.UsagePriceBook,
	rows []models.UsagePriceBookRate,
	providers []models.HostedLLMProvider,
) adminPriceBooksResponse {
	versions := make([]adminPriceBookVersion, 0, len(books))
	for _, book := range books {
		versions = append(versions, adminPriceBookVersion{
			Version:     book.Version,
			EffectiveAt: book.EffectiveAt.Format(time.RFC3339),
			CreatedAt:   book.CreatedAt.Format(time.RFC3339),
		})
	}

	allowlist := hostedModelAllowlist(providers)

	modelRates := make([]adminPriceBookModelRate, 0)
	vmRates := make([]adminPriceBookVMRate, 0)
	for _, row := range rows {
		switch strings.TrimSpace(row.UsageKind) {
		case models.UsageKindModel:
			modelRates = append(modelRates, adminPriceBookModelRate{
				Provider:                  row.Provider,
				MatchKey:                  row.MatchKey,
				MatchMode:                 row.MatchMode,
				InputCentsPerMillion:      row.InputCentsPerMillion,
				OutputCentsPerMillion:     row.OutputCentsPerMillion,
				CacheReadCentsPerMillion:  row.CacheReadCentsPerMillion,
				CacheWriteCentsPerMillion: row.CacheWriteCentsPerMillion,
				ReasoningCentsPerMillion:  row.ReasoningCentsPerMillion,
				Selected:                  allowlist[hostedAllowlistKey{provider: row.Provider, model: row.MatchKey}],
			})
		case models.UsageKindCompute:
			vmRates = append(vmRates, adminPriceBookVMRate{
				MatchKey:        row.MatchKey,
				MatchMode:       row.MatchMode,
				MicrosPerSecond: row.MicrosPerSecond,
			})
		}
	}

	if selected.Version == currentVersion {
		modelRates = appendAllowlistedModelRates(modelRates, allowlist)
	}

	return adminPriceBooksResponse{
		CurrentVersion: currentVersion,
		Version:        selected.Version,
		EffectiveAt:    selected.EffectiveAt.Format(time.RFC3339),
		CreatedAt:      selected.CreatedAt.Format(time.RFC3339),
		Versions:       versions,
		Models:         modelRates,
		VMs:            vmRates,
	}
}

type hostedAllowlistKey struct {
	provider string
	model    string
}

func hostedModelAllowlist(providers []models.HostedLLMProvider) map[hostedAllowlistKey]bool {
	allowlist := map[hostedAllowlistKey]bool{}
	for _, provider := range providers {
		if !provider.OffersHostedModels() {
			continue
		}
		for _, model := range provider.AllowedModels {
			key := strings.ToLower(strings.TrimSpace(pricebook.CatalogModelID(model)))
			if key == "" {
				continue
			}
			allowlist[hostedAllowlistKey{provider: provider.Provider, model: key}] = true
		}
	}
	return allowlist
}

func appendAllowlistedModelRates(modelRates []adminPriceBookModelRate, allowlist map[hostedAllowlistKey]bool) []adminPriceBookModelRate {
	present := map[hostedAllowlistKey]struct{}{}
	for _, rate := range modelRates {
		present[hostedAllowlistKey{provider: rate.Provider, model: rate.MatchKey}] = struct{}{}
	}
	for key := range allowlist {
		if _, ok := present[key]; ok {
			continue
		}
		rate, _ := pricebook.Lookup(key.provider, key.model)
		modelRates = append(modelRates, adminPriceBookModelRate{
			Provider:                  key.provider,
			MatchKey:                  key.model,
			MatchMode:                 models.UsagePriceBookMatchExact,
			InputCentsPerMillion:      rate.Input,
			OutputCentsPerMillion:     rate.Output,
			CacheReadCentsPerMillion:  rate.CacheRead,
			CacheWriteCentsPerMillion: rate.CacheWrite,
			ReasoningCentsPerMillion:  rate.Reasoning,
			Selected:                  true,
		})
	}
	slices.SortFunc(modelRates, compareAdminModelRates)
	return modelRates
}

func compareAdminModelRates(a, b adminPriceBookModelRate) int {
	if order := strings.Compare(a.Provider, b.Provider); order != 0 {
		return order
	}
	if order := strings.Compare(a.MatchKey, b.MatchKey); order != 0 {
		return order
	}
	return strings.Compare(a.MatchMode, b.MatchMode)
}
