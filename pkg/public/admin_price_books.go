package public

import (
	"errors"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
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

func (s *Server) adminGetPriceBooks(w http.ResponseWriter, r *http.Request) {
	tx := database.DB(r.Context())
	books, err := models.ListUsagePriceBooks(tx)
	if err != nil {
		log.Errorf("admin: failed to list price books: %v", err)
		http.Error(w, "Failed to load price books", http.StatusInternalServerError)
		return
	}

	requestedVersion := strings.TrimSpace(r.URL.Query().Get("version"))
	if len(books) == 0 {
		if requestedVersion != "" {
			http.Error(w, "Price book not found", http.StatusNotFound)
			return
		}

		respondJSON(w, emptyAdminPriceBooksResponse())
		return
	}

	current := books[0]
	selected := current
	if requestedVersion != "" {
		found, err := models.FindUsagePriceBook(tx, requestedVersion)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				http.Error(w, "Price book not found", http.StatusNotFound)
				return
			}

			log.Errorf("admin: failed to load price book %s: %v", requestedVersion, err)
			http.Error(w, "Failed to load price books", http.StatusInternalServerError)
			return
		}
		selected = *found
	}

	rows, err := models.ListUsagePriceBookRates(tx, selected.Version)
	if err != nil {
		log.Errorf("admin: failed to list price book rates for %s: %v", selected.Version, err)
		http.Error(w, "Failed to load price books", http.StatusInternalServerError)
		return
	}

	respondJSON(w, buildAdminPriceBooksResponse(current.Version, selected, books, rows))
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
) adminPriceBooksResponse {
	versions := make([]adminPriceBookVersion, 0, len(books))
	for _, book := range books {
		versions = append(versions, adminPriceBookVersion{
			Version:     book.Version,
			EffectiveAt: book.EffectiveAt.Format(time.RFC3339),
			CreatedAt:   book.CreatedAt.Format(time.RFC3339),
		})
	}

	modelRates := make([]adminPriceBookModelRate, 0)
	vmRates := make([]adminPriceBookVMRate, 0)
	for _, row := range rows {
		switch strings.TrimSpace(row.UsageKind) {
		case models.UsageKindModel:
			modelRates = append(modelRates, adminPriceBookModelRate{
				MatchKey:                  row.MatchKey,
				MatchMode:                 row.MatchMode,
				InputCentsPerMillion:      row.InputCentsPerMillion,
				OutputCentsPerMillion:     row.OutputCentsPerMillion,
				CacheReadCentsPerMillion:  row.CacheReadCentsPerMillion,
				CacheWriteCentsPerMillion: row.CacheWriteCentsPerMillion,
				ReasoningCentsPerMillion:  row.ReasoningCentsPerMillion,
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
