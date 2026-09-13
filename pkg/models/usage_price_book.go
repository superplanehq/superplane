package models

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"gorm.io/gorm"
)

const (
	UsagePriceBookMatchExact  = "exact"
	UsagePriceBookMatchPrefix = "prefix"
	UsagePriceBookMatchFamily = "family"
)

// UsagePriceBook is one versioned catalog of model and compute rates.
type UsagePriceBook struct {
	Version     string `gorm:"primaryKey"`
	EffectiveAt time.Time
	CreatedAt   time.Time
	IsCurrent   bool
}

func (UsagePriceBook) TableName() string {
	return "usage_price_books"
}

// UsagePriceBookRate is one match rule inside a price book version.
type UsagePriceBookRate struct {
	ID                        uuid.UUID
	Version                   string
	UsageKind                 string
	MatchKey                  string
	MatchMode                 string
	InputCentsPerMillion      int64
	OutputCentsPerMillion     int64
	CacheReadCentsPerMillion  int64
	CacheWriteCentsPerMillion int64
	ReasoningCentsPerMillion  int64
	MicrosPerSecond           int64
}

func (UsagePriceBookRate) TableName() string {
	return "usage_price_book_rates"
}

// ListUsagePriceBooks returns every catalog version, newest effective_at first.
func ListUsagePriceBooks(tx *gorm.DB) ([]UsagePriceBook, error) {
	var books []UsagePriceBook
	err := tx.Order("effective_at DESC").Find(&books).Error
	if err != nil {
		return nil, err
	}
	return books, nil
}

// FindUsagePriceBook returns one catalog version.
func FindUsagePriceBook(tx *gorm.DB, version string) (*UsagePriceBook, error) {
	var book UsagePriceBook
	err := tx.Where("version = ?", version).First(&book).Error
	if err != nil {
		return nil, err
	}
	return &book, nil
}

// FindCurrentUsagePriceBook returns the catalog marked current.
func FindCurrentUsagePriceBook(tx *gorm.DB) (*UsagePriceBook, error) {
	var book UsagePriceBook
	err := tx.Where("is_current").First(&book).Error
	if err != nil {
		return nil, err
	}
	return &book, nil
}

// ListUsagePriceBookRates returns the match rules for one catalog version.
func ListUsagePriceBookRates(tx *gorm.DB, version string) ([]UsagePriceBookRate, error) {
	var rows []UsagePriceBookRate
	err := tx.Where("version = ?", version).Find(&rows).Error
	if err != nil {
		return nil, err
	}

	slices.SortFunc(rows, compareUsagePriceBookRates)
	return rows, nil
}

// NextUsagePriceBookVersion returns YYYY-MM-DD.N for now, incrementing N.
func NextUsagePriceBookVersion(tx *gorm.DB, now time.Time) (string, error) {
	date := now.UTC().Format("2006-01-02")
	prefix := date + "."
	var books []UsagePriceBook
	err := tx.Where("version LIKE ?", prefix+"%").Find(&books).Error
	if err != nil {
		return "", err
	}

	maxN := 0
	for _, book := range books {
		n, parseErr := strconv.Atoi(strings.TrimPrefix(book.Version, prefix))
		if parseErr != nil || n <= maxN {
			continue
		}
		maxN = n
	}
	return fmt.Sprintf("%s%d", prefix, maxN+1), nil
}

// CloneUsagePriceBookRates copies rates without ids or version stamps.
func CloneUsagePriceBookRates(rows []UsagePriceBookRate) []UsagePriceBookRate {
	cloned := make([]UsagePriceBookRate, 0, len(rows))
	for _, row := range rows {
		cloned = append(cloned, NormalizeUsagePriceBookRate(UsagePriceBookRate{
			UsageKind:                 row.UsageKind,
			MatchKey:                  row.MatchKey,
			MatchMode:                 row.MatchMode,
			InputCentsPerMillion:      row.InputCentsPerMillion,
			OutputCentsPerMillion:     row.OutputCentsPerMillion,
			CacheReadCentsPerMillion:  row.CacheReadCentsPerMillion,
			CacheWriteCentsPerMillion: row.CacheWriteCentsPerMillion,
			ReasoningCentsPerMillion:  row.ReasoningCentsPerMillion,
			MicrosPerSecond:           row.MicrosPerSecond,
		}))
	}
	return cloned
}

// NormalizeUsagePriceBookRate trims identity fields and lowercases the match key.
func NormalizeUsagePriceBookRate(row UsagePriceBookRate) UsagePriceBookRate {
	row.UsageKind = strings.TrimSpace(row.UsageKind)
	row.MatchKey = strings.ToLower(strings.TrimSpace(row.MatchKey))
	row.MatchMode = strings.TrimSpace(row.MatchMode)
	return row
}

// ValidateUsagePriceBookRates rejects empty keys, unknown modes, negatives, and duplicates.
func ValidateUsagePriceBookRates(rows []UsagePriceBookRate) error {
	seen := map[string]struct{}{}
	for _, row := range rows {
		normalized := NormalizeUsagePriceBookRate(row)
		if normalized.MatchKey == "" {
			return fmt.Errorf("match key cannot be empty")
		}
		switch normalized.UsageKind {
		case UsageKindModel:
			if normalized.MatchMode != UsagePriceBookMatchPrefix && normalized.MatchMode != UsagePriceBookMatchFamily {
				return fmt.Errorf("invalid model match mode: %s", normalized.MatchMode)
			}
			if normalized.InputCentsPerMillion < 0 ||
				normalized.OutputCentsPerMillion < 0 ||
				normalized.CacheReadCentsPerMillion < 0 ||
				normalized.CacheWriteCentsPerMillion < 0 ||
				normalized.ReasoningCentsPerMillion < 0 {
				return fmt.Errorf("model rates cannot be negative")
			}
		case UsageKindCompute:
			if normalized.MatchMode != UsagePriceBookMatchExact {
				return fmt.Errorf("invalid compute match mode: %s", normalized.MatchMode)
			}
			if normalized.MicrosPerSecond < 0 {
				return fmt.Errorf("compute rate cannot be negative")
			}
		default:
			return fmt.Errorf("invalid usage kind: %s", normalized.UsageKind)
		}

		id := normalized.UsageKind + "\x00" + normalized.MatchKey + "\x00" + normalized.MatchMode
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate rate: %s %s %s", normalized.UsageKind, normalized.MatchKey, normalized.MatchMode)
		}
		seen[id] = struct{}{}
	}
	return nil
}

// PublishUsagePriceBook inserts a new current catalog. It does not touch the
// in-memory book, because the caller can still roll the transaction back.
// Call LoadCurrentPriceBook after the commit.
func PublishUsagePriceBook(tx *gorm.DB, rates []UsagePriceBookRate) (*UsagePriceBook, error) {
	normalized := make([]UsagePriceBookRate, 0, len(rates))
	for _, row := range rates {
		normalized = append(normalized, NormalizeUsagePriceBookRate(row))
	}
	if err := ValidateUsagePriceBookRates(normalized); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	version, err := NextUsagePriceBookVersion(tx, now)
	if err != nil {
		return nil, err
	}

	if err := clearCurrentUsagePriceBook(tx); err != nil {
		return nil, err
	}

	book := UsagePriceBook{
		Version:     version,
		EffectiveAt: now,
		CreatedAt:   now,
		IsCurrent:   true,
	}
	if err := tx.Create(&book).Error; err != nil {
		return nil, err
	}

	if len(normalized) > 0 {
		for i := range normalized {
			normalized[i].ID = uuid.New()
			normalized[i].Version = version
		}
		if err := tx.Create(&normalized).Error; err != nil {
			return nil, err
		}
	}

	return &book, nil
}

// ActivateUsagePriceBook marks an existing version current. It does not touch
// the in-memory book, because the caller can still roll the transaction back.
// Call LoadCurrentPriceBook after the commit.
func ActivateUsagePriceBook(tx *gorm.DB, version string) error {
	version = strings.TrimSpace(version)
	book, err := FindUsagePriceBook(tx, version)
	if err != nil {
		return err
	}
	if book.IsCurrent {
		return nil
	}

	if err := clearCurrentUsagePriceBook(tx); err != nil {
		return err
	}
	return tx.Model(&UsagePriceBook{}).Where("version = ?", version).Update("is_current", true).Error
}

// LoadCurrentPriceBook installs the current database catalog into the
// in-memory price book. Callers keep the compiled-in fallback when this
// returns an error (empty table or missing migration).
func LoadCurrentPriceBook(tx *gorm.DB) error {
	book, err := FindCurrentUsagePriceBook(tx)
	if err != nil {
		return err
	}

	rows, err := ListUsagePriceBookRates(tx, book.Version)
	if err != nil {
		return err
	}

	loaded := pricebook.Book{
		Version:      book.Version,
		ComputeRates: map[string]int64{},
	}
	for _, row := range rows {
		switch strings.TrimSpace(row.UsageKind) {
		case UsageKindModel:
			rate := pricebook.Rate{
				Input:      row.InputCentsPerMillion,
				Output:     row.OutputCentsPerMillion,
				CacheRead:  row.CacheReadCentsPerMillion,
				CacheWrite: row.CacheWriteCentsPerMillion,
				Reasoning:  row.ReasoningCentsPerMillion,
			}
			switch row.MatchMode {
			case UsagePriceBookMatchPrefix:
				loaded.PrefixRates = append(loaded.PrefixRates, pricebook.PrefixRate{
					Prefix: row.MatchKey,
					Rate:   rate,
				})
			case UsagePriceBookMatchFamily:
				loaded.FamilyRates = append(loaded.FamilyRates, pricebook.FamilyRate{
					Token: row.MatchKey,
					Rate:  rate,
				})
			}
		case UsageKindCompute:
			loaded.ComputeRates[row.MatchKey] = row.MicrosPerSecond
		}
	}

	pricebook.Replace(loaded)
	return nil
}

func clearCurrentUsagePriceBook(tx *gorm.DB) error {
	return tx.Model(&UsagePriceBook{}).Where("is_current").Updates(map[string]any{"is_current": false}).Error
}

func compareUsagePriceBookRates(a, b UsagePriceBookRate) int {
	if keyOrder := strings.Compare(a.MatchKey, b.MatchKey); keyOrder != 0 {
		return keyOrder
	}
	return strings.Compare(a.MatchMode, b.MatchMode)
}
