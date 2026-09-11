package models

import (
	"slices"
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

// LoadCurrentPriceBook installs the latest database catalog into the
// in-memory price book. Callers keep the compiled-in fallback when this
// returns an error (empty table or missing migration).
func LoadCurrentPriceBook(tx *gorm.DB) error {
	books, err := ListUsagePriceBooks(tx)
	if err != nil {
		return err
	}
	if len(books) == 0 {
		return gorm.ErrRecordNotFound
	}
	book := books[0]

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

func compareUsagePriceBookRates(a, b UsagePriceBookRate) int {
	if keyOrder := strings.Compare(a.MatchKey, b.MatchKey); keyOrder != 0 {
		return keyOrder
	}
	return strings.Compare(a.MatchMode, b.MatchMode)
}
