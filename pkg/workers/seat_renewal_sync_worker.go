package workers

import (
	"context"
	"time"

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
)

// SeatRenewalSyncWorker fixes organizations whose next-period Polar seat
// count differs from the member count. It covers a Polar outage, lost live
// updates, races between a purchase and a removal, and membership changes
// that do not schedule the renewal seat count, such as account deletion.
type SeatRenewalSyncWorker struct {
	scheduler *seatrenewal.Scheduler
	now       func() time.Time
	logger    *log.Entry
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

// tick checks every candidate organization, the one that renews first first.
func (w *SeatRenewalSyncWorker) tick(ctx context.Context) {
	if ctx.Err() != nil || !models.BillingEnabled() || !polar.Configured() {
		return
	}

	plans, err := models.ListSeatRenewalSyncPlans(database.Conn().WithContext(ctx))
	if err != nil {
		w.logger.WithError(err).Error("could not list organizations for the seat renewal sync")
		return
	}
	for _, plan := range plans {
		if ctx.Err() != nil {
			return
		}
		result, err := w.scheduler.Schedule(ctx, plan.OrganizationID)
		if err != nil {
			w.logFailure(plan, result, err)
			if polar.IsRateLimited(err) {
				// The next tick starts again with the organizations that renew first.
				return
			}
			continue
		}
		if result.Action == seatrenewal.ActionFixed {
			w.logger.WithFields(result.LogFields()).Warn(seatRenewalMismatchFixedKey)
		}
	}
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
