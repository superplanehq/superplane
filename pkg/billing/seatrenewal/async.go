package seatrenewal

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

// retryDelays are the waits between the attempts of a background schedule:
// three attempts over about ten seconds.
var retryDelays = []time.Duration{2 * time.Second, 6 * time.Second}

var background sync.WaitGroup

// ScheduleAsync schedules the renewal seat count after a membership, seat,
// account, or subscription lifecycle change commits. It returns at once and
// never fails the caller: the schedule runs in the background, outside the
// transaction and the organization lock. Transient Polar failures get a
// short retry. The seat renewal sync worker fixes what is left.
func ScheduleAsync(ctx context.Context, orgID uuid.UUID) {
	if !models.BillingEnabled() || !polar.Configured() {
		return
	}
	ctx = context.WithoutCancel(ctx)
	scheduler := NewScheduler(database.DB(ctx), polar.NewClientFromEnv(), time.Now)
	background.Go(func() {
		scheduler.scheduleWithRetry(ctx, orgID, retryDelays)
	})
}

// ScheduleAfterSeatPurchaseAsync restores the reduction that an applied
// purchase clears in Polar. Call it after a purchase reconciliation commits.
// It schedules only when the latest seat purchase succeeded. Repeated calls
// are safe: Schedule verifies the current target and provider state before
// any update.
func ScheduleAfterSeatPurchaseAsync(ctx context.Context, orgID uuid.UUID) {
	if !models.BillingEnabled() || !polar.Configured() {
		return
	}
	ctx = context.WithoutCancel(ctx)
	purchase, err := models.FindLatestSeatPurchase(database.DB(ctx), orgID)
	if err != nil {
		log.WithError(err).WithField("organization_id", orgID.String()).Warn("could not check the settled seat purchase for renewal scheduling")
		return
	}
	if purchase != nil && purchase.Status == models.SeatPurchaseStatusSucceeded {
		ScheduleAsync(ctx, orgID)
	}
}

// Wait blocks until every schedule that ScheduleAsync started has ended.
func Wait() {
	background.Wait()
}

func (s *Scheduler) scheduleWithRetry(ctx context.Context, orgID uuid.UUID, delays []time.Duration) {
	for attempt := 0; ; attempt++ {
		result, err := s.Schedule(ctx, orgID)
		if err == nil {
			if result.Action == ActionFixed {
				log.WithFields(result.LogFields()).Info("scheduled the renewal seat count in Polar")
			}
			return
		}
		if !isRetryable(err) || attempt >= len(delays) {
			log.WithError(err).WithFields(result.LogFields()).
				Warn("could not schedule the renewal seat count in Polar; the seat renewal sync worker retries")
			return
		}
		time.Sleep(delays[attempt])
	}
}

// isRetryable reports a Polar failure that can pass on its own: a rate
// limit, a concurrent subscription update, or a network or server error.
// Scheduling a seat count is idempotent, so a request with an unknown
// outcome is safe to send again.
func isRetryable(err error) bool {
	return polar.IsRateLimited(err) || polar.IsSubscriptionLocked(err) || polar.IsOutcomeUnknown(err)
}
