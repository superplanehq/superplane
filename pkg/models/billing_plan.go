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
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrCatalogPlan = errors.New("invalid billing catalog plan")
var catalogKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// pgUniqueViolation is the PostgreSQL SQLSTATE for unique constraint violations.
const pgUniqueViolation = "23505"

const (
	billingPlanKeyConflictMessage     = "This plan key already exists. Enter a different key."
	billingPlanProductConflictMessage = "This Polar product is already linked to another plan. Select a different product."
)

const (
	BillingPlanDeleteReasonSystem   = "System plans cannot be deleted."
	BillingPlanDeleteReasonPolar    = "This plan is linked to a Polar product and cannot be deleted."
	BillingPlanDeleteReasonAssigned = "One or more organizations use this plan. Assign them another plan before deleting it."
)

var systemBillingPlanKeys = []string{"none", "trial", "business_legacy", BillingPlanBusinessManual}

type BillingPlan struct {
	ID                   uuid.UUID `json:"id" gorm:"primaryKey"`
	Key                  string    `json:"key"`
	Name                 string    `json:"name"`
	Category             string    `json:"category"`
	RequirePaymentMethod bool      `json:"require_payment_method"`
	Visibility           string    `json:"visibility"`
	PriceCents           int64     `json:"price_cents"`
	PerSeat              bool      `json:"per_seat"`
	PolarProductID       *string   `json:"polar_product_id"`
	SeatLimit            *int      `json:"seat_limit"`
	ParallelTaskLimit    *int      `json:"parallel_task_limit"`
	CreditAmountCents    int64     `json:"credit_amount_cents"`
	CreditFrequency      string    `json:"credit_frequency"`
	CreditExpiration     string    `json:"credit_expiration"`
	CreditValidityDays   *int      `json:"credit_validity_days"`
	TrialCreditCents     int64     `json:"trial_credit_cents"`
	TrialLengthDays      int       `json:"trial_length_days"`
	MarkupBPS            *int      `json:"markup_bps"`
	Features             []string  `json:"features" gorm:"serializer:json;type:jsonb"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type OrganizationFeatureGrant struct {
	OrganizationID uuid.UUID `json:"organization_id" gorm:"primaryKey"`
	FeatureKey     string    `json:"feature_key" gorm:"primaryKey"`
	Enabled        bool      `json:"enabled"`
}

// BillingPlanAssignments is a set of billing plan IDs referenced by organizations.
type BillingPlanAssignments map[uuid.UUID]struct{}

func (p BillingPlan) ResponsePlan() string {
	if p.Key == "business_legacy" || p.Key == BillingPlanBusinessManual {
		return BillingPlanBusiness
	}
	return p.Key
}

// DeleteBlockReason returns the first rule that blocks deletion, or an empty
// string when the plan can be deleted. Callers pass whether any organization
// references the plan. Subscription status does not affect the result.
func (p BillingPlan) DeleteBlockReason(assigned bool) string {
	switch {
	case p.IsSystem():
		return BillingPlanDeleteReasonSystem
	case p.PolarProductID != nil:
		return BillingPlanDeleteReasonPolar
	case assigned:
		return BillingPlanDeleteReasonAssigned
	}
	return ""
}

// IsSystem reports whether the plan key is reserved for a built-in plan.
func (p BillingPlan) IsSystem() bool {
	return slices.Contains(systemBillingPlanKeys, p.Key)
}

// Contains reports whether the set references the plan.
func (a BillingPlanAssignments) Contains(planID uuid.UUID) bool {
	_, ok := a[planID]
	return ok
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

// LockBillingPlanByProduct finds the plan for a Polar product and locks the
// row FOR UPDATE. The subscription assignment path holds the lock through the
// write of organization_billing_plans.plan_id. SaveBillingPlan takes the same
// row lock, so an admin cannot unlink or re-link the plan between the lookup
// and the assignment.
func LockBillingPlanByProduct(tx *gorm.DB, productID string) (*BillingPlan, error) {
	var plan BillingPlan
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("polar_product_id = ?", strings.TrimSpace(productID)).
		First(&plan).Error
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
	err := tx.Transaction(func(inner *gorm.DB) error {
		if err := lockPlanFeatureReferences(inner, plan.Features); err != nil {
			return err
		}
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
		if err := checkLinkedPlanChange(inner, &current, plan); err != nil {
			return err
		}
		if current.IsSystem() && current.Category != plan.Category {
			return fmt.Errorf("%w: system plan categories cannot change", ErrCatalogPlan)
		}
		plan.CreatedAt = current.CreatedAt
		return inner.Save(plan).Error
	})
	return billingPlanConflictError(err)
}

// billingPlanConflictError maps the unique constraints on plan keys and Polar
// products to user-facing catalog errors. Other errors pass through unchanged.
func billingPlanConflictError(err error) error {
	var conflict *pgconn.PgError
	if !errors.As(err, &conflict) || conflict.Code != pgUniqueViolation {
		return err
	}
	switch conflict.ConstraintName {
	case "billing_plans_key_key":
		return fmt.Errorf("%w: %s", ErrCatalogPlan, billingPlanKeyConflictMessage)
	case "billing_plans_polar_product_id_key":
		return fmt.Errorf("%w: %s", ErrCatalogPlan, billingPlanProductConflictMessage)
	}
	return err
}

// DeleteBillingPlan removes a plan when no deletion rule blocks it. The row
// lock keeps assignment checks and deletion in one transaction.
func DeleteBillingPlan(tx *gorm.DB, id uuid.UUID) error {
	return tx.Transaction(func(inner *gorm.DB) error {
		var plan BillingPlan
		if err := inner.Clauses(clause.Locking{Strength: "UPDATE"}).First(&plan, "id = ?", id).Error; err != nil {
			return err
		}
		assigned, err := ListAssignedBillingPlanIDs(inner, []uuid.UUID{plan.ID})
		if err != nil {
			return err
		}
		if reason := plan.DeleteBlockReason(assigned.Contains(plan.ID)); reason != "" {
			return fmt.Errorf("%w: %s", ErrCatalogPlan, reason)
		}
		return inner.Delete(&plan).Error
	})
}

// ListAssignedBillingPlanIDs returns the plan IDs that any organization
// references, regardless of subscription status. It runs one query for all IDs.
func ListAssignedBillingPlanIDs(tx *gorm.DB, planIDs []uuid.UUID) (BillingPlanAssignments, error) {
	assigned := BillingPlanAssignments{}
	if len(planIDs) == 0 {
		return assigned, nil
	}
	var ids []uuid.UUID
	err := tx.Model(&OrganizationBillingPlan{}).
		Where("plan_id IN ?", planIDs).
		Distinct("plan_id").
		Pluck("plan_id", &ids).
		Error
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		assigned[id] = struct{}{}
	}
	return assigned, nil
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
	return tx.Transaction(func(inner *gorm.DB) error {
		if err := lockPlanFeatureReferences(inner, []string{grant.FeatureKey}); err != nil {
			return err
		}
		return inner.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "organization_id"}, {Name: "feature_key"}}, DoUpdates: clause.AssignmentColumns([]string{"enabled"})}).Create(&grant).Error
	})
}

// checkLinkedPlanChange rejects product, price, or seat changes on a
// Polar-linked plan that an organization uses. The caller holds a FOR UPDATE
// lock on the plan row. That lock conflicts with the KEY SHARE lock that the
// organization_billing_plans.plan_id foreign key takes, so a concurrent
// assignment waits until the save finishes.
func checkLinkedPlanChange(tx *gorm.DB, current, next *BillingPlan) error {
	if current.PolarProductID == nil {
		return nil
	}
	changed := next.PolarProductID == nil ||
		*current.PolarProductID != *next.PolarProductID ||
		current.PriceCents != next.PriceCents ||
		current.PerSeat != next.PerSeat
	if !changed {
		return nil
	}
	assigned, err := ListAssignedBillingPlanIDs(tx, []uuid.UUID{current.ID})
	if err != nil {
		return err
	}
	if assigned.Contains(current.ID) {
		return fmt.Errorf("%w: create a new plan and Polar product to change linked prices", ErrCatalogPlan)
	}
	return nil
}

func validateBillingPlan(p *BillingPlan) error {
	invalid := func(message string) error { return fmt.Errorf("%w: %s", ErrCatalogPlan, message) }
	if p.Key == BillingPlanBusiness || !catalogKeyPattern.MatchString(p.Key) || strings.TrimSpace(p.Name) == "" {
		return invalid("a valid key and name are required")
	}
	if !slices.Contains([]string{"none", "trial", "paid"}, p.Category) || !slices.Contains([]string{"public", "private"}, p.Visibility) {
		return invalid("invalid category or visibility")
	}
	if p.PriceCents < 0 || p.CreditAmountCents < 0 || p.TrialCreditCents < 0 || p.CreditAmountCents > math.MaxInt64/MicrosPerCent || p.TrialCreditCents > math.MaxInt64/MicrosPerCent {
		return invalid("invalid credit or price amount")
	}
	if p.PerSeat && (p.SeatLimit == nil || *p.SeatLimit < 1 || *p.SeatLimit > 1000) {
		return invalid("maximum seats must be between 1 and 1,000 for per-seat plans")
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
	if (p.Key == "business_legacy" || p.Key == BillingPlanBusinessManual) && (p.CreditFrequency != "billing_period" || p.CreditExpiration != "period_end") {
		return invalid("Business cadence and expiration are fixed")
	}
	if p.Key == "business_legacy" && (p.PriceCents != 19900 || p.PerSeat || p.SeatLimit != nil || p.ParallelTaskLimit != nil) {
		return invalid("legacy Business prices and unlimited limits are fixed")
	}

	if p.Features == nil {
		p.Features = []string{}
	}
	if p.PolarProductID != nil && strings.TrimSpace(*p.PolarProductID) == "" {
		p.PolarProductID = nil
	}
	if p.Key == BillingPlanBusinessManual && p.PolarProductID != nil {
		return invalid("Business Manual cannot use a Polar product")
	}
	return nil
}
