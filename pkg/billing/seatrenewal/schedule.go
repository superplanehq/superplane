// Package seatrenewal keeps the next-period Polar seat count equal to the
// member count, so that Polar renews only occupied seats. An active and a
// past-due subscription both renew, so both get the scheduled count. A
// past-due subscription keeps its payment block: the scheduled count changes
// only the next period and grants no access.
//
// Requests from this package only schedule a seat count for the next period
// (proration_behavior next_period). They never add seats, never charge or
// credit the current period, and never change the renewal date.
package seatrenewal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/entitlements"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// Client is the part of the Polar API that scheduling uses.
type Client interface {
	GetSubscription(ctx context.Context, subscriptionID string) (*polar.SubscriptionData, error)
	ScheduleSubscriptionSeats(ctx context.Context, subscriptionID string, seats int) (*polar.SubscriptionData, error)
}

// Action is what Schedule did for an organization.
type Action string

const (
	// ActionSkipped means that the organization has no renewal seat count to
	// keep, or that another flow owns the next seat change.
	ActionSkipped Action = "skipped"
	// ActionInSync means that Polar already renews with the target. SuperPlane
	// can still have sent the target again to repair the stored record.
	ActionInSync Action = "in_sync"
	// ActionFixed means that SuperPlane scheduled the target in Polar.
	ActionFixed Action = "fixed"
)

// Reasons for ActionSkipped.
const (
	SkipNoRenewalQuantity          = "no_renewal_quantity"
	SkipNotPerSeat                 = "not_per_seat"
	SkipAdminPlan                  = "admin_plan"
	SkipNoSubscription             = "no_subscription"
	SkipSubscriptionNotActive      = "subscription_not_active"
	SkipSubscriptionPaused         = "subscription_paused"
	SkipSubscriptionCanceling      = "subscription_canceling"
	SkipSeatPurchaseInFlight       = "seat_purchase_in_flight"
	SkipPolarSubscriptionNotActive = "polar_subscription_not_active"
	SkipSubscriptionChanged        = "subscription_changed"
)

// Result describes one Schedule call. Fields that Schedule did not reach are
// nil.
type Result struct {
	OrganizationID uuid.UUID
	Action         Action
	SkipReason     string
	SubscriptionID string
	// Target is the seat count that Polar must renew with.
	Target *int
	// NextPeriodSeats is the seat count that Polar renewed with before this call.
	NextPeriodSeats *int
	// CurrentSeats is the seat count of the current Polar period.
	CurrentSeats *int
}

// LogFields are the stable log fields for a schedule result.
func (r Result) LogFields() log.Fields {
	fields := log.Fields{"organization_id": r.OrganizationID.String()}
	if r.SubscriptionID != "" {
		fields["polar_subscription_id"] = r.SubscriptionID
	}
	if r.Target != nil {
		fields["renewal_target_seats"] = *r.Target
	}
	if r.NextPeriodSeats != nil {
		fields["next_period_seats"] = *r.NextPeriodSeats
	}
	if r.CurrentSeats != nil {
		fields["current_seats"] = *r.CurrentSeats
	}
	return fields
}

// Scheduler keeps the next-period seat count of an organization in Polar
// equal to its renewal quantity.
type Scheduler struct {
	db     *gorm.DB
	client Client
	now    func() time.Time
}

func NewScheduler(db *gorm.DB, client Client, now func() time.Time) *Scheduler {
	return &Scheduler{db: db, client: client, now: now}
}

// Schedule compares the next-period Polar seat count with the renewal
// quantity of the organization and schedules the renewal quantity when they
// differ. A session advisory lock coordinates schedules across processes. The
// organization row lock is held only to read local state, never during HTTP.
func (s *Scheduler) Schedule(ctx context.Context, orgID uuid.UUID) (Result, error) {
	result := Result{OrganizationID: orgID}
	err := models.WithSeatRenewalScheduleLock(s.db.WithContext(ctx), orgID, func(conn *gorm.DB) error {
		locked := *s
		locked.db = conn
		var err error
		result, err = locked.schedule(ctx, orgID)
		return err
	})
	return result, err
}

func (s *Scheduler) schedule(ctx context.Context, orgID uuid.UUID) (Result, error) {
	// An abandoned purchase can stay in flight forever when nobody previews
	// or buys seats again, and the skip rules would keep its organization
	// unchecked. Settle or expire it from Polar before the skip rules.
	if err := polar.ReconcileSeatPurchase(ctx, s.db.WithContext(ctx), s.client, orgID); err != nil {
		return Result{OrganizationID: orgID}, err
	}

	// Polar is the source of truth for seats, status, and period. A lost
	// webhook leaves the stored plan behind, and the skip rules and the
	// target below must not run on that stale plan. Apply the verified
	// Polar state first, even for a plan that gets no scheduled count.
	subscriptionID, reason, err := s.reconcilableSubscription(ctx, orgID)
	if err != nil {
		return Result{OrganizationID: orgID}, err
	}
	if reason != "" {
		return skipped(Result{OrganizationID: orgID, SubscriptionID: subscriptionID}, reason), nil
	}
	snapshot, err := s.client.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return Result{OrganizationID: orgID, SubscriptionID: subscriptionID}, err
	}
	if err := s.applySnapshot(ctx, orgID, snapshot); err != nil {
		return Result{OrganizationID: orgID, SubscriptionID: subscriptionID}, fmt.Errorf("apply polar subscription %s: %w", subscriptionID, err)
	}

	// The skip rules and the target come from the plan after the apply. When
	// the apply ignored the snapshot because the stored plan is newer, they
	// come from the stored plan, and the next run reads Polar again.
	result, plan, err := s.loadLocalState(ctx, orgID)
	if err != nil || result.Action == ActionSkipped {
		return result, err
	}
	if result.SubscriptionID != snapshot.ID {
		// Another flow replaced the subscription after the read. The snapshot
		// describes the replaced one.
		return skipped(result, SkipSubscriptionChanged), nil
	}
	result.CurrentSeats = snapshot.Seats
	result.NextPeriodSeats = snapshot.NextPeriodSeats()
	if !renewsWithScheduledSeats(snapshot.Status, snapshot.CancelAtPeriodEnd) {
		// The stored plan renews, but the snapshot says that the
		// subscription ends or stops. The apply ignored it as older than
		// the stored plan. The next run reads Polar again.
		return skipped(result, SkipPolarSubscriptionNotActive), nil
	}
	if snapshot.Seats == nil || *snapshot.Seats < 1 {
		return result, fmt.Errorf("polar subscription %s reports no seats", result.SubscriptionID)
	}

	// A count above the current seats would add seats, so the target never
	// exceeds them.
	target := min(*result.Target, *snapshot.Seats)
	result.Target = &target
	inSync := result.NextPeriodSeats != nil && *result.NextPeriodSeats == target
	if inSync && scheduledRecordMatches(plan, target) {
		result.Action = ActionInSync
		return result, nil
	}

	// The target is sent even when Polar already renews with it but the
	// stored record is missing or stale: the record describes only counts
	// that SuperPlane sent, so a matching Polar count alone never repairs
	// it. The send is idempotent, and the next attempt retries the record
	// when this write fails.
	updated, err := s.client.ScheduleSubscriptionSeats(ctx, result.SubscriptionID, target)
	if err != nil {
		// A rejection leaves the record unchanged, so the next run sends
		// the target again.
		return result, fmt.Errorf("schedule %d seats on polar subscription %s: %w", target, result.SubscriptionID, err)
	}
	result.Action = ActionFixed
	if inSync {
		result.Action = ActionInSync
	}
	if err := s.applySnapshot(ctx, orgID, updated); err != nil {
		return result, fmt.Errorf("apply polar subscription %s: %w", result.SubscriptionID, err)
	}
	if err := plan.RecordScheduledRenewalSeats(s.db.WithContext(ctx), target, s.now()); err != nil {
		return result, fmt.Errorf("record scheduled renewal seats: %w", err)
	}
	return result, nil
}

// reconcilableSubscription reads the stored Polar subscription ID without a
// row lock. Only a plan that Polar manages has a subscription to reconcile.
// Otherwise, the reason names why not.
func (s *Scheduler) reconcilableSubscription(ctx context.Context, orgID uuid.UUID) (subscriptionID, skipReason string, err error) {
	plan, err := models.FindOrganizationBillingPlan(s.db.WithContext(ctx), orgID)
	if err != nil {
		return "", "", err
	}
	if plan == nil {
		return "", SkipNoSubscription, nil
	}
	if plan.PolarSubscriptionID != nil {
		subscriptionID = strings.TrimSpace(*plan.PolarSubscriptionID)
	}
	switch {
	case plan.PlanSource == models.BillingPlanSourceAdmin:
		return subscriptionID, SkipAdminPlan, nil
	case plan.PlanSource != models.BillingPlanSourcePolar || subscriptionID == "":
		return subscriptionID, SkipNoSubscription, nil
	default:
		return subscriptionID, "", nil
	}
}

// applySnapshot applies a Polar response through the subscription apply path
// of the webhooks. That path checks the identity, rejects a response that is
// older than the stored plan, and rechecks the seat capacity. The scheduler
// holds a session connection, not a transaction, so the apply makes its own
// Polar reads outside any row lock.
func (s *Scheduler) applySnapshot(ctx context.Context, orgID uuid.UUID, snapshot *polar.SubscriptionData) error {
	if snapshot == nil || strings.TrimSpace(snapshot.ID) == "" {
		log.WithField("organization_id", orgID.String()).Debug("polar returned no subscription to apply")
		return nil
	}
	return polar.ApplyOrganizationSubscription(ctx, s.db.WithContext(ctx), orgID, *snapshot)
}

// loadLocalState reads the plan and the renewal quantity under the
// organization lock and applies the skip rules.
func (s *Scheduler) loadLocalState(ctx context.Context, orgID uuid.UUID) (Result, *models.OrganizationBillingPlan, error) {
	result := Result{OrganizationID: orgID}
	var plan *models.OrganizationBillingPlan
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockOrganization(tx, orgID); err != nil {
			return err
		}
		var err error
		plan, err = models.ResolveOrganizationBillingPlan(tx, orgID)
		if err != nil {
			return err
		}
		if plan.PolarSubscriptionID != nil {
			result.SubscriptionID = strings.TrimSpace(*plan.PolarSubscriptionID)
		}
		if reason := planSkipReason(plan); reason != "" {
			result = skipped(result, reason)
			return nil
		}

		target, err := entitlements.ResolveRenewalSeats(tx, orgID, plan)
		if errors.Is(err, entitlements.ErrInvalidPaidSeats) {
			return fmt.Errorf("plan of polar subscription %s reports no paid seats: %w", result.SubscriptionID, err)
		}
		if err != nil {
			return err
		}
		if target == nil {
			result = skipped(result, SkipNoRenewalQuantity)
			return nil
		}
		result.Target = target

		// A purchase inside its confirmation window owns the next seat
		// change: it replaces any scheduled count when it settles and
		// schedules the renewal quantity again after it. Schedule already
		// reconciled an older purchase, so only a recent one still skips.
		inFlight, err := models.FindInFlightSeatPurchase(tx, orgID)
		if err != nil {
			return err
		}
		if inFlight != nil {
			result = skipped(result, SkipSeatPurchaseInFlight)
		}
		return nil
	})
	return result, plan, err
}

// scheduledRecordMatches reports that the plan already records a send of the
// target. Only then does a matching Polar count prove nothing is left to do.
func scheduledRecordMatches(plan *models.OrganizationBillingPlan, target int) bool {
	return plan.RenewalSeatsScheduled != nil && *plan.RenewalSeatsScheduled == target
}

// planSkipReason names why a plan gets no scheduled count. Only a seat-based
// Polar subscription that renews gets one: an active subscription, or a
// past-due one that Polar still retries. The skip does not depend on the
// catalog, because a payment failure can move the row to a fallback catalog
// that is not per seat.
func planSkipReason(plan *models.OrganizationBillingPlan) string {
	switch {
	case !entitlements.IsSeatBased(plan):
		return SkipNotPerSeat
	case plan.PlanSource == models.BillingPlanSourceAdmin:
		return SkipAdminPlan
	case plan.PlanSource != models.BillingPlanSourcePolar || plan.PolarSubscriptionID == nil || strings.TrimSpace(*plan.PolarSubscriptionID) == "":
		return SkipNoSubscription
	case strings.EqualFold(strings.TrimSpace(plan.PolarSubscriptionStatus), models.PolarSubscriptionStatusPaused):
		return SkipSubscriptionPaused
	case !renewingStatus(plan.PolarSubscriptionStatus):
		return SkipSubscriptionNotActive
	case plan.CancelAtPeriodEnd:
		return SkipSubscriptionCanceling
	default:
		return ""
	}
}

// renewsWithScheduledSeats reports a Polar subscription that renews at the
// period end and is not set to cancel.
func renewsWithScheduledSeats(status string, cancelAtPeriodEnd bool) bool {
	return renewingStatus(status) && !cancelAtPeriodEnd
}

// renewingStatus is true for an active subscription and for a past-due one,
// because Polar keeps a past-due subscription and renews it while it retries
// the payment.
func renewingStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case models.PolarSubscriptionStatusActive, models.PolarSubscriptionStatusPastDue:
		return true
	default:
		return false
	}
}

func skipped(result Result, reason string) Result {
	result.Action = ActionSkipped
	result.SkipReason = reason
	return result
}
