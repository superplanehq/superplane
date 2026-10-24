// Package seatrenewal keeps the next-period Polar seat count equal to the
// member count, so that Polar renews only occupied seats.
//
// Requests from this package only schedule a seat count for the next period
// (proration_behavior next_period). They never add seats, never charge or
// credit the current period, and never change the renewal date.
package seatrenewal

import (
	"context"
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
	// ActionInSync means that Polar already renews with the target.
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
	SkipSubscriptionCanceling      = "subscription_canceling"
	SkipSeatPurchaseInFlight       = "seat_purchase_in_flight"
	SkipPolarSubscriptionNotActive = "polar_subscription_not_active"
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
// differ. The organization lock is held only to read local state, never
// during a Polar call.
func (s *Scheduler) Schedule(ctx context.Context, orgID uuid.UUID) (Result, error) {
	result, plan, err := s.loadLocalState(ctx, orgID)
	if err != nil || result.Action == ActionSkipped {
		return result, err
	}

	snapshot, err := s.client.GetSubscription(ctx, result.SubscriptionID)
	if err != nil {
		return result, err
	}
	result.CurrentSeats = snapshot.Seats
	result.NextPeriodSeats = snapshot.NextPeriodSeats()
	if !strings.EqualFold(strings.TrimSpace(snapshot.Status), models.PolarSubscriptionStatusActive) || snapshot.CancelAtPeriodEnd {
		// The stored plan is behind Polar. The next subscription webhook
		// updates it.
		return skipped(result, SkipPolarSubscriptionNotActive), nil
	}
	if snapshot.Seats == nil || *snapshot.Seats < 1 {
		return result, fmt.Errorf("polar subscription %s reports no seats", result.SubscriptionID)
	}

	// A count above the current seats would add seats, so the target never
	// exceeds them.
	target := min(*result.Target, *snapshot.Seats)
	result.Target = &target
	if result.NextPeriodSeats != nil && *result.NextPeriodSeats == target {
		result.Action = ActionInSync
		return result, nil
	}

	if _, err := s.client.ScheduleSubscriptionSeats(ctx, result.SubscriptionID, target); err != nil {
		return result, err
	}
	result.Action = ActionFixed
	if err := plan.RecordScheduledRenewalSeats(s.db.WithContext(ctx), target, s.now()); err != nil {
		return result, fmt.Errorf("record scheduled renewal seats: %w", err)
	}
	return result, nil
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

		seats, err := entitlements.ResolveSeatsForPlan(tx, orgID, plan)
		if err != nil {
			return err
		}
		if seats.RenewalQuantity == nil {
			result = skipped(result, SkipNoRenewalQuantity)
			return nil
		}
		result.Target = seats.RenewalQuantity

		// The purchase replaces any scheduled count when it settles and
		// schedules the renewal quantity again after it.
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

func planSkipReason(plan *models.OrganizationBillingPlan) string {
	switch {
	case !plan.Catalog.PerSeat:
		return SkipNotPerSeat
	case plan.PlanSource == models.BillingPlanSourceAdmin:
		return SkipAdminPlan
	case plan.PolarSubscriptionID == nil || strings.TrimSpace(*plan.PolarSubscriptionID) == "":
		return SkipNoSubscription
	case plan.PolarSubscriptionStatus != models.PolarSubscriptionStatusActive:
		return SkipSubscriptionNotActive
	case plan.CancelAtPeriodEnd:
		return SkipSubscriptionCanceling
	default:
		return ""
	}
}

func skipped(result Result, reason string) Result {
	result.Action = ActionSkipped
	result.SkipReason = reason
	return result
}
