package models

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrCatalogPlan = errors.New("invalid billing catalog plan")
var catalogKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type BillingPlan struct {
	ID                 uuid.UUID `json:"id" gorm:"primaryKey"`
	Key                string    `json:"key"`
	Name               string    `json:"name"`
	Category           string    `json:"category"`
	Visibility         string    `json:"visibility"`
	BasePriceCents     int64     `json:"base_price_cents"`
	SeatPriceCents     int64     `json:"seat_price_cents"`
	PolarProductID     *string   `json:"polar_product_id"`
	SeatLimit          *int      `json:"seat_limit"`
	ParallelTaskLimit  *int      `json:"parallel_task_limit"`
	CreditAmountCents  int64     `json:"credit_amount_cents"`
	CreditFrequency    string    `json:"credit_frequency"`
	CreditExpiration   string    `json:"credit_expiration"`
	CreditValidityDays *int      `json:"credit_validity_days"`
	TrialCreditCents   int64     `json:"trial_credit_cents"`
	TrialLengthDays    int       `json:"trial_length_days"`
	MarkupBPS          *int      `json:"markup_bps"`
	Features           []string  `json:"features" gorm:"serializer:json;type:jsonb"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type OrganizationFeatureGrant struct {
	OrganizationID uuid.UUID `json:"organization_id" gorm:"primaryKey"`
	FeatureKey     string    `json:"feature_key" gorm:"primaryKey"`
	Enabled        bool      `json:"enabled"`
}

func (p BillingPlan) ResponsePlan() string {
	if p.Key == "business_legacy" {
		return BillingPlanBusiness
	}
	return p.Key
}

func FindBillingPlan(tx *gorm.DB, key string) (*BillingPlan, error) {
	if key == BillingPlanBusiness {
		key = "business_legacy"
	}
	var plan BillingPlan
	err := tx.Where("key = ?", key).First(&plan).Error
	return &plan, err
}

func FindBillingPlanByProduct(tx *gorm.DB, productID string) (*BillingPlan, error) {
	var plan BillingPlan
	err := tx.Where("polar_product_id = ?", strings.TrimSpace(productID)).First(&plan).Error
	return &plan, err
}

func ListBillingPlans(tx *gorm.DB) ([]BillingPlan, error) {
	var plans []BillingPlan
	err := tx.Order("created_at, key").Find(&plans).Error
	return plans, err
}

func SaveBillingPlan(tx *gorm.DB, plan *BillingPlan) error {
	if err := validateBillingPlan(plan); err != nil {
		return err
	}
	return tx.Transaction(func(inner *gorm.DB) error {
		if plan.ID == uuid.Nil {
			plan.ID = uuid.New()
			return inner.Create(plan).Error
		}
		var current BillingPlan
		if err := inner.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "id = ?", plan.ID).Error; err != nil {
			return err
		}
		if current.Key != plan.Key {
			return fmt.Errorf("%w: plan keys cannot change", ErrCatalogPlan)
		}
		if current.PolarProductID != nil && (plan.PolarProductID == nil || *current.PolarProductID != *plan.PolarProductID || current.BasePriceCents != plan.BasePriceCents || current.SeatPriceCents != plan.SeatPriceCents) {
			return fmt.Errorf("%w: create a new plan and Polar product to change linked prices", ErrCatalogPlan)
		}
		if slices.Contains([]string{"none", "trial", "business_legacy"}, current.Key) && current.Category != plan.Category {
			return fmt.Errorf("%w: system plan categories cannot change", ErrCatalogPlan)
		}
		plan.CreatedAt = current.CreatedAt
		return inner.Save(plan).Error
	})
}

func DeleteBillingPlan(tx *gorm.DB, id uuid.UUID) error {
	return tx.Transaction(func(inner *gorm.DB) error {
		var plan BillingPlan
		if err := inner.Clauses(clause.Locking{Strength: "UPDATE"}).First(&plan, "id = ?", id).Error; err != nil {
			return err
		}
		if slices.Contains([]string{"none", "trial", "business_legacy"}, plan.Key) || plan.PolarProductID != nil {
			return fmt.Errorf("%w: system and Polar-linked plans cannot be deleted", ErrCatalogPlan)
		}
		var count int64
		if err := inner.Model(&OrganizationBillingPlan{}).Where("plan_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("%w: assigned plans cannot be deleted", ErrCatalogPlan)
		}
		return inner.Delete(&plan).Error
	})
}

// BindLegacyBillingProduct is idempotent and never replaces an existing binding.
func BindLegacyBillingProduct(tx *gorm.DB, productID string) error {
	if strings.TrimSpace(productID) == "" {
		return nil
	}
	return tx.Model(&BillingPlan{}).Where("key = ? AND polar_product_id IS NULL", "business_legacy").Update("polar_product_id", strings.TrimSpace(productID)).Error
}

func ListOrganizationFeatureGrants(tx *gorm.DB, orgID uuid.UUID) ([]OrganizationFeatureGrant, error) {
	var grants []OrganizationFeatureGrant
	err := tx.Where("organization_id = ?", orgID).Find(&grants).Error
	return grants, err
}

func SetOrganizationFeatureGrant(tx *gorm.DB, grant OrganizationFeatureGrant) error {
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "organization_id"}, {Name: "feature_key"}}, DoUpdates: clause.AssignmentColumns([]string{"enabled"})}).Create(&grant).Error
}

func validateBillingPlan(p *BillingPlan) error {
	invalid := func(message string) error { return fmt.Errorf("%w: %s", ErrCatalogPlan, message) }
	if p.Key == BillingPlanBusiness || !catalogKeyPattern.MatchString(p.Key) || strings.TrimSpace(p.Name) == "" {
		return invalid("a valid key and name are required")
	}
	if !slices.Contains([]string{"none", "trial", "paid"}, p.Category) || !slices.Contains([]string{"public", "private"}, p.Visibility) {
		return invalid("invalid category or visibility")
	}
	if p.BasePriceCents < 0 || p.SeatPriceCents < 0 || p.CreditAmountCents < 0 || p.TrialCreditCents < 0 || p.CreditAmountCents > math.MaxInt64/MicrosPerCent || p.TrialCreditCents > math.MaxInt64/MicrosPerCent {
		return invalid("invalid credit or price amount")
	}
	if p.SeatLimit != nil && *p.SeatLimit <= 0 || p.ParallelTaskLimit != nil && *p.ParallelTaskLimit <= 0 || p.MarkupBPS != nil && *p.MarkupBPS < 0 {
		return invalid("limits must be positive and markup cannot be negative")
	}
	if p.TrialLengthDays == 0 {
		p.TrialLengthDays = 14
	}
	if p.TrialLengthDays < 1 || p.TrialLengthDays > 36500 {
		return invalid("trial length must be between 1 and 36500 days")
	}
	if !slices.Contains([]string{"none", "once", "billing_period"}, p.CreditFrequency) || !slices.Contains([]string{"period_end", "after_days", "never"}, p.CreditExpiration) {
		return invalid("invalid credit policy")
	}
	if p.CreditExpiration == "after_days" && (p.CreditValidityDays == nil || *p.CreditValidityDays < 1 || *p.CreditValidityDays > 36500) {
		return invalid("positive validity days are required")
	}
	if p.Key == "business_legacy" && (p.CreditFrequency != "billing_period" || p.CreditExpiration != "period_end") {
		return invalid("legacy Business cadence and expiration are fixed")
	}
	if p.Key == "business_legacy" && (p.BasePriceCents != 19900 || p.SeatPriceCents != 0 || p.SeatLimit != nil || p.ParallelTaskLimit != nil) {
		return invalid("legacy Business prices and unlimited limits are fixed")
	}

	if p.Features == nil {
		p.Features = []string{}
	}
	if p.PolarProductID != nil && strings.TrimSpace(*p.PolarProductID) == "" {
		p.PolarProductID = nil
	}
	return nil
}
