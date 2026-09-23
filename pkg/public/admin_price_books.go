package public

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"github.com/superplanehq/superplane/pkg/usage/pricebooksync"
	"gorm.io/gorm"
)

type adminPriceBookVersion struct {
	Version     string `json:"version"`
	EffectiveAt string `json:"effective_at"`
	CreatedAt   string `json:"created_at"`
}

type adminPriceBookModelRate struct {
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
		rates = append(rates, models.UsagePriceBookRate{
			UsageKind:                 models.UsageKindModel,
			MatchKey:                  model.MatchKey,
			MatchMode:                 model.MatchMode,
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

func (s *Server) adminSyncPriceBooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	svc := pricebooksync.New(s.encryptor, s.registry.HTTPContext())
	result, err := svc.Sync(ctx, database.DB(ctx), pricebooksync.Options{SkipWhenUnchanged: false})
	if err != nil {
		if errors.Is(err, pricebooksync.ErrNoPricedCatalogProvider) {
			http.Error(w, "No enabled provider publishes catalog prices", http.StatusBadRequest)
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Price book not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, models.ErrUsagePriceBookConflict) {
			writePublishPriceBookError(w, err)
			return
		}
		log.Errorf("admin: failed to sync price book from provider catalog: %v", err)
		http.Error(w, "Unable to update model rates from the provider", http.StatusBadGateway)
		return
	}

	payload, status, message := loadAdminPriceBooks(database.DB(ctx), result.Book.Version)
	if status != http.StatusOK {
		log.Errorf("admin: failed to load synced price book %s: %s", result.Book.Version, message)
		http.Error(w, "Failed to load price books", http.StatusInternalServerError)
		return
	}
	respondJSON(w, adminPriceBookSyncResponse{
		adminPriceBooksResponse: payload,
		UpdatedCount:            result.UpdatedCount,
		AddedCount:              result.AddedCount,
		SkippedProviders:        result.SkippedProviders,
	})
}

// reloadCurrentPriceBook installs the committed catalog into the in-memory
// book. The process keeps the previous book when the reload fails.
func reloadCurrentPriceBook(ctx context.Context) {
	if err := models.LoadCurrentPriceBook(database.DB(ctx)); err != nil {
		log.Errorf("admin: failed to reload the current price book: %v", err)
	}
}

const adminPriceBookConflictMessage = "The current price book changed. Load the latest version and try again."

func writePublishPriceBookError(w http.ResponseWriter, err error) {
	if errors.Is(err, models.ErrUsagePriceBookConflict) {
		http.Error(w, adminPriceBookConflictMessage, http.StatusConflict)
		return
	}
	http.Error(w, err.Error(), http.StatusBadRequest)
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

	prefixes := make([]pricebook.PrefixRate, 0)
	families := make([]pricebook.FamilyRate, 0)
	for _, row := range rows {
		if row.UsageKind != models.UsageKindModel {
			continue
		}
		rate := pricebook.Rate{
			Input:      row.InputCentsPerMillion,
			Output:     row.OutputCentsPerMillion,
			CacheRead:  row.CacheReadCentsPerMillion,
			CacheWrite: row.CacheWriteCentsPerMillion,
			Reasoning:  row.ReasoningCentsPerMillion,
		}
		switch row.MatchMode {
		case models.UsagePriceBookMatchPrefix:
			prefixes = append(prefixes, pricebook.PrefixRate{Prefix: row.MatchKey, Rate: rate})
		case models.UsagePriceBookMatchFamily:
			families = append(families, pricebook.FamilyRate{Token: row.MatchKey, Rate: rate})
		}
	}
	selectedKeys := make(map[string]bool)
	for _, model := range allowlist {
		match, ok := pricebook.MatchModel(model, prefixes, families)
		if ok {
			selectedKeys[match.Key+"\x00"+match.Mode] = true
		}
	}

	modelRates := make([]adminPriceBookModelRate, 0)
	vmRates := make([]adminPriceBookVMRate, 0)
	for _, row := range rows {
		switch strings.TrimSpace(row.UsageKind) {
		case models.UsageKindModel:
			key := row.MatchKey + "\x00" + row.MatchMode
			modelRates = append(modelRates, adminPriceBookModelRate{
				MatchKey:                  row.MatchKey,
				MatchMode:                 row.MatchMode,
				InputCentsPerMillion:      row.InputCentsPerMillion,
				OutputCentsPerMillion:     row.OutputCentsPerMillion,
				CacheReadCentsPerMillion:  row.CacheReadCentsPerMillion,
				CacheWriteCentsPerMillion: row.CacheWriteCentsPerMillion,
				ReasoningCentsPerMillion:  row.ReasoningCentsPerMillion,
				Selected:                  selectedKeys[key],
			})
		case models.UsageKindCompute:
			vmRates = append(vmRates, adminPriceBookVMRate{
				MatchKey:        row.MatchKey,
				MatchMode:       row.MatchMode,
				MicrosPerSecond: row.MicrosPerSecond,
			})
		}
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

func hostedModelAllowlist(providers []models.HostedLLMProvider) []string {
	allowlist := make([]string, 0)
	for _, provider := range providers {
		if !provider.OffersHostedModels() {
			continue
		}
		for _, model := range provider.AllowedModels {
			if strings.TrimSpace(model) != "" {
				allowlist = append(allowlist, model)
			}
		}
	}
	return allowlist
}
