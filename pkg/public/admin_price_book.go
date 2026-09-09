package public

import (
	"errors"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage/pricesync"
	"gorm.io/gorm"
)

type priceBookResponse struct {
	Version          string                  `json:"version"`
	PreviousVersion  string                  `json:"previous_version,omitempty"`
	EffectiveAt      *time.Time              `json:"effective_at"`
	ModelRateCount   int                     `json:"model_rate_count"`
	ComputeRateCount int                     `json:"compute_rate_count"`
	Sources          []string                `json:"sources,omitempty"`
	Rates            []priceBookRateResponse `json:"rates"`
}

type priceBookRateResponse struct {
	UsageKind                 string `json:"usage_kind"`
	MatchKey                  string `json:"match_key"`
	MatchMode                 string `json:"match_mode"`
	InputCentsPerMillion      int64  `json:"input_cents_per_million"`
	OutputCentsPerMillion     int64  `json:"output_cents_per_million"`
	CacheReadCentsPerMillion  int64  `json:"cache_read_cents_per_million"`
	CacheWriteCentsPerMillion int64  `json:"cache_write_cents_per_million"`
	ReasoningCentsPerMillion  int64  `json:"reasoning_cents_per_million"`
	MicrosPerSecond           int64  `json:"micros_per_second"`
}

func (s *Server) priceBookScanner() pricesync.Scanner {
	if s.newPriceBookScanner != nil {
		return s.newPriceBookScanner()
	}
	return pricesync.NewScanner(pricesync.Options{HTTP: s.registry.HTTPContext()})
}

func (s *Server) adminGetPriceBook(w http.ResponseWriter, r *http.Request) {
	tx := database.DB(r.Context())
	book, err := models.FindLatestPriceBook(tx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		respondJSON(w, priceBookResponse{Rates: []priceBookRateResponse{}})
		return
	}
	if err != nil {
		log.Errorf("admin: failed to load price book: %v", err)
		http.Error(w, "Failed to load the price book", http.StatusInternalServerError)
		return
	}

	rows, err := models.ListPriceBookRates(tx, book.Version)
	if err != nil {
		log.Errorf("admin: failed to load price book rates: %v", err)
		http.Error(w, "Failed to load the price book", http.StatusInternalServerError)
		return
	}

	respondJSON(w, buildPriceBookResponse(book.Version, "", book.EffectiveAt, nil, rows))
}

func (s *Server) adminScanPriceBook(w http.ResponseWriter, r *http.Request) {
	result, err := pricesync.ScanAndPublish(r.Context(), database.DB(r.Context()), s.priceBookScanner())
	if err != nil {
		log.Errorf("admin: failed to scan price book: %v", err)
		http.Error(w, "SuperPlane could not scan provider prices. Try again.", http.StatusBadGateway)
		return
	}

	rows, err := models.ListPriceBookRates(database.DB(r.Context()), result.Version)
	if err != nil {
		log.Errorf("admin: failed to load scanned price book rates: %v", err)
		http.Error(w, "Failed to load the price book", http.StatusInternalServerError)
		return
	}

	respondJSON(w, buildPriceBookResponse(result.Version, result.PreviousVersion, result.EffectiveAt, result.Sources, rows))
}

func buildPriceBookResponse(
	version, previous string,
	effectiveAt time.Time,
	sources []string,
	rows []models.UsagePriceBookRate,
) priceBookResponse {
	response := priceBookResponse{
		Version:         version,
		PreviousVersion: previous,
		Sources:         sources,
		Rates:           make([]priceBookRateResponse, 0, len(rows)),
	}
	if !effectiveAt.IsZero() {
		response.EffectiveAt = &effectiveAt
	}
	for _, row := range rows {
		if row.UsageKind == models.UsageKindCompute {
			response.ComputeRateCount++
		} else {
			response.ModelRateCount++
		}
		response.Rates = append(response.Rates, priceBookRateResponse{
			UsageKind:                 row.UsageKind,
			MatchKey:                  row.MatchKey,
			MatchMode:                 row.MatchMode,
			InputCentsPerMillion:      row.InputCentsPerMillion,
			OutputCentsPerMillion:     row.OutputCentsPerMillion,
			CacheReadCentsPerMillion:  row.CacheReadCentsPerMillion,
			CacheWriteCentsPerMillion: row.CacheWriteCentsPerMillion,
			ReasoningCentsPerMillion:  row.ReasoningCentsPerMillion,
			MicrosPerSecond:           row.MicrosPerSecond,
		})
	}
	return response
}
