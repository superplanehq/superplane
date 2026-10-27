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
	"github.com/superplanehq/superplane/pkg/billing"
	coremodels "github.com/superplanehq/superplane/pkg/models"
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
	PolarSubscriptionStatusPaused     = "paused"
	PolarSubscriptionStatusIncomplete = "incomplete"

	PolarSubscriptionStatusIncompleteExpired = "incomplete_expired"

	DefaultIncludedGrantCents  int64 = 5000
	DefaultIncludedGrantMonths       = 1
	DefaultTopupGrantMonths          = 12
)

var (
	ErrPaymentMethodRequired          = billing.ErrPaymentMethodRequired
	ErrHostedSubscriptionRequired     = billing.ErrHostedSubscriptionRequired
	ErrInvalidBillingPlan             = errors.New("billing plan is invalid")
	ErrInvalidPurchasedSeats          = errors.New("purchased seats must be a nonnegative integer")
	ErrPolarSubscriptionRefreshFailed = errors.New("stored Polar subscription refresh failed")
	ErrBillingPlanChanged             = billing.ErrBillingPlanChanged
	ErrPolarManagedBillingPlan        = errors.New("this organization uses Polar for billing. Cancel or change the subscription in Polar")
)

// ErrHostedPaymentPastDue wraps ErrHostedSubscriptionRequired, so callers that
// match the broader error keep working.
var ErrHostedPaymentPastDue = billing.ErrHostedPaymentPastDue

var ErrHostedSubscriptionPaused = billing.ErrHostedSubscriptionPaused

// OrganizationBillingPlan is the org plan and Polar subscription period.
// PastDueSince is when the current renewal payment failure started. It is set
// only while the subscription status is past_due.
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
	PurchasedSeats          *int
	PastDueSince            *time.Time
	CurrentPeriodStart      *time.Time
	CurrentPeriodEnd        *time.Time
	TrialStartedAt          *time.Time
	TrialEndsAt             *time.Time
	UpdatedAt               time.Time

	// RenewalSeatsScheduled is the last next-period seat count that the
	// retired automatic renewal scheduler sent to Polar, and
	// RenewalSeatsScheduledAt is when it was sent. Nothing writes them now.
	// They stay so that legacy cleanup can identify verified automatic
	// reductions and keep schedules that an Admin made.
	RenewalSeatsScheduled   *int
	RenewalSeatsScheduledAt *time.Time

	// OverCapacitySince is when a per-seat plan first had more members than
	// purchased seats. A Polar snapshot never writes it.
	// UpdateSeatCapacityState owns it.
	OverCapacitySince *time.Time
}

// SeatCapacityChange tells whether a capacity check started or ended the
// over-capacity state.
type SeatCapacityChange string

const (
	SeatCapacityUnchanged SeatCapacityChange = "unchanged"
	SeatCapacityStarted   SeatCapacityChange = "started"
	SeatCapacityResolved  SeatCapacityChange = "resolved"
)

// SeatCapacityTransition is the result of UpdateSeatCapacityState. The caller
// logs it after its transaction commits.
type SeatCapacityTransition struct {
	Change         SeatCapacityChange
	SubscriptionID string
	Purchased      int
	Occupied       int
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
	PurchasedSeats    *int
	PastDueAt         *time.Time
	// Verified marks a snapshot that a direct Polar API call returned. Webhook
	// payloads are unverified: they can arrive delayed and claim access that
	// Polar already ended.
	Verified bool
}

type PolarSubscriptionDisposition string

const (
	PolarSubscriptionApplied         PolarSubscriptionDisposition = "applied"
	PolarSubscriptionIgnored         PolarSubscriptionDisposition = "ignored"
	PolarSubscriptionRefreshRequired PolarSubscriptionDisposition = "refresh_required"
	// PolarSubscriptionVerificationRequired asks the caller to fetch the
	// incoming subscription from the Polar API and retry with the verified
	// snapshot before it can replace an ended stored subscription.
	PolarSubscriptionVerificationRequired PolarSubscriptionDisposition = "verification_required"
)

// PolarSubscriptionApplyResult describes an accepted or ignored snapshot.
// RefreshedStored carries the ended subscription's effects before a replacement.
type PolarSubscriptionApplyResult struct {
	Plan            *OrganizationBillingPlan
	GrantIncluded   bool
	Disposition     PolarSubscriptionDisposition
	RefreshedStored *PolarSubscriptionApplyResult
}

func (OrganizationBillingPlan) TableName() string {
	return "organization_billing_plans"
}

// BillingEnabledEnv turns product billing on or off.
// An empty or missing value is on. yes and true are on. no and false are off.
const BillingEnabledEnv = billing.BillingEnabledEnv

// BillingEnabled reports whether product billing is on.
// An empty or missing value is on, so SuperPlane Cloud and local development
// keep plans, credit limits, and checkout. A self-hosted install sets no.
func BillingEnabled() bool {
	return billing.Enabled()
}

// HostedPlanGatesEnabled is true when Polar billing is configured.
// Models read the same provider setting as polar.Configured without importing
// the Polar service. Product behavior also requires BillingEnabled.
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
	if PolarSubscriptionKeepsAccess(p.PolarSubscriptionStatus, p.CancelAtPeriodEnd, p.CurrentPeriodEnd, time.Now()) {
		return true
	}
	return p.holdsPastDueGraceRow()
}

// IsPaidPerSeat is true for an active paid plan that sells seats.
func (p *OrganizationBillingPlan) IsPaidPerSeat() bool {
	return p != nil && p.Catalog.PerSeat && p.IsActiveBusiness()
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

// PolarSubscriptionKeepsAccess reports whether a subscription snapshot still
// gives paid access. An active or trialing subscription does. A subscription
// set to cancel keeps access until its period ends, unless payment failed or
// the customer paused it. Those subscriptions must not regain access by
// setting "cancel at period end" in the Polar portal.
func PolarSubscriptionKeepsAccess(status string, cancelAtPeriodEnd bool, periodEnd *time.Time, now time.Time) bool {
	if PolarSubscriptionIsPaid(status) {
		return true
	}
	if polarSubscriptionCannotCollectPayment(status) {
		return false
	}
	return cancelAtPeriodEnd && periodStillOpen(periodEnd, now)
}

// HasLivePolarSubscription reports whether Polar can still charge for this
// plan. An elapsed scheduled cancellation does not end a subscription: Polar
// can resume it, and only a terminal status from Polar proves the end.
// Admin and system plans do not have a live Polar subscription.
func (p *OrganizationBillingPlan) HasLivePolarSubscription(now time.Time) bool {
	if !p.hasStoredPolarSubscription() {
		return false
	}
	return PolarSubscriptionIsLive(p.PolarSubscriptionStatus, p.CancelAtPeriodEnd, p.CurrentPeriodEnd, now)
}

// MayBeBillable reports whether the stored row cannot prove that Polar
// stopped billing. Only a terminal status proves the end. An empty or unknown
// status, and a scheduled cancellation whose period elapsed while the status
// stays non-terminal, can still bill. Deletion needs Polar to confirm the end
// of such a subscription.
func (p *OrganizationBillingPlan) MayBeBillable(now time.Time) bool {
	if !p.hasStoredPolarSubscription() {
		return false
	}
	return !PolarSubscriptionIsTerminal(p.PolarSubscriptionStatus, p.CancelAtPeriodEnd, p.CurrentPeriodEnd, now)
}

// PolarSubscriptionIsTerminal reports whether Polar ended the subscription
// and cannot bill it again. A canceled subscription that keeps its paid
// period open is not terminal yet.
func PolarSubscriptionIsTerminal(status string, cancelAtPeriodEnd bool, periodEnd *time.Time, now time.Time) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case PolarSubscriptionStatusCanceled:
		return !cancelAtPeriodEnd || !periodStillOpen(periodEnd, now)
	case PolarSubscriptionStatusRevoked, PolarSubscriptionStatusIncompleteExpired:
		return true
	default:
		return false
	}
}

// PolarSubscriptionIsPastDue is true while Polar retries a failed renewal payment.
func PolarSubscriptionIsPastDue(status string) bool {
	return strings.ToLower(strings.TrimSpace(status)) == PolarSubscriptionStatusPastDue
}

// PolarSubscriptionIsPaymentBlocked is true while Polar does not collect
// payment for the subscription, because the renewal failed or the customer
// paused it. No new members can join in either state.
func PolarSubscriptionIsPaymentBlocked(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case PolarSubscriptionStatusPastDue, PolarSubscriptionStatusPaused:
		return true
	default:
		return false
	}
}

// PolarSubscriptionIsLive protects subscription identity, including failed renewals
// and paused subscriptions that can resume.
// Paid access remains governed by PolarSubscriptionIsPaid and the billing period.
func PolarSubscriptionIsLive(status string, cancelAtPeriodEnd bool, periodEnd *time.Time, now time.Time) bool {
	return PolarSubscriptionIsPaid(status) || PolarSubscriptionIsPastDue(status) ||
		strings.ToLower(strings.TrimSpace(status)) == PolarSubscriptionStatusPaused ||
		cancelAtPeriodEnd && periodStillOpen(periodEnd, now)
}

// PolarSubscriptionStatusIsKnown rejects unusable snapshots before conflict resolution.
func PolarSubscriptionStatusIsKnown(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case PolarSubscriptionStatusActive, PolarSubscriptionStatusTrialing, PolarSubscriptionStatusCanceled,
		PolarSubscriptionStatusUnpaid, PolarSubscriptionStatusRevoked, PolarSubscriptionStatusPastDue,
		PolarSubscriptionStatusIncomplete, PolarSubscriptionStatusIncompleteExpired, PolarSubscriptionStatusPaused:
		return true
	default:
		return false
	}
}

// PastDueAccessEndsAt is when Team features end after a failed renewal.
// It returns nil when the subscription is not past due or no grace applies.
func (p *OrganizationBillingPlan) PastDueAccessEndsAt(graceDays int) *time.Time {
	if p == nil || p.PastDueSince == nil || graceDays <= 0 || !PolarSubscriptionIsPastDue(p.PolarSubscriptionStatus) {
		return nil
	}
	end := p.PastDueSince.Add(pastDueGrace(graceDays))
	return &end
}

// AssignAdminPlanAfterPolarVerification requires a confirmed absence from Polar.
// Keep the subscription ID and event timestamp for history and webhook ordering.
func (p *OrganizationBillingPlan) AssignAdminPlanAfterPolarVerification(tx *gorm.DB, planName string, trialEndsAt *time.Time) (*OrganizationBillingPlan, error) {
	var saved *OrganizationBillingPlan
	err := tx.Transaction(func(inner *gorm.DB) error {
		current, err := lockOrganizationBillingPlan(inner, p.OrganizationID)
		if err != nil {
			return err
		}
		if current == nil || !current.UpdatedAt.Equal(p.UpdatedAt) {
			return ErrBillingPlanChanged
		}
		if err := inner.Model(current).Updates(map[string]any{
			"polar_subscription_status": "", "cancel_at_period_end": false, "past_due_since": nil,
		}).Error; err != nil {
			return err
		}
		saved, err = SetAdminOrganizationPlan(inner, p.OrganizationID, planName, trialEndsAt)
		return err
	})
	return saved, err
}

// polarSubscriptionEndsPaidPlan is true when a snapshot moves the organization
// off the paid plan at once: Polar ended the subscription, or it no longer
// collects payment. ApplyPolarSubscription checks the past-due grace period
// first, so only a past-due snapshot without grace lapses here.
func polarSubscriptionEndsPaidPlan(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case PolarSubscriptionStatusCanceled, PolarSubscriptionStatusUnpaid,
		PolarSubscriptionStatusRevoked, PolarSubscriptionStatusPastDue,
		PolarSubscriptionStatusPaused:
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

// ListSeatRenewalSyncPlans returns the seat-based plans whose stored Polar
// subscription can still change, so that the seat renewal sync can reconcile
// them with Polar. The plan that renews first comes first.
//
// A plan is seat-based when its catalog plan is per seat or when it keeps
// purchased seats. A past-due, unpaid, or paused subscription lapses to a
// catalog plan that is not per seat but keeps its purchased seats, so it stays
// in the list. A canceling subscription stays too. Only a canceled or revoked
// subscription is final and leaves the list.
//
// This list decides which plans the sync reads from Polar. The sync never
// writes a seat count.
func ListSeatRenewalSyncPlans(tx *gorm.DB) ([]OrganizationBillingPlan, error) {
	var plans []OrganizationBillingPlan
	err := tx.
		Joins("JOIN billing_plans ON billing_plans.id = organization_billing_plans.plan_id").
		Joins("JOIN organizations ON organizations.id = organization_billing_plans.organization_id").
		Where("organizations.deleted_at IS NULL").
		Where("organization_billing_plans.plan_source = ?", BillingPlanSourcePolar).
		Where("organization_billing_plans.polar_subscription_id IS NOT NULL AND organization_billing_plans.polar_subscription_id <> ''").
		Where("lower(organization_billing_plans.polar_subscription_status) NOT IN ?", []string{PolarSubscriptionStatusCanceled, PolarSubscriptionStatusRevoked}).
		Where("billing_plans.per_seat = ? OR organization_billing_plans.purchased_seats IS NOT NULL", true).
		Order("organization_billing_plans.current_period_end ASC NULLS LAST").
		Order("organization_billing_plans.organization_id").
		Find(&plans).Error
	if err != nil {
		return nil, err
	}
	return plans, nil
}

// UpdateSeatCapacityState records when a per-seat plan starts or stops having
// more members than purchased seats. It counts the active members while it
// holds the billing plan row lock, so a concurrent membership change commits
// before the count or waits for it. It does not log. The caller logs the
// returned transition after commit.
//
// A plan that is not an active per-seat plan has no over-capacity state. The
// record clears without a transition, so a later recovery alerts again.
func UpdateSeatCapacityState(tx *gorm.DB, orgID uuid.UUID, now time.Time) (SeatCapacityTransition, error) {
	transition := SeatCapacityTransition{Change: SeatCapacityUnchanged}
	err := tx.Transaction(func(inner *gorm.DB) error {
		plan, err := lockOrganizationBillingPlan(inner, orgID)
		if err != nil || plan == nil {
			return err
		}
		// A payment grace period keeps Team features, but admission stays
		// blocked. Recovery must start a fresh capacity alert.
		if !plan.IsPaidPerSeat() || PolarSubscriptionIsPaymentBlocked(plan.PolarSubscriptionStatus) {
			return plan.setOverCapacitySince(inner, nil)
		}

		count, err := coremodels.CountActiveHumanUsersByOrganization(inner, orgID.String())
		if err != nil {
			return err
		}
		occupied := int(count)

		purchased := 0
		if plan.PurchasedSeats != nil {
			purchased = *plan.PurchasedSeats
		}
		transition.Purchased = purchased
		transition.Occupied = occupied
		if plan.PolarSubscriptionID != nil {
			transition.SubscriptionID = *plan.PolarSubscriptionID
		}

		overCapacity := occupied > purchased
		switch {
		case overCapacity && plan.OverCapacitySince == nil:
			transition.Change = SeatCapacityStarted
			return plan.setOverCapacitySince(inner, &now)
		case !overCapacity && plan.OverCapacitySince != nil:
			transition.Change = SeatCapacityResolved
			return plan.setOverCapacitySince(inner, nil)
		default:
			return nil
		}
	})
	return transition, err
}

// setOverCapacitySince writes only that column. It keeps updated_at, which
// other flows use to detect a changed billing record.
func (p *OrganizationBillingPlan) setOverCapacitySince(tx *gorm.DB, since *time.Time) error {
	if p.OverCapacitySince == nil && since == nil {
		return nil
	}
	return tx.Model(&OrganizationBillingPlan{}).
		Where("organization_id = ?", p.OrganizationID).
		UpdateColumn("over_capacity_since", since).
		Error
}

func lockOrganizationBillingPlan(tx *gorm.DB, orgID uuid.UUID) (*OrganizationBillingPlan, error) {
	return FindOrganizationBillingPlan(tx.Clauses(clause.Locking{Strength: "UPDATE"}), orgID)
}

// ResolveOrganizationBillingPlan loads the org plan and writes plan=none when
// there is no row or the trial window has ended. It also ends a scheduled
// cancellation and a past-due grace period, because Polar sends no event then.
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
	if plan.shouldLapsePaymentBlockedCatalog() {
		return lapsePaymentBlockedCatalog(tx, plan)
	}
	if !plan.holdsPastDueGraceRow() {
		return plan, nil
	}
	graceDays, err := installationPastDueGraceDays(tx)
	if err != nil {
		return nil, err
	}
	if plan.pastDueGraceEnded(now, graceDays) {
		return lapseEndedPastDueGrace(tx, plan, graceDays)
	}
	return plan, nil
}

// AssertOrganizationPlanAllowsRuns rejects an agent run when the organization
// plan is not an active paid plan or an open trial. Remaining credit does not
// override the plan. The gate applies to hosted and BYOK runs, with or without
// Polar. It does nothing when billing is off.
func AssertOrganizationPlanAllowsRuns(tx *gorm.DB, orgID uuid.UUID) error {
	if !BillingEnabled() {
		return nil
	}
	plan, err := ResolveOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return err
	}
	if plan.AllowsHostedExecution(time.Now()) {
		return nil
	}
	if PolarSubscriptionIsPastDue(plan.PolarSubscriptionStatus) {
		return ErrHostedPaymentPastDue
	}
	if strings.EqualFold(strings.TrimSpace(plan.PolarSubscriptionStatus), PolarSubscriptionStatusPaused) {
		return ErrHostedSubscriptionPaused
	}
	return ErrHostedSubscriptionRequired
}

// EnsureOrganizationBillingPlan creates a None plan for an organization that
// has no billing row. The source is system and the row has no trial dates.
// An existing plan, including an open trial or a paid subscription, stays as stored.
func EnsureOrganizationBillingPlan(tx *gorm.DB, orgID uuid.UUID) (*OrganizationBillingPlan, error) {
	existing, err := FindOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	return persistNoneBillingPlan(tx, orgID)
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
	var saved *OrganizationBillingPlan
	err := tx.Transaction(func(inner *gorm.DB) error {
		// Match the subscription path's lock order: organization billing row,
		// then catalog row. Keep both locks through validation and assignment.
		if _, err := lockOrganizationBillingPlan(inner, orgID); err != nil {
			return err
		}
		catalog, err := FindBillingPlan(inner.Clauses(clause.Locking{Strength: "UPDATE"}), planName)
		if err != nil {
			return ErrInvalidBillingPlan
		}
		if catalog.PerSeat {
			// Preserve the billing API's required override guidance.
			//revive:disable-next-line:error-strings
			return errors.New("Per-seat plans need a Polar subscription. Use a seat limit override instead.")
		}
		if catalog.RequirePaymentMethod {
			return errors.New("Plans that require a payment method cannot be assigned by an admin")
		}
		saved, err = setOrganizationBillingPlan(inner, orgID, catalog, BillingPlanSourceAdmin, trialEndsAt)
		return err
	})
	return saved, err
}

// ActivateOrganizationBillingPlan assigns a public plan after the caller verifies its payment requirements.
// It assigns None when no public paid plans are available.
// Subscription plans must be activated by a Polar subscription event instead.
func ActivateOrganizationBillingPlan(tx *gorm.DB, orgID uuid.UUID, planKey string) (*OrganizationBillingPlan, error) {
	var saved *OrganizationBillingPlan
	err := tx.Transaction(func(inner *gorm.DB) error {
		if _, err := coremodels.LockOrganization(inner, orgID); err != nil {
			return err
		}
		pending, err := FindUnresolvedOrganizationBillingCheckout(inner, orgID)
		if err != nil {
			return err
		}
		if pending != nil {
			return errors.New("a checkout is already in progress")
		}
		current, err := FindOrganizationBillingPlan(inner, orgID)
		if err != nil {
			return err
		}
		if current.HasLivePolarSubscription(time.Now()) {
			return ErrBillingCheckoutLiveSubscription
		}
		saved, err = activateOrganizationBillingPlan(inner, orgID, planKey)
		return err
	})
	return saved, err
}

func activateOrganizationBillingPlan(tx *gorm.DB, orgID uuid.UUID, planKey string) (*OrganizationBillingPlan, error) {
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

// ApplyPolarSubscription checks identity before snapshot ordering or side effects.
// The caller can refresh a conflicting stored subscription and retry under its lock.
func ApplyPolarSubscription(tx *gorm.DB, orgID uuid.UUID, sub PolarSubscriptionApply, refreshedStored *PolarSubscriptionApply) (*PolarSubscriptionApplyResult, error) {
	var result *PolarSubscriptionApplyResult
	err := tx.Transaction(func(inner *gorm.DB) error {
		if _, err := coremodels.LockOrganization(inner, orgID); err != nil {
			return err
		}
		existing, err := lockOrganizationBillingPlan(inner, orgID)
		if err != nil {
			return err
		}
		if existing == nil {
			existing, err = EnsureOrganizationBillingPlan(inner, orgID)
			if err != nil {
				return err
			}
		}

		now := time.Now()
		// Ended subscriptions are history after an admin assigns a local plan.
		// Replaying them must not remove access granted by that plan.
		if existing.PlanSource == BillingPlanSourceAdmin && !PolarSubscriptionIsLive(sub.Status, sub.CancelAtPeriodEnd, sub.PeriodEnd, now) {
			result = &PolarSubscriptionApplyResult{Plan: existing, Disposition: PolarSubscriptionIgnored}
			return nil
		}
		var refreshedResult *PolarSubscriptionApplyResult
		if existing.PolarSubscriptionID != nil && strings.TrimSpace(*existing.PolarSubscriptionID) != "" && !samePolarSubscription(existing, sub.ID) {
			fields := log.Fields{
				"organization_id":              orgID.String(),
				"stored_subscription_id":       strings.TrimSpace(*existing.PolarSubscriptionID),
				"incoming_subscription_id":     strings.TrimSpace(sub.ID),
				"stored_subscription_status":   existing.PolarSubscriptionStatus,
				"incoming_subscription_status": strings.ToLower(strings.TrimSpace(sub.Status)),
			}
			if PolarSubscriptionIsLive(existing.PolarSubscriptionStatus, existing.CancelAtPeriodEnd, existing.CurrentPeriodEnd, now) {
				if !PolarSubscriptionIsLive(sub.Status, sub.CancelAtPeriodEnd, sub.PeriodEnd, now) {
					if err := settleIgnoredForeignSeatPurchase(inner, orgID, sub); err != nil {
						return err
					}
					log.WithFields(fields).Warn("billing.foreign_subscription_snapshot_ignored")
					result = &PolarSubscriptionApplyResult{Plan: existing, Disposition: PolarSubscriptionIgnored}
					return nil
				}
				if refreshedStored == nil {
					result = &PolarSubscriptionApplyResult{Plan: existing, Disposition: PolarSubscriptionRefreshRequired}
					return nil
				}
				if !samePolarSubscription(existing, refreshedStored.ID) {
					return fmt.Errorf("%w: subscription does not match stored identity", ErrPolarSubscriptionRefreshFailed)
				}
				if !PolarSubscriptionStatusIsKnown(refreshedStored.Status) {
					return fmt.Errorf("%w: unexpected status %q", ErrPolarSubscriptionRefreshFailed, refreshedStored.Status)
				}
				// Same-subscription snapshots retain omitted periods when applied.
				refreshedPeriodEnd := refreshedStored.PeriodEnd
				if refreshedPeriodEnd == nil {
					refreshedPeriodEnd = existing.CurrentPeriodEnd
				}
				if PolarSubscriptionIsLive(refreshedStored.Status, refreshedStored.CancelAtPeriodEnd, refreshedPeriodEnd, now) {
					fields["stored_subscription_status"] = strings.ToLower(strings.TrimSpace(refreshedStored.Status))
					log.WithFields(fields).Error("billing.multiple_live_subscriptions")
					result = &PolarSubscriptionApplyResult{Plan: existing, Disposition: PolarSubscriptionIgnored}
					return nil
				}
				if polarSnapshotIsStale(existing, refreshedStored.ModifiedAt) {
					return fmt.Errorf("%w: cancellation is older than stored subscription", ErrPolarSubscriptionRefreshFailed)
				}
				refreshedResult, err = applyPolarSubscriptionSnapshot(inner, existing, *refreshedStored)
				if err != nil {
					return fmt.Errorf("%w: %w", ErrPolarSubscriptionRefreshFailed, err)
				}
				existing = refreshedResult.Plan
			}
			// The stored subscription has ended, so its timestamp no longer
			// orders a foreign snapshot: a delayed event can claim access that
			// Polar already ended. Accept a different subscription only after
			// the caller verifies it against the Polar API, and only while the
			// verified subscription is live.
			if !PolarSubscriptionIsLive(sub.Status, sub.CancelAtPeriodEnd, sub.PeriodEnd, now) {
				if err := settleIgnoredForeignSeatPurchase(inner, orgID, sub); err != nil {
					return err
				}
				log.WithFields(fields).Warn("billing.foreign_subscription_snapshot_ignored")
				result = &PolarSubscriptionApplyResult{Plan: existing, Disposition: PolarSubscriptionIgnored, RefreshedStored: refreshedResult}
				return nil
			}
			if !sub.Verified {
				result = &PolarSubscriptionApplyResult{Plan: existing, Disposition: PolarSubscriptionVerificationRequired, RefreshedStored: refreshedResult}
				return nil
			}
		}

		result, err = applyPolarSubscriptionSnapshot(inner, existing, sub)
		if err != nil {
			return err
		}
		result.RefreshedStored = refreshedResult
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func samePolarSubscription(existing *OrganizationBillingPlan, subscriptionID string) bool {
	return existing.PolarSubscriptionID != nil && strings.TrimSpace(*existing.PolarSubscriptionID) == strings.TrimSpace(subscriptionID)
}

// hasStoredPolarSubscription is true for a Polar-managed row that names a
// subscription. Admin and system rows keep an old ID only for history.
func (p *OrganizationBillingPlan) hasStoredPolarSubscription() bool {
	return p != nil && p.PlanSource == BillingPlanSourcePolar &&
		p.PolarSubscriptionID != nil && strings.TrimSpace(*p.PolarSubscriptionID) != ""
}

// settleIgnoredForeignSeatPurchase settles the in-flight purchase of a
// subscription whose snapshot cannot replace the stored one. A snapshot that
// proves the purchase settles it without a change to the replacement's state.
func settleIgnoredForeignSeatPurchase(tx *gorm.DB, orgID uuid.UUID, sub PolarSubscriptionApply) error {
	return SettleSeatPurchase(tx, orgID, sub.seatPurchaseEvidence())
}

// seatPurchaseEvidence is the snapshot as evidence for a seat purchase. A
// snapshot is never the response to a purchase request.
func (s PolarSubscriptionApply) seatPurchaseEvidence() SeatPurchaseEvidence {
	return SeatPurchaseEvidence{
		SubscriptionID: s.ID,
		PeriodStart:    s.PeriodStart,
		Seats:          s.PurchasedSeats,
		ModifiedAt:     s.ModifiedAt,
	}
}

func applyPolarSubscriptionSnapshot(tx *gorm.DB, existing *OrganizationBillingPlan, sub PolarSubscriptionApply) (*PolarSubscriptionApplyResult, error) {
	orgID := existing.OrganizationID
	if samePolarSubscription(existing, sub.ID) && polarSnapshotIsStale(existing, sub.ModifiedAt) {
		log.WithFields(log.Fields{
			"organization_id":      orgID.String(),
			"polar_modified_at":    existing.PolarModifiedAt,
			"incoming_modified_at": sub.ModifiedAt,
		}).Warn("skipped stale polar subscription snapshot")
		return &PolarSubscriptionApplyResult{Plan: existing, Disposition: PolarSubscriptionIgnored}, nil
	}
	// Only a snapshot fresh enough to apply can prove the success of an
	// in-flight seat purchase. A stale snapshot can report seats that
	// Polar no longer has, so it must not settle a purchase. A snapshot
	// that does not prove success still updates the capacity below, and the
	// purchase keeps its status.
	if err := SettleSeatPurchase(tx, orgID, sub.seatPurchaseEvidence()); err != nil {
		return nil, err
	}
	// Lock the plan row before reading its fields and keep the lock through
	// the plan_id write. SaveBillingPlan takes the same lock, so a
	// concurrent admin save cannot unlink or re-link the plan between this
	// lookup and the assignment.
	catalog, catalogErr := LockBillingPlanByProduct(tx, sub.ProductID)
	if catalogErr != nil && !errors.Is(catalogErr, gorm.ErrRecordNotFound) {
		return nil, catalogErr
	}
	if sub.PurchasedSeats != nil && *sub.PurchasedSeats < 0 || catalog.PerSeat && sub.PurchasedSeats == nil {
		return nil, ErrInvalidPurchasedSeats
	}

	now := time.Now()
	next := *existing
	next.PlanSource = BillingPlanSourcePolar
	next.UpdatedAt = now
	if !samePolarSubscription(existing, sub.ID) {
		next.PolarModifiedAt = nil
		next.CurrentPeriodStart = nil
		next.CurrentPeriodEnd = nil
	}
	next.PolarModifiedAt = polarSnapshotModifiedAt(next.PolarModifiedAt, sub.ModifiedAt)
	next.PurchasedSeats = sub.PurchasedSeats
	if strings.TrimSpace(sub.ID) != "" {
		id := strings.TrimSpace(sub.ID)
		next.PolarSubscriptionID = &id
	}
	next.PolarSubscriptionStatus = strings.ToLower(strings.TrimSpace(sub.Status))
	next.CancelAtPeriodEnd = sub.CancelAtPeriodEnd
	next.PastDueSince = nextPastDueSince(existing, sub, now)
	if sub.PeriodStart != nil {
		start := sub.PeriodStart.UTC()
		next.CurrentPeriodStart = &start
	}
	if sub.PeriodEnd != nil {
		end := sub.PeriodEnd.UTC()
		next.CurrentPeriodEnd = &end
	}

	keepsPastDueGrace := false
	if catalogErr == nil && catalog.Category == "paid" && next.PastDueSince != nil {
		graceDays, err := installationPastDueGraceDays(tx)
		if err != nil {
			return nil, err
		}
		keepsPastDueGrace = next.pastDueGraceOpen(now, graceDays)
	}
	if polarPaidAccessContinues(&next, now) || keepsPastDueGrace {
		if catalogErr != nil {
			return nil, fmt.Errorf("unknown subscription product: %w", catalogErr)
		}
		if catalog.Category != "paid" {
			return nil, ErrInvalidBillingPlan
		}
		next.Plan = catalog.ResponsePlan()
		next.PlanID = catalog.ID
		next.Catalog = *catalog
	} else if polarSubscriptionEndsPaidPlan(next.PolarSubscriptionStatus) || polarScheduledCancelEnded(&next, now) {
		next.Plan = planAfterPaidSubscriptionEnds(&next, now)
		catalog, err := FindBillingPlan(tx, next.Plan)
		if err != nil {
			return nil, err
		}
		next.PlanID = catalog.ID
		next.Catalog = *catalog
	}

	err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "organization_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"plan",
			"plan_id",
			"plan_source",
			"polar_subscription_id",
			"polar_subscription_status",
			"cancel_at_period_end",
			"polar_modified_at",
			"purchased_seats",
			"past_due_since",
			"current_period_start",
			"current_period_end",
			"trial_started_at",
			"trial_ends_at",
			"updated_at",
		}),
	}).Omit("Catalog").Create(&next).Error
	if err != nil {
		return nil, err
	}

	saved, err := loadedOrganizationBillingPlan(tx, orgID)
	if err != nil {
		return nil, err
	}
	if err := addInitialPlanTrialCredit(tx, existing, saved); err != nil {
		return nil, err
	}
	return &PolarSubscriptionApplyResult{Plan: saved, GrantIncluded: shouldGrantIncludedUsage(existing, saved), Disposition: PolarSubscriptionApplied}, nil
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
	return PolarSubscriptionKeepsAccess(plan.PolarSubscriptionStatus, plan.CancelAtPeriodEnd, plan.CurrentPeriodEnd, now)
}

// nextPastDueSince keeps the start of one failure across snapshots. It is
// empty for every status except past_due, so a later failure starts a new one.
func nextPastDueSince(existing *OrganizationBillingPlan, sub PolarSubscriptionApply, now time.Time) *time.Time {
	if !PolarSubscriptionIsPastDue(sub.Status) {
		return nil
	}
	if samePolarSubscription(existing, sub.ID) && existing.PastDueSince != nil {
		return existing.PastDueSince
	}
	since := now
	if sub.PastDueAt != nil {
		since = *sub.PastDueAt
	}
	since = since.UTC()
	return &since
}

// holdsPastDueGraceRow is true for a past-due Polar row that still sits on a
// paid catalog plan. Only a grace period keeps a row in that state.
func (p *OrganizationBillingPlan) holdsPastDueGraceRow() bool {
	return p != nil && p.PastDueSince != nil && p.isPaidPlan() &&
		p.PlanSource == BillingPlanSourcePolar && PolarSubscriptionIsPastDue(p.PolarSubscriptionStatus)
}

func pastDueGrace(graceDays int) time.Duration {
	return time.Duration(graceDays) * 24 * time.Hour
}

// installationPastDueGraceDays reads the grace setting. Only a past-due
// subscription needs it, so other statuses skip the read.
func installationPastDueGraceDays(tx *gorm.DB) (int, error) {
	return GetPastDueGraceDays(tx)
}

func (p *OrganizationBillingPlan) pastDueGraceOpen(now time.Time, graceDays int) bool {
	if p == nil || p.PastDueSince == nil || !PolarSubscriptionIsPastDue(p.PolarSubscriptionStatus) || polarScheduledCancelEnded(p, now) {
		return false
	}
	return now.Before(p.PastDueSince.Add(pastDueGrace(graceDays)))
}

func (p *OrganizationBillingPlan) pastDueGraceEnded(now time.Time, graceDays int) bool {
	return p.holdsPastDueGraceRow() && !now.Before(p.PastDueSince.Add(pastDueGrace(graceDays)))
}

func polarScheduledCancelEnded(plan *OrganizationBillingPlan, now time.Time) bool {
	return plan != nil && plan.CancelAtPeriodEnd && !periodStillOpen(plan.CurrentPeriodEnd, now)
}

// polarSubscriptionCannotCollectPayment is true when Polar cannot collect payment:
// a renewal failed and is retried, retries ended, or the customer paused it.
func polarSubscriptionCannotCollectPayment(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case PolarSubscriptionStatusPastDue, PolarSubscriptionStatusUnpaid, PolarSubscriptionStatusPaused:
		return true
	default:
		return false
	}
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

// shouldLapsePaymentBlockedCatalog is true for a Polar row on a paid catalog
// while Polar does not collect payment and no grace period applies: a paused
// or unpaid subscription, or a past-due row written before the failure start was
// recorded. ApplyPolarSubscription lapses new snapshots; this catches rows
// written before it did.
func (p *OrganizationBillingPlan) shouldLapsePaymentBlockedCatalog() bool {
	return p != nil && p.isPaidPlan() && p.PlanSource == BillingPlanSourcePolar &&
		polarSubscriptionCannotCollectPayment(p.PolarSubscriptionStatus) && !p.holdsPastDueGraceRow()
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

func lapseEndedPastDueGrace(tx *gorm.DB, plan *OrganizationBillingPlan, graceDays int) (*OrganizationBillingPlan, error) {
	var saved *OrganizationBillingPlan
	err := tx.Transaction(func(inner *gorm.DB) error {
		now := time.Now()
		nextPlan := planAfterPaidSubscriptionEnds(plan, now)
		result := inner.Model(&OrganizationBillingPlan{}).
			Where("organization_id = ? AND plan = ? AND plan_source = ?", plan.OrganizationID, plan.Plan, BillingPlanSourcePolar).
			Where("polar_subscription_status = ?", PolarSubscriptionStatusPastDue).
			Where("past_due_since IS NOT NULL AND past_due_since <= ?", now.Add(-pastDueGrace(graceDays))).
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

// lapsePaymentBlockedCatalog moves a payment-blocked Polar row off the paid
// catalog when no grace period applies. The subscription details and the
// purchased seats stay, so the seat lock and the billing page keep working
// while payment is blocked.
func lapsePaymentBlockedCatalog(tx *gorm.DB, plan *OrganizationBillingPlan) (*OrganizationBillingPlan, error) {
	var saved *OrganizationBillingPlan
	err := tx.Transaction(func(inner *gorm.DB) error {
		now := time.Now()
		nextPlan := planAfterPaidSubscriptionEnds(plan, now)
		result := inner.Model(&OrganizationBillingPlan{}).
			Where("organization_id = ? AND plan = ? AND plan_source = ?", plan.OrganizationID, plan.Plan, BillingPlanSourcePolar).
			Where("polar_subscription_status IN ?", []string{PolarSubscriptionStatusPastDue, PolarSubscriptionStatusPaused, PolarSubscriptionStatusUnpaid}).
			Where("past_due_since IS NULL").
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
			next.PastDueSince = existing.PastDueSince
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
				"past_due_since",
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
