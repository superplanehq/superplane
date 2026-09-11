package models

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	BillingPlanTrial    = "trial"
	BillingPlanBusiness = "business"
	BillingPlanNone     = "none"

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
	ErrHostedSubscriptionRequired = errors.New("a Business subscription is required for SuperPlane-hosted runs")
	ErrInvalidBillingPlan         = errors.New("billing plan is invalid")
)

// OrganizationBillingPlan is the org plan and Polar subscription period.
type OrganizationBillingPlan struct {
	OrganizationID          uuid.UUID `gorm:"primary_key"`
	Plan                    string
	PlanSource              string
	PolarSubscriptionID     *string
	PolarSubscriptionStatus string
	CurrentPeriodStart      *time.Time
	CurrentPeriodEnd        *time.Time
	TrialStartedAt          *time.Time
	TrialEndsAt             *time.Time
	UpdatedAt               time.Time
}

func (OrganizationBillingPlan) TableName() string {
	return "organization_billing_plans"
}

// HostedPlanGatesEnabled is true when Polar billing is configured.
// pkg/models cannot import pkg/billing/polar, so this reads POLAR_ACCESS_TOKEN
// the same way polar.Configured does.
func HostedPlanGatesEnabled() bool {
	return strings.TrimSpace(os.Getenv("POLAR_ACCESS_TOKEN")) != ""
}

func (p *OrganizationBillingPlan) IsActiveBusiness() bool {
	if p == nil || p.Plan != BillingPlanBusiness {
		return false
	}
	if p.PlanSource == BillingPlanSourceAdmin {
		return true
	}
	return PolarSubscriptionIsPaid(p.PolarSubscriptionStatus)
}

func (p *OrganizationBillingPlan) IsOpenTrial(now time.Time) bool {
	if p == nil || p.Plan != BillingPlanTrial {
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
	err := tx.Where("organization_id = ?", orgID).First(&plan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &plan, nil
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
	trialEnd := now.Add(DefaultWelcomeGrantTTL)
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

func SetAdminOrganizationPlan(tx *gorm.DB, orgID uuid.UUID, planName string) (*OrganizationBillingPlan, error) {
	if orgID == uuid.Nil {
		return nil, fmt.Errorf("organization is required")
	}
	switch planName {
	case BillingPlanTrial, BillingPlanBusiness, BillingPlanNone:
	default:
		return nil, ErrInvalidBillingPlan
	}

	var saved *OrganizationBillingPlan
	err := tx.Transaction(func(inner *gorm.DB) error {
		existing, err := FindOrganizationBillingPlan(inner, orgID)
		if err != nil {
			return err
		}

		now := time.Now()
		next := OrganizationBillingPlan{
			OrganizationID: orgID,
			Plan:           planName,
			PlanSource:     BillingPlanSourceAdmin,
			UpdatedAt:      now,
		}
		if planName == BillingPlanTrial {
			trialEnd := now.Add(DefaultWelcomeGrantTTL)
			next.TrialStartedAt = &now
			next.TrialEndsAt = &trialEnd
		}
		if existing != nil {
			next.PolarSubscriptionID = existing.PolarSubscriptionID
			next.PolarSubscriptionStatus = existing.PolarSubscriptionStatus
			next.CurrentPeriodStart = existing.CurrentPeriodStart
			next.CurrentPeriodEnd = existing.CurrentPeriodEnd
			if planName != BillingPlanTrial {
				next.TrialStartedAt = existing.TrialStartedAt
				next.TrialEndsAt = existing.TrialEndsAt
			}
		}
		assignIncludedUsagePeriod(&next, now)

		err = inner.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "organization_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"plan",
				"plan_source",
				"polar_subscription_id",
				"polar_subscription_status",
				"current_period_start",
				"current_period_end",
				"trial_started_at",
				"trial_ends_at",
				"updated_at",
			}),
		}).Create(&next).Error
		if err != nil {
			return err
		}

		saved, err = loadedOrganizationBillingPlan(inner, orgID)
		if err != nil {
			return err
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

func ApplyPolarSubscription(tx *gorm.DB, orgID uuid.UUID, subscriptionID, status string, periodStart, periodEnd *time.Time) (*OrganizationBillingPlan, bool, error) {
	existing, err := EnsureOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, false, err
	}
	if existing.PlanSource == BillingPlanSourceAdmin {
		return existing, false, nil
	}

	now := time.Now()
	next := *existing
	next.PlanSource = BillingPlanSourcePolar
	next.UpdatedAt = now
	if strings.TrimSpace(subscriptionID) != "" {
		id := strings.TrimSpace(subscriptionID)
		next.PolarSubscriptionID = &id
	}
	next.PolarSubscriptionStatus = strings.ToLower(strings.TrimSpace(status))
	if periodStart != nil {
		start := periodStart.UTC()
		next.CurrentPeriodStart = &start
	}
	if periodEnd != nil {
		end := periodEnd.UTC()
		next.CurrentPeriodEnd = &end
	}

	if PolarSubscriptionIsPaid(next.PolarSubscriptionStatus) {
		next.Plan = BillingPlanBusiness
	} else if polarSubscriptionIsCanceled(next.PolarSubscriptionStatus) {
		next.Plan = planAfterPaidSubscriptionEnds(&next, now)
	}

	err = tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "organization_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"plan",
			"plan_source",
			"polar_subscription_id",
			"polar_subscription_status",
			"current_period_start",
			"current_period_end",
			"trial_started_at",
			"trial_ends_at",
			"updated_at",
		}),
	}).Create(&next).Error
	if err != nil {
		return nil, false, err
	}

	saved, err := loadedOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, false, err
	}
	return saved, shouldGrantIncludedUsage(existing, saved), nil
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
	if plan == nil || plan.Plan != BillingPlanBusiness {
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
