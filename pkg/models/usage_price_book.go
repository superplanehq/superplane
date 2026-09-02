package models

import (
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

// LoadCurrentPriceBook installs the latest database catalog into the
// in-memory price book. Callers keep the compiled-in fallback when this
// returns an error (empty table or missing migration).
func LoadCurrentPriceBook(tx *gorm.DB) error {
	var book UsagePriceBook
	err := tx.Order("effective_at DESC").First(&book).Error
	if err != nil {
		return err
	}

	var rows []UsagePriceBookRate
	err = tx.Where("version = ?", book.Version).Find(&rows).Error
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
