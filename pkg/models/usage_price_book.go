package models

import (
	"fmt"
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
			case UsagePriceBookMatchExact:
				loaded.ExactRates = append(loaded.ExactRates, pricebook.ExactRate{
					Key:  row.MatchKey,
					Rate: rate,
				})
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

// FindLatestPriceBook returns the newest catalog version by effective_at.
func FindLatestPriceBook(tx *gorm.DB) (*UsagePriceBook, error) {
	var book UsagePriceBook
	err := tx.Order("effective_at DESC").First(&book).Error
	if err != nil {
		return nil, err
	}
	return &book, nil
}

// ListPriceBookRates returns every match rule for one catalog version.
func ListPriceBookRates(tx *gorm.DB, version string) ([]UsagePriceBookRate, error) {
	var rows []UsagePriceBookRate
	err := tx.Where("version = ?", version).
		Order("usage_kind ASC, match_mode ASC, match_key ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// NextPriceBookVersion returns date.N for today in UTC, starting at .1.
func NextPriceBookVersion(tx *gorm.DB, now time.Time) (string, error) {
	prefix := now.UTC().Format("2006-01-02") + "."
	var versions []string
	err := tx.Model(&UsagePriceBook{}).
		Where("version LIKE ?", prefix+"%").
		Pluck("version", &versions).Error
	if err != nil {
		return "", err
	}

	max := 0
	for _, version := range versions {
		n, parseErr := strconv.Atoi(strings.TrimPrefix(version, prefix))
		if parseErr != nil {
			continue
		}
		if n > max {
			max = n
		}
	}
	return fmt.Sprintf("%s%d", prefix, max+1), nil
}

// InsertPriceBook writes one catalog version and its rates in the caller transaction.
func InsertPriceBook(tx *gorm.DB, book UsagePriceBook, rates []UsagePriceBookRate) error {
	if strings.TrimSpace(book.Version) == "" {
		return fmt.Errorf("price book version is required")
	}
	if book.EffectiveAt.IsZero() {
		book.EffectiveAt = time.Now().UTC()
	}
	if book.CreatedAt.IsZero() {
		book.CreatedAt = book.EffectiveAt
	}
	if err := tx.Create(&book).Error; err != nil {
		return err
	}
	if len(rates) == 0 {
		return nil
	}
	for i := range rates {
		if rates[i].ID == uuid.Nil {
			rates[i].ID = uuid.New()
		}
		rates[i].Version = book.Version
	}
	return tx.Create(&rates).Error
}
