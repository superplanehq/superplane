package workers

import (
	"context"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/billing/seatrenewal"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	seatRenewalSyncEvery = 5 * time.Minute

	// seatRenewalUrgentWindow is how close to renewal a failed fix becomes
	// an error: Polar can renew with the wrong seat count soon.
	seatRenewalUrgentWindow = 24 * time.Hour

	seatRenewalMismatchFixedKey = "billing.seat_renewal_mismatch_fixed"
	seatRenewalSyncFailedKey    = "billing.seat_renewal_sync_failed"

	// seatPurchaseUnresolvedAfter is how long a purchase can stay pending or
	// unknown before the worker reports it. It is much longer than the
	// purchase confirmation window, so only a purchase that no Polar
	// snapshot can settle any more reaches it.
	seatPurchaseUnresolvedAfter = 1 * time.Hour

	seatPurchaseUnresolvedKey       = "billing.seat_purchase_unresolved"
	seatRenewalBlockedByPurchaseKey = "billing.seat_renewal_blocked_by_purchase"
)

// SeatRenewalSyncWorker reconciles the stored plan of each organization with
// its Polar subscription and fixes the next-period Polar seat count when it
// differs from the member count. The reconciliation repairs lost subscription
// webhooks: a missed renewal, payment failure, or recovery. It covers
// organizations whose subscription lapsed, and these get no scheduled seat
// count. The seat count fix covers a Polar outage, races between a purchase
// and a removal, and membership changes that do not schedule the renewal seat
// count, such as account deletion.
type SeatRenewalSyncWorker struct {
	scheduler *seatrenewal.Scheduler
	now       func() time.Time
	logger    *log.Entry
	// resumeAfter is the organization where a rate limit stopped the last
	// run. The next run continues there instead of spending the whole Polar
	// request budget on the same organizations that renew first.
	resumeAfter uuid.UUID
}

func NewSeatRenewalSyncWorker() *SeatRenewalSyncWorker {
	w := &SeatRenewalSyncWorker{
		now:    time.Now,
		logger: log.WithFields(log.Fields{"worker": "SeatRenewalSyncWorker"}),
	}
	w.scheduler = seatrenewal.NewScheduler(database.Conn(), polar.NewClientFromEnv(), func() time.Time { return w.now() })
	return w
}

func (w *SeatRenewalSyncWorker) Start(ctx context.Context) {
	w.tick(ctx)

	ticker := time.NewTicker(seatRenewalSyncEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

// tick reports unresolved seat purchases and then reconciles every candidate
// organization, the one that renews first first.
func (w *SeatRenewalSyncWorker) tick(ctx context.Context) {
	if ctx.Err() != nil || !models.BillingEnabled() || !polar.Configured() {
		return
	}

	// The report does not depend on the candidate list or on the schedule
	// results, so a stuck purchase is visible even for an organization that
	// the list excludes or that every schedule skips.
	w.reportUnresolvedSeatPurchases(ctx)

	plans, err := models.ListSeatRenewalSyncPlans(database.Conn().WithContext(ctx))
	if err != nil {
		w.logger.WithError(err).Error("could not list organizations for the seat renewal sync")
		return
	}
	for _, plan := range w.resumeFrom(plans) {
		if ctx.Err() != nil {
			return
		}
		result, err := w.scheduler.Schedule(ctx, plan.OrganizationID)
		if err != nil {
			w.logFailure(plan, result, err)
			if polar.IsRateLimited(err) {
				// The next run continues at this organization, so a request
				// budget smaller than the list cannot starve the
				// organizations after it.
				w.resumeAfter = plan.OrganizationID
				return
			}
			continue
		}
		switch {
		case result.Action == seatrenewal.ActionFixed:
			w.logger.WithFields(result.LogFields()).Warn(seatRenewalMismatchFixedKey)
		case result.SkipReason == seatrenewal.SkipSeatPurchaseInFlight:
			w.logger.WithFields(result.LogFields()).Warn(seatRenewalBlockedByPurchaseKey)
		}
	}
	// A completed run clears the position, so the next run starts again
	// with the organizations that renew first.
	w.resumeAfter = uuid.Nil
}

// reportUnresolvedSeatPurchases writes one error line for each purchase that
// stayed pending or unknown longer than seatPurchaseUnresolvedAfter. The line
// repeats on every run while the purchase stays unresolved. The alert policy
// suppresses duplicates, so the worker keeps no state and the signal
// survives restarts. The report changes no purchase.
func (w *SeatRenewalSyncWorker) reportUnresolvedSeatPurchases(ctx context.Context) {
	db := database.Conn().WithContext(ctx)
	now := w.now()
	purchases, err := models.ListUnresolvedSeatPurchasesCreatedBefore(db, now.Add(-seatPurchaseUnresolvedAfter))
	if err != nil {
		w.logger.WithError(err).Error("could not list unresolved seat purchases")
		return
	}
	for i := range purchases {
		purchase := &purchases[i]
		plan, err := models.FindOrganizationBillingPlan(db, purchase.OrganizationID)
		if err != nil {
			w.logger.WithError(err).WithFields(seatrenewal.SeatPurchaseLogFields(purchase)).Error("could not read the plan of an unresolved seat purchase")
			continue
		}
		w.logger.WithFields(unresolvedSeatPurchaseFields(purchase, plan, now)).Error(seatPurchaseUnresolvedKey)
	}
}

func unresolvedSeatPurchaseFields(purchase *models.SeatPurchase, plan *models.OrganizationBillingPlan, now time.Time) log.Fields {
	fields := seatrenewal.SeatPurchaseLogFields(purchase)
	fields["age_seconds"] = int64(now.Sub(purchase.CreatedAt) / time.Second)
	fields["renewal_impact"] = seatrenewal.RenewalImpact(purchase, plan, now)
	if plan == nil {
		return fields
	}
	if plan.PolarSubscriptionID != nil {
		fields["current_polar_subscription_id"] = *plan.PolarSubscriptionID
	}
	if plan.CurrentPeriodStart != nil {
		fields["current_period_start"] = plan.CurrentPeriodStart.UTC()
	}
	return fields
}

// resumeFrom starts the list at the organization where a rate limit stopped
// the last run. It starts from the top again when that organization left the
// candidate list, for example after its renewal passed.
func (w *SeatRenewalSyncWorker) resumeFrom(plans []models.OrganizationBillingPlan) []models.OrganizationBillingPlan {
	if w.resumeAfter == uuid.Nil {
		return plans
	}
	for i, plan := range plans {
		if plan.OrganizationID == w.resumeAfter {
			return plans[i:]
		}
	}
	return plans
}

func (w *SeatRenewalSyncWorker) logFailure(plan models.OrganizationBillingPlan, result seatrenewal.Result, err error) {
	if result.SubscriptionID == "" && plan.PolarSubscriptionID != nil {
		result.SubscriptionID = *plan.PolarSubscriptionID
	}
	entry := w.logger.WithError(err).WithFields(result.LogFields())
	if plan.CurrentPeriodEnd != nil && plan.CurrentPeriodEnd.Sub(w.now()) <= seatRenewalUrgentWindow {
		entry.Error(seatRenewalSyncFailedKey)
		return
	}
	entry.Warn("could not sync the renewal seat count; the next run retries")
}
