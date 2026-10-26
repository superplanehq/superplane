package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrCreditPack = errors.New("invalid credit pack")

type CreditPack struct {
	ID   uuid.UUID `json:"id" gorm:"primaryKey"`
	Name string    `json:"name"`
	// A zero amount allows the customer to choose the credit amount.
	AmountCents    int64          `json:"amount_cents"`
	Enabled        bool           `json:"enabled"`
	PolarProductID *string        `json:"polar_product_id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-"`
}

func ListCreditPacks(tx *gorm.DB) ([]CreditPack, error) {
	packs := []CreditPack{}
	err := tx.Order("amount_cents = 0, amount_cents, created_at, id").Find(&packs).Error
	return packs, err
}

// CreditPackCatalogConfigured includes deleted packs so removing the last pack
// cannot restore products from the legacy Polar catalog.
func CreditPackCatalogConfigured(tx *gorm.DB) (bool, error) {
	var count int64
	err := tx.Unscoped().Model(&CreditPack{}).Count(&count).Error
	return count > 0, err
}

func FindCreditPackByProduct(tx *gorm.DB, productID string) (*CreditPack, error) {
	var pack CreditPack
	err := tx.Where("polar_product_id = ?", strings.TrimSpace(productID)).First(&pack).Error
	return &pack, err
}

func SaveCreditPack(tx *gorm.DB, pack *CreditPack) error {
	pack.Name = strings.TrimSpace(pack.Name)
	if pack.Name == "" {
		return fmt.Errorf("%w: enter a credit pack name", ErrCreditPack)
	}
	if pack.AmountCents != 0 && (pack.AmountCents < 50 || pack.AmountCents > 99999999) {
		return fmt.Errorf("%w: enter a credit amount between $0.50 and $999,999.99", ErrCreditPack)
	}
	if pack.PolarProductID != nil {
		productID := strings.TrimSpace(*pack.PolarProductID)
		pack.PolarProductID = nil
		if productID != "" {
			pack.PolarProductID = &productID
		}
	}
	err := tx.Transaction(func(inner *gorm.DB) error {
		if pack.ID == uuid.Nil {
			pack.ID = uuid.New()
			return inner.Create(pack).Error
		}
		var current CreditPack
		if err := inner.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "id = ?", pack.ID).Error; err != nil {
			return err
		}
		pack.CreatedAt = current.CreatedAt
		return inner.Save(pack).Error
	})
	var conflict *pgconn.PgError
	if errors.As(err, &conflict) && conflict.ConstraintName == "credit_packs_polar_product_id_idx" {
		return fmt.Errorf("%w: this Polar product is already linked to another credit pack", ErrCreditPack)
	}
	return err
}

func DeleteCreditPack(tx *gorm.DB, id uuid.UUID) error {
	result := tx.Delete(&CreditPack{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
