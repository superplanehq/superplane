package models

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	BillingPlanTrial          = "trial"
	BillingPlanBusiness       = "business"
	BillingPlanBusinessManual = "business_manual"
	BillingPlanNone           = "none"

	BillingPlanSourceSystem = "system"
	BillingPlanSourcePolar  = "polar"
	BillingPlanSourceAdmin  = "admin"

	PolarSubscriptionStatusActive     = "active"
	PolarSubscriptionStatusTrialing   = "trialing"
	PolarSubscriptionStatusCanceled   = "canceled"
	PolarSubscriptionStatusUnpaid     = "unpaid"
	PolarSubscriptionStatusRevoked    = "revoked"
	PolarSubscriptionStatusPastDue    = "past_due"
	PolarSubscriptionStatusIncomplete = "incomplete"

	DefaultIncludedGrantCents  int64 = 5000
	DefaultIncludedGrantMonths       = 1
	DefaultTopupGrantMonths          = 12
)

var (
	ErrPaymentMethodRequired      = errors.New("add a payment method in organization billing before usage")
	ErrHostedSubscriptionRequired = errors.New("an active paid plan is required for SuperPlane-hosted runs")
	ErrInvalidBillingPlan         = errors.New("billing plan is invalid")
	ErrPolarManagedBillingPlan    = errors.New("this organization uses Polar for billing. Cancel or change the subscription in Polar")
)

// OrganizationBillingPlan is the org plan and Polar subscription period.
type OrganizationBillingPlan struct {
	OrganizationID          uuid.UUID `gorm:"primary_key"`
	PlanID                  uuid.UUID
	Catalog                 BillingPlan `gorm:"foreignKey:PlanID"`
	Plan                    string
	PlanSource              string
	PolarSubscriptionID     *string
	PolarSubscriptionStatus string
	CancelAtPeriodEnd       bool
	PolarModifiedAt         *time.Time
	CurrentPeriodStart      *time.Time
	CurrentPeriodEnd        *time.Time
	TrialStartedAt          *time.Time
	TrialEndsAt             *time.Time
	UpdatedAt               time.Time
}

// PolarSubscriptionApply is the Polar subscription snapshot to persist.
type PolarSubscriptionApply struct {
	ID                string
	ProductID         string
	Status            string
	PeriodStart       *time.Time
	PeriodEnd         *time.Time
	CancelAtPeriodEnd bool
	ModifiedAt        *time.Time
}

func (OrganizationBillingPlan) TableName() string {
	return "organization_billing_plans"
}

// BillingEnabledEnv turns product billing on or off.
// An empty or missing value is on. yes and true are on. no and false are off.
const BillingEnabledEnv = "SUPERPLANE_BILLING_ENABLED"

// BillingEnabled reports whether product billing is on.
// An empty or missing value is on, so SuperPlane Cloud and local development
// keep plans, credit limits, and checkout. A self-hosted install sets no.
func BillingEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(BillingEnabledEnv))) {
	case "no", "false":
		return false
	default:
		return true
	}
}

// HostedPlanGatesEnabled is true when Polar billing is configured.
// pkg/models cannot import pkg/billing/polar, so this reads POLAR_ACCESS_TOKEN
// the same way polar.Configured does. Product behavior also requires BillingEnabled.
func HostedPlanGatesEnabled() bool {
	return strings.TrimSpace(os.Getenv("POLAR_ACCESS_TOKEN")) != ""
}

func (p *OrganizationBillingPlan) IsActiveBusiness() bool {
	if p == nil || !p.isPaidPlan() {
		return false
	}
	if p.PlanSource == BillingPlanSourceAdmin {
		return true
	}
	if p.PlanSource == BillingPlanSourceSystem && p.Catalog.ID != uuid.Nil && p.Catalog.PolarProductID == nil && p.Catalog.PriceCents == 0 {
		return true
	}
	if PolarSubscriptionIsPaid(p.PolarSubscriptionStatus) {
		return true
	}
	return p.CancelAtPeriodEnd && periodStillOpen(p.CurrentPeriodEnd, time.Now())
}

func (p *OrganizationBillingPlan) IsOpenTrial(now time.Time) bool {
	if p == nil || (p.Catalog.ID != uuid.Nil && p.Catalog.Category != "trial" || p.Catalog.ID == uuid.Nil && p.Plan != BillingPlanTrial) {
		return false
	}
	if p.TrialEndsAt == nil {
		return false
	}
	return p.TrialEndsAt.After(now)
}

func (p *OrganizationBillingPlan) AllowsHostedExecution(now time.Time) bool {
	return p.IsActiveBusiness() || p.IsOpenTrial(now)
}

func (p *OrganizationBillingPlan) AllowsCreditPurchase() bool {
	return p.IsActiveBusiness()
}

func PolarSubscriptionIsPaid(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case PolarSubscriptionStatusActive, PolarSubscriptionStatusTrialing:
		return true
	default:
		return false
	}
}

func polarSubscriptionIsCanceled(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case PolarSubscriptionStatusCanceled, PolarSubscriptionStatusUnpaid,
		PolarSubscriptionStatusRevoked, PolarSubscriptionStatusPastDue:
		return true
	default:
		return false
	}
}

func FindOrganizationBillingPlan(tx *gorm.DB, orgID uuid.UUID) (*OrganizationBillingPlan, error) {
	var plan OrganizationBillingPlan
	err := tx.Preload("Catalog").Where("organization_id = ?", orgID).First(&plan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &plan, nil
}

func lockOrganizationBillingPlan(tx *gorm.DB, orgID uuid.UUID) (*OrganizationBillingPlan, error) {
	return FindOrganizationBillingPlan(tx.Clauses(clause.Locking{Strength: "UPDATE"}), orgID)
}

// ResolveOrganizationBillingPlan loads the org plan and writes plan=none when
// there is no row or the trial window has ended.
func ResolveOrganizationBillingPlan(tx *gorm.DB, orgID uuid.UUID) (*OrganizationBillingPlan, error) {
	if orgID == uuid.Nil {
		return nil, fmt.Errorf("organization is required")
	}

	plan, err := FindOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return persistNoneBillingPlan(tx, orgID)
	}
	now := time.Now()
	if plan.Catalog.Category == "trial" && !plan.IsOpenTrial(now) {
		return lapseExpiredTrial(tx, plan)
	}
	if plan.shouldLapseEndedPolarBusiness(now) {
		return lapseEndedPolarBusiness(tx, plan)
	}
	return plan, nil
}

func EnsureOrganizationBillingPlan(tx *gorm.DB, orgID uuid.UUID) (*OrganizationBillingPlan, error) {
	existing, err := FindOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	now := time.Now()
	catalog, err := FindBillingPlan(tx, BillingPlanTrial)
	if err != nil {
		return nil, err
	}
	trialEnd := now.AddDate(0, 0, catalog.TrialLengthDays)
	plan := OrganizationBillingPlan{
		OrganizationID: orgID,
		Plan:           BillingPlanTrial,
		PlanSource:     BillingPlanSourceSystem,
		TrialStartedAt: &now,
		TrialEndsAt:    &trialEnd,
		UpdatedAt:      now,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "organization_id"}},
		DoNothing: true,
	}).Create(&plan).Error
	if err != nil {
		return nil, err
	}
	return loadedOrganizationBillingPlan(tx, orgID)
}

func OrganizationBillingIsPolarManaged(plan *OrganizationBillingPlan) bool {
	if !BillingEnabled() || !HostedPlanGatesEnabled() {
		return false
	}
	return polarPaidAccessContinues(plan, time.Now())
}

func SetAdminOrganizationPlan(tx *gorm.DB, orgID uuid.UUID, planName string, trialEndsAt *time.Time) (*OrganizationBillingPlan, error) {
	if orgID == uuid.Nil {
		return nil, fmt.Errorf("organization is required")
	}
	// The existing admin API uses business for manual assignments.
	if planName == BillingPlanBusiness {
		planName = BillingPlanBusinessManual
	}
	if planName == "business_legacy" {
		return nil, ErrInvalidBillingPlan
	}
	catalog, err := FindBillingPlan(tx, planName)
	if err != nil {
		return nil, ErrInvalidBillingPlan
	}
	if catalog.RequirePaymentMethod {
		return nil, errors.New("Plans that require a payment method cannot be assigned by an admin")
	}
	return setOrganizationBillingPlan(tx, orgID, catalog, BillingPlanSourceAdmin, trialEndsAt)
}

// ActivateOrganizationBillingPlan assigns a public plan after the caller verifies its payment requirements.
// It assigns None when no public paid plans are available.
// Subscription plans must be activated by a Polar subscription event instead.
func ActivateOrganizationBillingPlan(tx *gorm.DB, orgID uuid.UUID, planKey string) (*OrganizationBillingPlan, error) {
	catalog, err := FindBillingPlan(tx, planKey)
	if err != nil {
		return nil, ErrInvalidBillingPlan
	}
	if catalog.Key == BillingPlanNone {
		plans, err := ListBillingPlans(tx)
		if err != nil {
			return nil, err
		}
		if slices.ContainsFunc(plans, func(plan BillingPlan) bool {
			return plan.Category == "paid" && plan.Visibility == "public"
		}) {
			return nil, ErrInvalidBillingPlan
		}
		return setOrganizationBillingPlan(tx, orgID, catalog, BillingPlanSourceSystem, nil)
	}
	if catalog.Category != "paid" || catalog.Visibility != "public" || catalog.PriceCents != 0 || catalog.PolarProductID != nil {
		return nil, ErrInvalidBillingPlan
	}
	return setOrganizationBillingPlan(tx, orgID, catalog, BillingPlanSourceSystem, nil)
}

func ApplyPolarSubscription(tx *gorm.DB, orgID uuid.UUID, sub PolarSubscriptionApply) (*OrganizationBillingPlan, bool, error) {
	var saved *OrganizationBillingPlan
	var grantIncluded bool
	err := tx.Transaction(func(inner *gorm.DB) error {
		existing, err := EnsureOrganizationBillingPlan(inner, orgID)
		if err != nil {
			return err
		}
		existing, err = lockOrganizationBillingPlan(inner, orgID)
		if err != nil {
			return err
		}
		if existing == nil {
			return fmt.Errorf("organization billing plan is missing")
		}
		if polarSnapshotIsStale(existing, sub.ModifiedAt) {
			log.WithFields(log.Fields{
				"organization_id":      orgID.String(),
				"polar_modified_at":    existing.PolarModifiedAt,
				"incoming_modified_at": sub.ModifiedAt,
			}).Warn("skipped stale polar subscription snapshot")
			saved = existing
			return nil
		}

		now := time.Now()
		next := *existing
		next.PlanSource = BillingPlanSourcePolar
		next.UpdatedAt = now
		next.PolarModifiedAt = polarSnapshotModifiedAt(existing.PolarModifiedAt, sub.ModifiedAt)
		if strings.TrimSpace(sub.ID) != "" {
			id := strings.TrimSpace(sub.ID)
			next.PolarSubscriptionID = &id
		}
		next.PolarSubscriptionStatus = strings.ToLower(strings.TrimSpace(sub.Status))
		next.CancelAtPeriodEnd = sub.CancelAtPeriodEnd
		if sub.PeriodStart != nil {
			start := sub.PeriodStart.UTC()
			next.CurrentPeriodStart = &start
		}
		if sub.PeriodEnd != nil {
			end := sub.PeriodEnd.UTC()
			next.CurrentPeriodEnd = &end
		}

		if polarPaidAccessContinues(&next, now) {
			catalog, err := FindBillingPlanByProduct(inner, sub.ProductID)
			if err != nil {
				return fmt.Errorf("unknown subscription product: %w", err)
			}
			if catalog.Category != "paid" {
				return ErrInvalidBillingPlan
			}
			next.Plan = catalog.ResponsePlan()
			next.PlanID = catalog.ID
			next.Catalog = *catalog
		} else if polarSubscriptionIsCanceled(next.PolarSubscriptionStatus) || polarScheduledCancelEnded(&next, now) {
			next.Plan = planAfterPaidSubscriptionEnds(&next, now)
			catalog, err := FindBillingPlan(inner, next.Plan)
			if err != nil {
				return err
			}
			next.PlanID = catalog.ID
			next.Catalog = *catalog
		}

		err = inner.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "organization_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"plan",
				"plan_id",
				"plan_source",
				"polar_subscription_id",
				"polar_subscription_status",
				"cancel_at_period_end",
				"polar_modified_at",
				"current_period_start",
				"current_period_end",
				"trial_started_at",
				"trial_ends_at",
				"updated_at",
			}),
		}).Omit("Catalog").Create(&next).Error
		if err != nil {
			return err
		}

		saved, err = loadedOrganizationBillingPlan(inner, orgID)
		if err != nil {
			return err
		}
		if err := addInitialPlanTrialCredit(inner, existing, saved); err != nil {
			return err
		}
		grantIncluded = shouldGrantIncludedUsage(existing, saved)
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return saved, grantIncluded, nil
}

func loadedOrganizationBillingPlan(tx *gorm.DB, orgID uuid.UUID) (*OrganizationBillingPlan, error) {
	plan, err := FindOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, fmt.Errorf("organization billing plan is missing")
	}
	return plan, nil
}

func periodStartChanged(previous, next *time.Time) bool {
	if next == nil {
		return false
	}
	if previous == nil {
		return true
	}
	return !previous.Equal(*next)
}

func shouldGrantIncludedUsage(existing, next *OrganizationBillingPlan) bool {
	if next == nil || !next.IsActiveBusiness() {
		return false
	}
	becamePaid := existing == nil || !existing.IsActiveBusiness()
	if becamePaid {
		return true
	}
	return periodStartChanged(existing.CurrentPeriodStart, next.CurrentPeriodStart)
}

func assignIncludedUsagePeriod(plan *OrganizationBillingPlan, now time.Time) {
	if plan == nil || !plan.isPaidPlan() {
		return
	}
	if plan.CurrentPeriodStart != nil && plan.CurrentPeriodEnd != nil {
		return
	}
	start := now.UTC().Truncate(time.Second)
	end := start.AddDate(0, DefaultIncludedGrantMonths, 0)
	plan.CurrentPeriodStart = &start
	plan.CurrentPeriodEnd = &end
}

func includedUsageSubscriptionID(plan *OrganizationBillingPlan) string {
	if plan == nil {
		return ""
	}
	if plan.PolarSubscriptionID != nil {
		if id := strings.TrimSpace(*plan.PolarSubscriptionID); id != "" {
			return id
		}
	}
	return "admin:" + plan.OrganizationID.String()
}

func planAfterPaidSubscriptionEnds(plan *OrganizationBillingPlan, now time.Time) string {
	if plan != nil && plan.TrialEndsAt != nil && plan.TrialEndsAt.After(now) {
		return BillingPlanTrial
	}
	return BillingPlanNone
}

func polarPaidAccessContinues(plan *OrganizationBillingPlan, now time.Time) bool {
	if plan == nil || polarScheduledCancelEnded(plan, now) {
		return false
	}
	if PolarSubscriptionIsPaid(plan.PolarSubscriptionStatus) {
		return true
	}
	return plan.CancelAtPeriodEnd && periodStillOpen(plan.CurrentPeriodEnd, now)
}

func polarScheduledCancelEnded(plan *OrganizationBillingPlan, now time.Time) bool {
	return plan != nil && plan.CancelAtPeriodEnd && !periodStillOpen(plan.CurrentPeriodEnd, now)
}

func polarSnapshotIsStale(existing *OrganizationBillingPlan, incoming *time.Time) bool {
	if existing == nil || existing.PolarModifiedAt == nil || incoming == nil {
		return false
	}
	return incoming.Before(*existing.PolarModifiedAt)
}

func polarSnapshotModifiedAt(existing, incoming *time.Time) *time.Time {
	if incoming != nil {
		at := incoming.UTC()
		return &at
	}
	return existing
}

func (p *OrganizationBillingPlan) shouldLapseEndedPolarBusiness(now time.Time) bool {
	if p == nil || !p.isPaidPlan() || p.PlanSource == BillingPlanSourceAdmin {
		return false
	}
	return polarScheduledCancelEnded(p, now)
}

func periodStillOpen(periodEnd *time.Time, now time.Time) bool {
	return periodEnd != nil && periodEnd.After(now)
}

func lapseEndedPolarBusiness(tx *gorm.DB, plan *OrganizationBillingPlan) (*OrganizationBillingPlan, error) {
	var saved *OrganizationBillingPlan
	err := tx.Transaction(func(inner *gorm.DB) error {
		now := time.Now()
		nextPlan := planAfterPaidSubscriptionEnds(plan, now)
		result := inner.Model(&OrganizationBillingPlan{}).
			Where("organization_id = ? AND plan = ? AND plan_source <> ?", plan.OrganizationID, plan.Plan, BillingPlanSourceAdmin).
			Where("cancel_at_period_end = ?", true).
			Where("current_period_end IS NOT NULL AND current_period_end <= ?", now).
			Updates(map[string]any{
				"plan":       nextPlan,
				"plan_id":    inner.Model(&BillingPlan{}).Select("id").Where("key = ?", nextPlan),
				"updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}

		loaded, err := loadedOrganizationBillingPlan(inner, plan.OrganizationID)
		if err != nil {
			return err
		}
		saved = loaded
		return SyncIncludedLLMCreditGrant(inner, IncludedUsageSync{
			OrganizationID:   plan.OrganizationID,
			SubscriptionID:   includedUsageSubscriptionID(saved),
			IsActiveBusiness: saved.IsActiveBusiness(),
		})
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}

func persistNoneBillingPlan(tx *gorm.DB, orgID uuid.UUID) (*OrganizationBillingPlan, error) {
	now := time.Now()
	plan := OrganizationBillingPlan{
		OrganizationID: orgID,
		Plan:           BillingPlanNone,
		PlanSource:     BillingPlanSourceSystem,
		UpdatedAt:      now,
	}
	err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "organization_id"}},
		DoNothing: true,
	}).Create(&plan).Error
	if err != nil {
		return nil, err
	}
	return loadedOrganizationBillingPlan(tx, orgID)
}

func lapseExpiredTrial(tx *gorm.DB, plan *OrganizationBillingPlan) (*OrganizationBillingPlan, error) {
	var saved *OrganizationBillingPlan
	err := tx.Transaction(func(inner *gorm.DB) error {
		now := time.Now()
		result := inner.Model(&OrganizationBillingPlan{}).
			Where("organization_id = ? AND plan = ?", plan.OrganizationID, plan.Plan).
			Where("trial_ends_at IS NULL OR trial_ends_at <= ?", now).
			Updates(map[string]any{
				"plan":       BillingPlanNone,
				"plan_id":    inner.Model(&BillingPlan{}).Select("id").Where("key = ?", BillingPlanNone),
				"updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}

		loaded, err := loadedOrganizationBillingPlan(inner, plan.OrganizationID)
		if err != nil {
			return err
		}
		saved = loaded
		return SyncIncludedLLMCreditGrant(inner, IncludedUsageSync{
			OrganizationID:   plan.OrganizationID,
			SubscriptionID:   includedUsageSubscriptionID(saved),
			IsActiveBusiness: saved.IsActiveBusiness(),
		})
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}

func (p *OrganizationBillingPlan) isPaidPlan() bool {
	if p.Catalog.ID != uuid.Nil {
		return p.Catalog.Category == "paid"
	}
	return p.Plan == BillingPlanBusiness
}

func (p *OrganizationBillingPlan) BeforeCreate(tx *gorm.DB) error {
	if p.PlanID != uuid.Nil {
		return nil
	}
	key := p.Plan
	if key == BillingPlanBusiness && p.PlanSource == BillingPlanSourceAdmin {
		key = BillingPlanBusinessManual
	}
	catalog, err := FindBillingPlan(tx, key)
	if err != nil {
		return err
	}
	p.PlanID = catalog.ID
	return nil
}

func adminTrialEnd(now time.Time, trialEndsAt *time.Time) time.Time {
	if trialEndsAt == nil {
		return now.Add(DefaultWelcomeGrantTTL)
	}
	return trialEndsAt.UTC()
}

func adminTrialStart(existing *OrganizationBillingPlan, now time.Time) *time.Time {
	if existing != nil && existing.IsOpenTrial(now) && existing.TrialStartedAt != nil {
		started := *existing.TrialStartedAt
		return &started
	}
	started := now
	return &started
}

func setOrganizationBillingPlan(tx *gorm.DB, orgID uuid.UUID, catalog *BillingPlan, source string, trialEndsAt *time.Time) (*OrganizationBillingPlan, error) {
	if orgID == uuid.Nil {
		return nil, fmt.Errorf("organization is required")
	}
	planName := catalog.ResponsePlan()
	var saved *OrganizationBillingPlan
	err := tx.Transaction(func(inner *gorm.DB) error {
		existing, err := lockOrganizationBillingPlan(inner, orgID)
		if err != nil {
			return err
		}
		if OrganizationBillingIsPolarManaged(existing) {
			return ErrPolarManagedBillingPlan
		}

		now := time.Now()
		next := OrganizationBillingPlan{
			OrganizationID: orgID,
			Plan:           planName,
			PlanID:         catalog.ID,
			Catalog:        *catalog,
			PlanSource:     source,
			UpdatedAt:      now,
		}
		if catalog.Category == "trial" {
			trialEnd := now.AddDate(0, 0, catalog.TrialLengthDays)
			if trialEndsAt != nil {
				trialEnd = trialEndsAt.UTC()
			}
			next.TrialStartedAt = adminTrialStart(existing, now)
			next.TrialEndsAt = &trialEnd
		}
		if existing != nil && (source == BillingPlanSourceAdmin || existing.PlanSource == source) {
			next.PolarSubscriptionID = existing.PolarSubscriptionID
			next.PolarSubscriptionStatus = existing.PolarSubscriptionStatus
			next.CancelAtPeriodEnd = existing.CancelAtPeriodEnd
			next.PolarModifiedAt = existing.PolarModifiedAt
			next.CurrentPeriodStart = existing.CurrentPeriodStart
			next.CurrentPeriodEnd = existing.CurrentPeriodEnd
		}
		if existing == nil || existing.PlanID != next.PlanID || existing.PlanSource != source {
			next.CurrentPeriodStart = nil
			next.CurrentPeriodEnd = nil
		}
		assignIncludedUsagePeriod(&next, now)

		err = inner.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "organization_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"plan",
				"plan_id",
				"plan_source",
				"polar_subscription_id",
				"polar_subscription_status",
				"cancel_at_period_end",
				"polar_modified_at",
				"current_period_start",
				"current_period_end",
				"trial_started_at",
				"trial_ends_at",
				"updated_at",
			}),
		}).Omit("Catalog").Create(&next).Error
		if err != nil {
			return err
		}

		saved, err = loadedOrganizationBillingPlan(inner, orgID)
		if err != nil {
			return err
		}
		if err := addInitialPlanTrialCredit(inner, existing, saved); err != nil {
			return err
		}
		if catalog.Category == "trial" && trialEndsAt != nil && saved.TrialEndsAt != nil {
			if err := SetWelcomeGrantExpiresAt(inner, orgID, *saved.TrialEndsAt); err != nil {
				return err
			}
		}
		return SyncIncludedLLMCreditGrant(inner, IncludedUsageSync{
			OrganizationID:   orgID,
			SubscriptionID:   includedUsageSubscriptionID(saved),
			PeriodEnd:        saved.CurrentPeriodEnd,
			GrantIncluded:    shouldGrantIncludedUsage(existing, saved),
			IsActiveBusiness: saved.IsActiveBusiness(),
		})
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}
