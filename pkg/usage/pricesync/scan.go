package pricesync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"gorm.io/gorm"
)

// ModelRate is one scanned LLM match rule.
type ModelRate struct {
	MatchKey  string
	MatchMode string
	Rate      pricebook.Rate
}

// ComputeRate is one scanned runner VM rate.
type ComputeRate struct {
	MachineType     string
	MicrosPerSecond int64
}

// Snapshot is the merged catalog a scan is ready to publish.
type Snapshot struct {
	ModelRates   []ModelRate
	ComputeRates []ComputeRate
	Sources      []string
}

// Result is the published catalog version after a scan.
type Result struct {
	Version          string
	PreviousVersion  string
	ModelRateCount   int
	ComputeRateCount int
	Sources          []string
	EffectiveAt      time.Time
}

// ModelSource fetches LLM rates from one vendor or catalog.
type ModelSource interface {
	Name() string
	ScanModels(ctx context.Context) ([]ModelRate, error)
}

// ComputeSource fetches SuperPlane runner VM rates.
type ComputeSource interface {
	Name() string
	ScanCompute(ctx context.Context) ([]ComputeRate, error)
}

// Options configure a scan. HTTP is required for OpenRouter.
type Options struct {
	HTTP          core.HTTPContext
	OpenRouterURL string
	Now           func() time.Time
	Models        []ModelSource
	Compute       ComputeSource
}

// Scanner merges vendor and catalog sources, then publishes a new version.
type Scanner struct {
	models  []ModelSource
	compute ComputeSource
	now     func() time.Time
}

// NewScanner builds the default OpenRouter + SuperPlane catalog scanner.
func NewScanner(opts Options) Scanner {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	models := opts.Models
	if models == nil {
		models = []ModelSource{
			CatalogSource{},
			OpenRouterSource{HTTP: opts.HTTP, URL: opts.OpenRouterURL},
		}
	}
	compute := opts.Compute
	if compute == nil {
		compute = CatalogSource{}
	}
	return Scanner{models: models, compute: compute, now: now}
}

// Scan fetches every source and merges exact vendor rates over catalog fallbacks.
func (s Scanner) Scan(ctx context.Context) (Snapshot, error) {
	if s.compute == nil {
		return Snapshot{}, fmt.Errorf("compute source is required")
	}
	if len(s.models) == 0 {
		return Snapshot{}, fmt.Errorf("model source is required")
	}

	snapshot := Snapshot{}
	merged := map[string]ModelRate{}
	for _, source := range s.models {
		rates, err := source.ScanModels(ctx)
		if err != nil {
			return Snapshot{}, fmt.Errorf("scan models from %s: %w", source.Name(), err)
		}
		snapshot.Sources = appendUnique(snapshot.Sources, source.Name())
		for _, rate := range rates {
			key := rate.MatchMode + ":" + rate.MatchKey
			merged[key] = rate
		}
	}

	snapshot.ModelRates = make([]ModelRate, 0, len(merged))
	for _, rate := range merged {
		snapshot.ModelRates = append(snapshot.ModelRates, rate)
	}

	computeRates, err := s.compute.ScanCompute(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("scan compute from %s: %w", s.compute.Name(), err)
	}
	if len(computeRates) == 0 {
		return Snapshot{}, fmt.Errorf("compute catalog returned no VM rates")
	}
	snapshot.ComputeRates = computeRates
	snapshot.Sources = appendUnique(snapshot.Sources, s.compute.Name())
	return snapshot, nil
}

// ScanAndPublish writes a new price book version and loads it in memory.
func ScanAndPublish(ctx context.Context, tx *gorm.DB, scanner Scanner) (*Result, error) {
	snapshot, err := scanner.Scan(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if scanner.now != nil {
		now = scanner.now().UTC()
	}

	var result *Result
	err = tx.WithContext(ctx).Transaction(func(inner *gorm.DB) error {
		previous := ""
		current, findErr := models.FindLatestPriceBook(inner)
		if findErr == nil {
			previous = current.Version
		} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		version, versionErr := models.NextPriceBookVersion(inner, now)
		if versionErr != nil {
			return versionErr
		}

		insertErr := models.InsertPriceBook(inner, models.UsagePriceBook{
			Version:     version,
			EffectiveAt: now,
		}, snapshotRates(snapshot))
		if insertErr != nil {
			return insertErr
		}
		if loadErr := models.LoadCurrentPriceBook(inner); loadErr != nil {
			return loadErr
		}

		result = &Result{
			Version:          version,
			PreviousVersion:  previous,
			ModelRateCount:   len(snapshot.ModelRates),
			ComputeRateCount: len(snapshot.ComputeRates),
			Sources:          snapshot.Sources,
			EffectiveAt:      now,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func snapshotRates(snapshot Snapshot) []models.UsagePriceBookRate {
	rows := make([]models.UsagePriceBookRate, 0, len(snapshot.ModelRates)+len(snapshot.ComputeRates))
	for _, rate := range snapshot.ModelRates {
		rows = append(rows, models.UsagePriceBookRate{
			ID:                        uuid.New(),
			UsageKind:                 models.UsageKindModel,
			MatchKey:                  rate.MatchKey,
			MatchMode:                 rate.MatchMode,
			InputCentsPerMillion:      rate.Rate.Input,
			OutputCentsPerMillion:     rate.Rate.Output,
			CacheReadCentsPerMillion:  rate.Rate.CacheRead,
			CacheWriteCentsPerMillion: rate.Rate.CacheWrite,
			ReasoningCentsPerMillion:  rate.Rate.Reasoning,
		})
	}
	for _, rate := range snapshot.ComputeRates {
		rows = append(rows, models.UsagePriceBookRate{
			ID:              uuid.New(),
			UsageKind:       models.UsageKindCompute,
			MatchKey:        rate.MachineType,
			MatchMode:       models.UsagePriceBookMatchExact,
			MicrosPerSecond: rate.MicrosPerSecond,
		})
	}
	return rows
}

func appendUnique(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}
