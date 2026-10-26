package seatrenewal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/entitlements"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/polarfake"
	"gorm.io/gorm"
)

type scheduleFixture struct {
	t         *testing.T
	r         *support.ResourceRegistry
	polar     *polarfake.Server
	sub       polarfake.Subscription
	scheduler *Scheduler
	now       time.Time
}

// newScheduleFixture is an organization with the owner and members-1 other
// members on an active per-seat Polar subscription with the given seats.
func newScheduleFixture(t *testing.T, seats, members int) *scheduleFixture {
	t.Helper()
	r := support.Setup(t)
	t.Cleanup(r.Close)
	t.Setenv(models.BillingEnabledEnv, "yes")
	f := &scheduleFixture{t: t, r: r, polar: polarfake.NewServer(t), now: time.Now().UTC().Truncate(time.Second)}
	f.sub = f.polar.AddSeatPlan(database.Conn(), r.Organization.ID, seats, f.now.Add(10*24*time.Hour))
	for range members - 1 {
		support.CreateUser(t, r, r.Organization.ID)
	}
	f.scheduler = NewScheduler(database.Conn(), polar.NewClientFromEnv(), func() time.Time { return f.now })
	return f
}

func (f *scheduleFixture) schedule() Result {
	f.t.Helper()
	result, err := f.scheduler.Schedule(context.Background(), f.r.Organization.ID)
	require.NoError(f.t, err)
	return result
}

func (f *scheduleFixture) updatePlan(values map[string]any) {
	f.t.Helper()
	require.NoError(f.t, database.Conn().Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", f.r.Organization.ID).Updates(values).Error)
}

func (f *scheduleFixture) plan() *models.OrganizationBillingPlan {
	f.t.Helper()
	plan, err := models.FindOrganizationBillingPlan(database.Conn(), f.r.Organization.ID)
	require.NoError(f.t, err)
	return plan
}

func nextPeriod(seats int) map[string]any {
	return map[string]any{"seats": float64(seats), "proration_behavior": "next_period"}
}

func Test__ScheduleSetsTheNextPeriodSeatsToTheMemberCount(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)

	result := f.schedule()

	assert.Equal(t, ActionFixed, result.Action)
	assert.Equal(t, new(9), result.Target)
	assert.Equal(t, new(10), result.NextPeriodSeats)
	assert.Equal(t, new(10), result.CurrentSeats)
	require.Len(t, f.polar.Patches(), 1)
	assert.Equal(t, nextPeriod(9), f.polar.Patches()[0].Body)
	sub := f.polar.Subscription(f.sub.ID)
	assert.Equal(t, 10, sub.Seats, "the current period keeps its seats")
	assert.Equal(t, new(9), sub.PendingSeats)

	plan := f.plan()
	assert.Equal(t, new(9), plan.RenewalSeatsScheduled)
	require.NotNil(t, plan.RenewalSeatsScheduledAt)
	assert.True(t, plan.RenewalSeatsScheduledAt.Equal(f.now))
	assert.Equal(t, new(10), plan.PurchasedSeats)
	assert.True(t, plan.CurrentPeriodEnd.Equal(f.sub.PeriodEnd))
}

type pausedSubscriptionClient struct {
	Client
	once   sync.Once
	read   chan struct{}
	resume chan struct{}
}

func (c *pausedSubscriptionClient) GetSubscription(ctx context.Context, id string) (*polar.SubscriptionData, error) {
	snapshot, err := c.Client.GetSubscription(ctx, id)
	c.once.Do(func() {
		close(c.read)
		<-c.resume
	})
	return snapshot, err
}

func Test__ConcurrentSchedulesCannotLeaveAnOlderRenewalTarget(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	client := &pausedSubscriptionClient{Client: polar.NewClientFromEnv(), read: make(chan struct{}), resume: make(chan struct{})}
	first := NewScheduler(database.Conn(), client, time.Now)
	firstDone := make(chan error, 1)
	go func() {
		_, err := first.Schedule(t.Context(), f.r.Organization.ID)
		firstDone <- err
		close(firstDone)
	}()
	<-client.read // A captured nine members and is paused outside the row lock.
	var resume sync.Once
	t.Cleanup(func() { resume.Do(func() { close(client.resume) }); require.NoError(t, <-firstDone) })

	// Membership can still change while A is making an external request.
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	require.NoError(t, database.Conn().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockOrganization(tx, f.r.Organization.ID); err != nil {
			return err
		}
		return tx.Model(&models.User{}).Where("organization_id = ? AND is_owner = false", f.r.Organization.ID).
			Where("id = (SELECT id FROM users WHERE organization_id = ? AND is_owner = false LIMIT 1)", f.r.Organization.ID).
			Update("deleted_at", time.Now()).Error
	}))

	second := NewScheduler(database.Conn(), polar.NewClientFromEnv(), time.Now)
	secondDone := make(chan error, 1)
	go func() {
		_, err := second.Schedule(t.Context(), f.r.Organization.ID)
		secondDone <- err
	}()
	// Without coordination, B completes with eight before A resumes. With
	// coordination, B waits in PostgreSQL and reads its target after A ends.
	completed := false
	require.Eventually(t, func() bool {
		select {
		case err := <-secondDone:
			require.NoError(t, err)
			completed = true
			return true
		default:
			var waiting bool
			require.NoError(t, database.Conn().Raw("SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE wait_event = 'advisory' AND query LIKE '%pg_advisory_lock%')").Scan(&waiting).Error)
			return waiting
		}
	}, 5*time.Second, 10*time.Millisecond)
	resume.Do(func() { close(client.resume) })
	if !completed {
		require.NoError(t, <-secondDone)
	}
	require.NoError(t, <-firstDone)
	assert.Equal(t, new(8), f.polar.Subscription(f.sub.ID).PendingSeats)
	assert.Equal(t, new(8), f.plan().RenewalSeatsScheduled)
}

func Test__CanceledScheduleReleasesOrganizationCoordination(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	client := &pausedSubscriptionClient{Client: polar.NewClientFromEnv(), read: make(chan struct{}), resume: make(chan struct{})}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := NewScheduler(database.Conn(), client, time.Now).Schedule(ctx, f.r.Organization.ID)
		done <- err
	}()
	<-client.read
	cancel()
	close(client.resume)
	require.ErrorIs(t, <-done, context.Canceled)

	retryCtx, retryCancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer retryCancel()
	result, err := f.scheduler.Schedule(retryCtx, f.r.Organization.ID)
	require.NoError(t, err, "a canceled request must release its session lock")
	assert.Equal(t, ActionFixed, result.Action)
	assert.Equal(t, new(9), f.polar.Subscription(f.sub.ID).PendingSeats)
}

func Test__SlowPolarSchedulesLeaveConnectionsForForegroundRequests(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	pool, err := database.Conn().DB()
	require.NoError(t, err)
	previousLimit := pool.Stats().MaxOpenConnections
	pool.SetMaxOpenConns(5)
	t.Cleanup(func() { pool.SetMaxOpenConns(previousLimit) })

	client := &pausedSubscriptionClient{Client: f.scheduler.client, read: make(chan struct{}), resume: make(chan struct{})}
	firstDone := make(chan error, 1)
	go func() {
		_, err := NewScheduler(database.Conn(), client, time.Now).Schedule(t.Context(), f.r.Organization.ID)
		firstDone <- err
		close(firstDone)
	}()
	<-client.read
	var resume sync.Once
	t.Cleanup(func() { resume.Do(func() { close(client.resume) }); require.NoError(t, <-firstDone) })

	// Five more schedules for the same organization must wait without taking
	// every connection. Their deadlines also exercise cancellation in the queue.
	queuedCtx, cancelQueued := context.WithTimeout(t.Context(), time.Second)
	defer cancelQueued()
	queuedDone := make(chan error, 5)
	for range 5 {
		go func() {
			_, err := f.scheduler.Schedule(queuedCtx, f.r.Organization.ID)
			queuedDone <- err
		}()
	}

	peakConnections := 0
	foregroundFailures := 0
	for remaining := 5; remaining > 0; {
		select {
		case err := <-queuedDone:
			assert.ErrorIs(t, err, context.DeadlineExceeded)
			remaining--
		default:
			peakConnections = max(peakConnections, pool.Stats().InUse)
			ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
			if err := pool.PingContext(ctx); err != nil {
				foregroundFailures++
			}
			cancel()
			time.Sleep(time.Millisecond)
		}
	}
	assert.Zero(t, foregroundFailures, "foreground database requests must finish while Polar is paused")
	assert.LessOrEqual(t, peakConnections, 2, "renewal schedules must leave capacity in the five-connection pool")

	resume.Do(func() { close(client.resume) })
	require.NoError(t, <-firstDone)
	result := f.schedule()
	assert.Equal(t, ActionInSync, result.Action, "canceled waiters must release their scheduling capacity")
	assert.Equal(t, new(9), f.polar.Subscription(f.sub.ID).PendingSeats)
}

func Test__ScheduleRecordsTheCountPolarAlreadyRenews(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) { sub.PendingSeats = new(9) })

	// Polar already renews the member count, but nothing records a send of
	// it. The record describes only counts SuperPlane sent, so the count is
	// sent once and recorded. The send changes nothing in Polar.
	result := f.schedule()

	assert.Equal(t, ActionInSync, result.Action)
	assert.Equal(t, []polarfake.Request{{Method: http.MethodPatch, SubscriptionID: f.sub.ID, Body: nextPeriod(9)}}, f.polar.Patches())
	assert.Equal(t, new(9), f.polar.Subscription(f.sub.ID).PendingSeats)
	plan := f.plan()
	assert.Equal(t, new(9), plan.RenewalSeatsScheduled)
	assert.True(t, plan.RenewalSeatsScheduledAt.Equal(f.now))

	result = f.schedule()

	assert.Equal(t, ActionInSync, result.Action)
	assert.Len(t, f.polar.Patches(), 1, "the stored record matches the target now")
}

func Test__ScheduleSendsAgainWhenTheStoredRecordIsStale(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) { sub.PendingSeats = new(9) })
	require.NoError(t, f.plan().RecordScheduledRenewalSeats(database.Conn(), 8, f.now.Add(-time.Hour)))

	// Polar renews the target, but the record names a count SuperPlane sent
	// earlier. A matching Polar count alone never proves who set it, so the
	// target is sent again and the record is repaired.
	result := f.schedule()

	assert.Equal(t, ActionInSync, result.Action)
	assert.Equal(t, []polarfake.Request{{Method: http.MethodPatch, SubscriptionID: f.sub.ID, Body: nextPeriod(9)}}, f.polar.Patches())
	plan := f.plan()
	assert.Equal(t, new(9), plan.RenewalSeatsScheduled)
	assert.True(t, plan.RenewalSeatsScheduledAt.Equal(f.now))
}

func Test__ScheduleBackToTheCurrentSeatsClearsTheScheduledReduction(t *testing.T) {
	f := newScheduleFixture(t, 10, 10)
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) { sub.PendingSeats = new(9) })

	result := f.schedule()

	assert.Equal(t, ActionFixed, result.Action)
	assert.Equal(t, []polarfake.Request{{Method: http.MethodPatch, SubscriptionID: f.sub.ID, Body: nextPeriod(10)}}, f.polar.Patches())
	assert.Nil(t, f.polar.Subscription(f.sub.ID).PendingSeats)
}

func Test__ScheduleNeverSendsMoreSeatsThanTheCurrentPeriod(t *testing.T) {
	// An outside change lowered the seats below the member count.
	f := newScheduleFixture(t, 2, 3)
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) { sub.PendingSeats = new(1) })

	result := f.schedule()

	assert.Equal(t, new(2), result.Target)
	assert.Equal(t, []polarfake.Request{{Method: http.MethodPatch, SubscriptionID: f.sub.ID, Body: nextPeriod(2)}}, f.polar.Patches())

	assert.Equal(t, ActionInSync, f.schedule().Action)
	assert.Len(t, f.polar.Patches(), 1)
}

func Test__ScheduleSkipsOrganizationsWithoutARenewalSeatCountToKeep(t *testing.T) {
	tests := []struct {
		name   string
		change func(f *scheduleFixture)
		reason string
		// reconciles is true when the plan has a Polar subscription to read.
		reconciles bool
	}{
		{name: "plan is not per seat", reason: SkipNotPerSeat, reconciles: true, change: func(f *scheduleFixture) {
			product := "prod_flat_" + uuid.NewString()
			catalog := models.BillingPlan{Key: "flat_" + uuid.NewString()[:8], Name: "Flat", Category: "paid", Visibility: "private", PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
			require.NoError(t, models.SaveBillingPlan(database.Conn(), &catalog))
			f.updatePlan(map[string]any{"plan_id": catalog.ID, "plan": catalog.Key, "purchased_seats": nil})
			f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) {
				sub.ProductID = product
				sub.NoSeats = true
			})
		}},
		{name: "admin plan", reason: SkipAdminPlan, change: func(f *scheduleFixture) {
			f.updatePlan(map[string]any{"plan_source": models.BillingPlanSourceAdmin})
		}},
		{name: "no subscription", reason: SkipNoSubscription, change: func(f *scheduleFixture) {
			f.updatePlan(map[string]any{"polar_subscription_id": nil})
		}},
		{name: "billing is off", reason: SkipNoRenewalQuantity, reconciles: true, change: func(f *scheduleFixture) {
			f.t.Setenv(models.BillingEnabledEnv, "no")
		}},
		{name: "seat purchase in flight", reason: SkipSeatPurchaseInFlight, reconciles: true, change: func(f *scheduleFixture) {
			purchase := models.NewSeatPurchase(f.r.Organization.ID, nil, f.sub.ID, 10, 11, nil)
			require.NoError(t, purchase.Create(database.Conn()))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newScheduleFixture(t, 10, 9)
			tt.change(f)

			result := f.schedule()

			assert.Equal(t, ActionSkipped, result.Action)
			assert.Equal(t, tt.reason, result.SkipReason)
			assert.Empty(t, f.polar.Patches())
			if !tt.reconciles {
				assert.Empty(t, f.polar.Requests(), "a plan that Polar does not manage needs no Polar request")
			}
		})
	}
}

func Test__ScheduleStoresPolarStateOfAPlanThatGetsNoScheduledCount(t *testing.T) {
	tests := []struct {
		name              string
		status            string
		cancelAtPeriodEnd bool
		reason            string
	}{
		{name: "paused", status: models.PolarSubscriptionStatusPaused, reason: SkipSubscriptionPaused},
		{name: "unpaid", status: models.PolarSubscriptionStatusUnpaid, reason: SkipSubscriptionNotActive},
		{name: "canceled", status: models.PolarSubscriptionStatusCanceled, reason: SkipSubscriptionNotActive},
		{name: "revoked", status: models.PolarSubscriptionStatusRevoked, reason: SkipSubscriptionNotActive},
		{name: "set to cancel", status: models.PolarSubscriptionStatusActive, cancelAtPeriodEnd: true, reason: SkipSubscriptionCanceling},
		{name: "past due and set to cancel", status: models.PolarSubscriptionStatusPastDue, cancelAtPeriodEnd: true, reason: SkipSubscriptionCanceling},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newScheduleFixture(t, 10, 9)
			f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) {
				sub.Status = tt.status
				sub.CancelAtPeriodEnd = tt.cancelAtPeriodEnd
			})

			result := f.schedule()

			assert.Equal(t, ActionSkipped, result.Action)
			assert.Equal(t, tt.reason, result.SkipReason)
			assert.Empty(t, f.polar.Patches())
			plan := f.plan()
			assert.Equal(t, tt.status, plan.PolarSubscriptionStatus, "the plan stores the Polar state that blocked the schedule")
			assert.Equal(t, tt.cancelAtPeriodEnd, plan.CancelAtPeriodEnd)
			assert.Nil(t, plan.RenewalSeatsScheduled)
		})
	}
}

func Test__ScheduleSkipsAnOlderSnapshotThatDoesNotRenew(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	f.updatePlan(map[string]any{"polar_subscription_status": models.PolarSubscriptionStatusPastDue, "polar_modified_at": f.now})
	// Polar answers with a snapshot that is older than the stored plan, so
	// the apply ignores it. Its status still blocks the schedule.
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) {
		sub.Status = models.PolarSubscriptionStatusPaused
		sub.ModifiedAt = new(f.now.Add(-time.Hour))
	})

	result := f.schedule()

	assert.Equal(t, ActionSkipped, result.Action)
	assert.Equal(t, SkipPolarSubscriptionNotActive, result.SkipReason)
	assert.Empty(t, f.polar.Patches())
}

// pastDue puts the subscription in the past-due state of Polar. The grace
// period is the number of days that Team features stay on.
func (f *scheduleFixture) pastDue(graceDays int) {
	f.t.Helper()
	f.setGraceDays(graceDays)
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) { sub.Status = models.PolarSubscriptionStatusPastDue })
}

func (f *scheduleFixture) setGraceDays(days int) {
	f.t.Helper()
	set := func(days int) error {
		return models.ApplyInstallationMetadata(database.Conn(), func(metadata *models.InstallationMetadata) {
			metadata.PastDueGraceDays = days
		})
	}
	require.NoError(f.t, set(days))
	f.t.Cleanup(func() { require.NoError(f.t, set(0)) })
}

// removeMember removes one member that is not the owner.
func (f *scheduleFixture) removeMember() {
	f.t.Helper()
	require.NoError(f.t, database.Conn().Model(&models.User{}).
		Where("id = (SELECT id FROM users WHERE organization_id = ? AND is_owner = false AND deleted_at IS NULL LIMIT 1)", f.r.Organization.ID).
		Update("deleted_at", time.Now()).Error)
}

// assertOnlyNextPeriodUpdates fails when any request could change the
// current period or charge now.
func (f *scheduleFixture) assertOnlyNextPeriodUpdates() {
	f.t.Helper()
	for _, patch := range f.polar.Patches() {
		assert.Equal(f.t, "next_period", patch.Body["proration_behavior"], "a seat update must not charge now")
	}
}

func (f *scheduleFixture) assertAdmissionBlocked() {
	f.t.Helper()
	usage, err := entitlements.ResolveSeats(database.Conn(), f.r.Organization.ID)
	require.NoError(f.t, err)
	assert.True(f.t, usage.PaymentBlocked)
	assert.Equal(f.t, new(0), usage.Invitable, "no invitations while past due")
	assert.Equal(f.t, new(0), usage.Purchasable, "no purchases while past due")
	assert.Equal(f.t, new(usage.Occupied), usage.JoinLimit(), "no acceptance while past due")
}

func Test__SchedulePastDueSubscriptionReducesTheNextPeriod(t *testing.T) {
	tests := []struct {
		name      string
		graceDays int
		// pastDueDaysAgo moves the stored start of the failure back.
		pastDueDaysAgo int
		teamFeatures   bool
	}{
		{name: "within grace", graceDays: 7, teamFeatures: true},
		{name: "after the grace period ends", graceDays: 7, pastDueDaysAgo: 8},
		{name: "without grace", graceDays: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newScheduleFixture(t, 10, 9)
			f.pastDue(tt.graceDays)
			if tt.pastDueDaysAgo > 0 {
				f.updatePlan(map[string]any{
					"polar_subscription_status": models.PolarSubscriptionStatusPastDue,
					"past_due_since":            time.Now().Add(-time.Duration(tt.pastDueDaysAgo) * 24 * time.Hour),
				})
			}

			result := f.schedule()

			assert.Equal(t, ActionFixed, result.Action)
			assert.Equal(t, new(9), result.Target)
			assert.Equal(t, []polarfake.Request{{Method: http.MethodPatch, SubscriptionID: f.sub.ID, Body: nextPeriod(9)}}, f.polar.Patches())
			sub := f.polar.Subscription(f.sub.ID)
			assert.Equal(t, 10, sub.Seats, "the current period keeps its seats")
			assert.Equal(t, new(9), sub.PendingSeats)
			plan := f.plan()
			assert.Equal(t, models.PolarSubscriptionStatusPastDue, plan.PolarSubscriptionStatus)
			assert.Equal(t, tt.teamFeatures, plan.Catalog.PerSeat, "the plan row is on the expected catalog")
			assert.Equal(t, new(10), plan.PurchasedSeats)
			assert.Equal(t, new(9), plan.RenewalSeatsScheduled)
			f.assertAdmissionBlocked()
		})
	}
}

func Test__SchedulePastDueSubscriptionNeverAddsSeats(t *testing.T) {
	f := newScheduleFixture(t, 2, 3)
	f.pastDue(0)

	result := f.schedule()

	assert.Equal(t, new(2), result.Target, "the target is the paid seats, not the member count")
	assert.Equal(t, []polarfake.Request{{Method: http.MethodPatch, SubscriptionID: f.sub.ID, Body: nextPeriod(2)}}, f.polar.Patches())
	sub := f.polar.Subscription(f.sub.ID)
	assert.Equal(t, 2, sub.Seats)
	assert.Nil(t, sub.PendingSeats)
	f.assertAdmissionBlocked()
}

func Test__ScheduleFailsForMissingPaidSeats(t *testing.T) {
	tests := []struct {
		name          string
		storedSeats   int
		snapshotSeats int
		// staleSnapshot makes the apply ignore the Polar snapshot, so the
		// stored plan keeps its seats.
		staleSnapshot bool
		// storedInvalid is true when the stored plan has no paid seats
		// after the apply.
		storedInvalid bool
	}{
		{name: "Polar reports no seats", storedSeats: 10, snapshotSeats: 0, storedInvalid: true},
		{name: "older Polar snapshot reports no seats", storedSeats: 10, snapshotSeats: 0, staleSnapshot: true},
		{name: "stored plan reports no seats", storedSeats: 0, snapshotSeats: 10, staleSnapshot: true, storedInvalid: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newScheduleFixture(t, 10, 9)
			f.pastDue(0)
			f.updatePlan(map[string]any{
				"polar_subscription_status": models.PolarSubscriptionStatusPastDue,
				"purchased_seats":           tt.storedSeats,
				"polar_modified_at":         f.now,
			})
			f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) {
				sub.Seats = tt.snapshotSeats
				if tt.staleSnapshot {
					sub.ModifiedAt = new(f.now.Add(-time.Hour))
				}
			})

			result, err := f.scheduler.Schedule(context.Background(), f.r.Organization.ID)

			require.Error(t, err, "missing capacity needs repair and is not a success")
			assert.Equal(t, tt.storedInvalid, errors.Is(err, entitlements.ErrInvalidPaidSeats))
			assert.Empty(t, result.Action)
			assert.Empty(t, f.polar.Patches())
			assert.Nil(t, f.plan().RenewalSeatsScheduled)
		})
	}
}

func Test__ScheduleReportsAPolarRejectionAndRetriesLater(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	f.pastDue(0)
	// Polar rejects the seat update of the past-due subscription.
	client := &observedClient{Client: f.scheduler.client, beforeSchedule: func() {
		f.polar.FailNext(1, http.StatusForbidden, "InactiveSubscription")
	}}

	result, err := NewScheduler(database.Conn(), client, func() time.Time { return f.now }).Schedule(context.Background(), f.r.Organization.ID)

	require.Error(t, err)
	assert.True(t, polar.IsSubscriptionInactive(err), "the caller can identify the rejection")
	assert.NotEqual(t, ActionFixed, result.Action)
	assert.Nil(t, f.plan().RenewalSeatsScheduled, "a rejected update is not recorded as sent")
	assert.Nil(t, f.polar.Subscription(f.sub.ID).PendingSeats)

	result = f.schedule()

	assert.Equal(t, ActionFixed, result.Action)
	assert.Equal(t, new(9), f.polar.Subscription(f.sub.ID).PendingSeats)
	assert.Equal(t, new(9), f.plan().RenewalSeatsScheduled)
	f.assertOnlyNextPeriodUpdates()
}

func Test__SchedulePastDueReductionSurvivesTheCycleAndRecovery(t *testing.T) {
	f := newScheduleFixture(t, 10, 10)
	f.pastDue(0)
	assert.Equal(t, ActionInSync, f.schedule().Action, "all paid seats are occupied")

	f.removeMember()
	result := f.schedule()

	assert.Equal(t, ActionFixed, result.Action)
	assert.Equal(t, new(9), f.polar.Subscription(f.sub.ID).PendingSeats)
	f.assertAdmissionBlocked()

	// Polar renews the subscription while it still retries the payment.
	periodEnd := f.sub.PeriodEnd
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) {
		sub.Seats = *sub.PendingSeats
		sub.PendingSeats = nil
		sub.PeriodStart = periodEnd
		sub.PeriodEnd = periodEnd.AddDate(0, 1, 0)
	})
	patches := len(f.polar.Patches())

	result = f.schedule()

	assert.Equal(t, ActionInSync, result.Action)
	assert.Equal(t, new(9), result.Target)
	assert.Len(t, f.polar.Patches(), patches, "the renewed period already has the target")
	plan := f.plan()
	assert.Equal(t, new(9), plan.PurchasedSeats)
	assert.True(t, plan.CurrentPeriodStart.Equal(periodEnd))
	f.assertAdmissionBlocked()

	// The payment recovers.
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) { sub.Status = models.PolarSubscriptionStatusActive })

	result = f.schedule()

	assert.Equal(t, ActionInSync, result.Action)
	assert.Equal(t, new(9), result.Target)
	assert.Len(t, f.polar.Patches(), patches)
	plan = f.plan()
	assert.Equal(t, models.PolarSubscriptionStatusActive, plan.PolarSubscriptionStatus)
	assert.True(t, plan.Catalog.PerSeat, "recovery restores the Team plan")
	assert.Equal(t, new(9), plan.RenewalSeatsScheduled)
	f.assertOnlyNextPeriodUpdates()
}

// observedClient runs a hook around each Polar call that Schedule makes.
type observedClient struct {
	Client
	beforeGet      func()
	afterGet       func(snapshot *polar.SubscriptionData)
	beforeSchedule func()
}

func (c *observedClient) GetSubscription(ctx context.Context, id string) (*polar.SubscriptionData, error) {
	if c.beforeGet != nil {
		c.beforeGet()
	}
	snapshot, err := c.Client.GetSubscription(ctx, id)
	if err == nil && c.afterGet != nil {
		c.afterGet(snapshot)
	}
	return snapshot, err
}

func (c *observedClient) ScheduleSubscriptionSeats(ctx context.Context, id string, seats int) (*polar.SubscriptionData, error) {
	if c.beforeSchedule != nil {
		c.beforeSchedule()
	}
	return c.Client.ScheduleSubscriptionSeats(ctx, id, seats)
}

func (f *scheduleFixture) scheduleWith(client Client) Result {
	f.t.Helper()
	result, err := NewScheduler(database.Conn(), client, func() time.Time { return f.now }).Schedule(context.Background(), f.r.Organization.ID)
	require.NoError(f.t, err)
	return result
}

func Test__ScheduleCapsTheTargetAtTheSeatsOfTheRenewedPeriod(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	// The renewal event never arrived. Polar renewed with fewer seats.
	renewedStart := f.now
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) {
		sub.Seats = 4
		sub.PeriodStart = renewedStart
		sub.PeriodEnd = renewedStart.AddDate(0, 1, 0)
	})

	result := f.schedule()

	assert.Equal(t, new(4), result.Target, "the target never exceeds the seats of the renewed period")
	assert.Equal(t, []polarfake.Request{{Method: http.MethodPatch, SubscriptionID: f.sub.ID, Body: nextPeriod(4)}}, f.polar.Patches())
	plan := f.plan()
	assert.Equal(t, new(4), plan.PurchasedSeats)
	assert.True(t, plan.CurrentPeriodStart.Equal(renewedStart))
	assert.Equal(t, new(4), plan.RenewalSeatsScheduled)
}

func Test__ScheduleAppliesTheResponseToTheSeatUpdate(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	// Seats change in Polar between the read and the update, and the webhook
	// is lost. The update response is the only report of the new seats.
	client := &observedClient{Client: f.scheduler.client, beforeSchedule: func() {
		f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) { sub.Seats = 12 })
	}}

	result := f.scheduleWith(client)

	assert.Equal(t, ActionFixed, result.Action)
	assert.Equal(t, new(12), f.plan().PurchasedSeats)
}

func Test__ScheduleDoesNotLetAnOlderResponseOverwriteANewerWebhook(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) { sub.ModifiedAt = new(f.now.Add(-time.Hour)) })
	// A payment failure webhook arrives after Polar answered the read.
	client := &observedClient{Client: f.scheduler.client, afterGet: func(snapshot *polar.SubscriptionData) {
		event := polar.SubscriptionWebhookEvent{}
		require.NoError(t, json.Unmarshal([]byte(fmt.Sprintf(
			`{"type":"subscription.updated","data":{"id":%q,"product_id":%q,"status":"past_due","seats":10,"external_customer_id":%q,
			"current_period_start":%q,"current_period_end":%q,"modified_at":%q}}`,
			f.sub.ID, f.sub.ProductID, f.r.Organization.ID,
			f.sub.PeriodStart.Format(time.RFC3339), f.sub.PeriodEnd.Format(time.RFC3339), time.Now().Add(time.Minute).Format(time.RFC3339),
		)), &event))
		require.NoError(t, polar.ApplySubscriptionEvent(context.Background(), database.Conn(), &event))
	}}

	f.scheduleWith(client)

	assert.Equal(t, models.PolarSubscriptionStatusPastDue, f.plan().PolarSubscriptionStatus, "the older active response must not undo the failure")
	// A past-due subscription still renews, so it gets the scheduled count.
	assert.Equal(t, []polarfake.Request{{Method: http.MethodPatch, SubscriptionID: f.sub.ID, Body: nextPeriod(9)}}, f.polar.Patches())
}

func Test__ScheduleMakesNoPolarRequestWhileTheOrganizationRowIsLocked(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	calls := 0
	requireUnlocked := func() {
		calls++
		// A different connection can lock the row only when Schedule holds no lock.
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			return tx.Exec("SELECT id FROM organizations WHERE id = ? FOR UPDATE NOWAIT", f.r.Organization.ID).Error
		})
		assert.NoError(t, err, "the organization row is locked during a Polar request")
	}
	client := &observedClient{
		Client:         f.scheduler.client,
		beforeGet:      requireUnlocked,
		beforeSchedule: requireUnlocked,
	}

	result := f.scheduleWith(client)

	assert.Equal(t, ActionFixed, result.Action)
	assert.Equal(t, 2, calls, "the read and the update were both checked")
}

func Test__ScheduleFailsAnAbandonedPurchaseAndSchedules(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	purchase := models.NewSeatPurchase(f.r.Organization.ID, nil, f.sub.ID, 10, 12, nil)
	require.NoError(t, purchase.Create(database.Conn()))
	require.NoError(t, purchase.UpdateStatus(database.Conn(), models.SeatPurchaseStatusUnknown, ""))

	result := f.schedule()

	// Polar still reports the original seats after the purchase, so the
	// abandoned purchase failed and no longer blocks the renewal seat count.
	assert.Equal(t, ActionFixed, result.Action)
	assert.Equal(t, []polarfake.Request{{Method: http.MethodPatch, SubscriptionID: f.sub.ID, Body: nextPeriod(9)}}, f.polar.Patches())
	stored, err := models.FindLatestSeatPurchase(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.SeatPurchaseStatusFailed, stored.Status)
	assert.Equal(t, models.SeatPurchaseErrorNotApplied, stored.ErrorCode)
}

func Test__ScheduleExpiresAPendingPurchaseAfterTheConfirmationWindow(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	purchase := models.NewSeatPurchase(f.r.Organization.ID, nil, f.sub.ID, 10, 12, nil)
	require.NoError(t, purchase.Create(database.Conn()))
	created := time.Now().Add(-polar.SeatPurchaseConfirmationWindow - time.Minute)
	require.NoError(t, database.Conn().Model(&models.SeatPurchase{}).Where("id = ?", purchase.ID).Update("created_at", created).Error)

	result := f.schedule()

	assert.Equal(t, ActionFixed, result.Action)
	assert.Equal(t, []polarfake.Request{{Method: http.MethodPatch, SubscriptionID: f.sub.ID, Body: nextPeriod(9)}}, f.polar.Patches())
	stored, err := models.FindLatestSeatPurchase(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.SeatPurchaseStatusFailed, stored.Status)
	assert.Equal(t, models.SeatPurchaseErrorNotApplied, stored.ErrorCode)
}

func Test__ScheduleAppliesTheSnapshotThatConfirmsAPurchase(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	purchase := models.NewSeatPurchase(f.r.Organization.ID, nil, f.sub.ID, 10, 12, nil)
	require.NoError(t, purchase.Create(database.Conn()))
	require.NoError(t, purchase.UpdateStatus(database.Conn(), models.SeatPurchaseStatusUnknown, ""))
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) { sub.Seats = 12 })

	result := f.schedule()

	// Polar reports the purchased seats, so the purchase succeeded and the
	// stored plan picks up the seats. The purchase replaced the scheduled
	// count, so it is scheduled again.
	assert.Equal(t, ActionFixed, result.Action)
	assert.Equal(t, []polarfake.Request{{Method: http.MethodPatch, SubscriptionID: f.sub.ID, Body: nextPeriod(9)}}, f.polar.Patches())
	stored, err := models.FindLatestSeatPurchase(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.SeatPurchaseStatusSucceeded, stored.Status)
	assert.Equal(t, new(12), f.plan().PurchasedSeats)
}

func Test__ScheduleWithRetryRetriesOnlyTransientPolarFailures(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		code     string
		failures int
		attempts int
		fixed    bool
	}{
		{name: "rate limited", status: http.StatusTooManyRequests, failures: 2, attempts: 3, fixed: true},
		{name: "subscription locked", status: http.StatusConflict, code: "SubscriptionLocked", failures: 1, attempts: 2, fixed: true},
		{name: "server error", status: http.StatusBadGateway, failures: 2, attempts: 3, fixed: true},
		{name: "outage longer than the retries", status: http.StatusServiceUnavailable, failures: 3, attempts: 3},
		{name: "rejected values", status: http.StatusUnprocessableEntity, code: "RequestValidationError", failures: 1, attempts: 1},
		{name: "unauthorized", status: http.StatusUnauthorized, failures: 1, attempts: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newScheduleFixture(t, 10, 9)
			f.polar.FailNext(tt.failures, tt.status, tt.code)

			f.scheduler.scheduleWithRetry(context.Background(), f.r.Organization.ID, []time.Duration{0, 0})

			gets := 0
			for _, request := range f.polar.Requests() {
				if request.Method == http.MethodGet {
					gets++
				}
			}
			assert.Equal(t, tt.attempts, gets)
			if tt.fixed {
				assert.Equal(t, new(9), f.polar.Subscription(f.sub.ID).PendingSeats)
				return
			}
			assert.Empty(t, f.polar.Patches())
			assert.Nil(t, f.polar.Subscription(f.sub.ID).PendingSeats)
		})
	}
}

// unresolvedPurchase records an unknown purchase from fromSeats to toSeats on
// the subscription. A nil period records a purchase from before operation
// keys.
func (f *scheduleFixture) unresolvedPurchase(subscriptionID string, fromSeats, toSeats int, createdAt time.Time, periodStart, periodEnd *time.Time) *models.SeatPurchase {
	f.t.Helper()
	var operation *models.SeatPurchaseOperation
	if periodStart != nil {
		operation = &models.SeatPurchaseOperation{
			Key: uuid.NewString(), Fingerprint: models.SeatPurchaseFingerprint(toSeats-fromSeats, fromSeats),
			PeriodStart: *periodStart, PeriodEnd: *periodEnd,
		}
	}
	purchase := models.NewSeatPurchase(f.r.Organization.ID, nil, subscriptionID, fromSeats, toSeats, operation)
	purchase.Status = models.SeatPurchaseStatusUnknown
	purchase.CreatedAt = createdAt
	require.NoError(f.t, purchase.Create(database.Conn()))
	return purchase
}

// assertStillUnresolved fails when Schedule changed the purchase.
func (f *scheduleFixture) assertStillUnresolved(purchase *models.SeatPurchase) {
	f.t.Helper()
	var stored models.SeatPurchase
	require.NoError(f.t, database.Conn().First(&stored, "id = ?", purchase.ID).Error)
	assert.Equal(f.t, models.SeatPurchaseStatusUnknown, stored.Status, "renewal maintenance never settles a purchase")
	assert.Empty(f.t, stored.ErrorCode)
	assert.Equal(f.t, purchase.OperationKey, stored.OperationKey)
	latest, err := models.FindLatestSeatPurchase(database.Conn(), f.r.Organization.ID)
	require.NoError(f.t, err)
	assert.Equal(f.t, purchase.ID, latest.ID, "the purchase keeps its identity")
}

// previousPeriod is the billing period before the current fake period.
func (f *scheduleFixture) previousPeriod() (start, end time.Time) {
	current := f.polar.Subscription(f.sub.ID).PeriodStart
	return current.AddDate(0, -1, 0), current
}

func captureLogs(t *testing.T) *logtest.Hook {
	t.Helper()
	hook := logtest.NewGlobal()
	t.Cleanup(func() { log.StandardLogger().ReplaceHooks(make(log.LevelHooks)) })
	return hook
}

func Test__ScheduleContinuesPastAnUnresolvedPurchaseFromAnEarlierPeriod(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	// Polar renewed after the outcome of the purchase was lost. The new
	// period proves nothing about the charge, but the purchase can no longer
	// change this period.
	start, end := f.previousPeriod()
	purchase := f.unresolvedPurchase(f.sub.ID, 10, 11, end.Add(-time.Hour), &start, &end)

	result := f.schedule()

	assert.Equal(t, ActionFixed, result.Action)
	assert.Equal(t, purchase.ID, result.SeatPurchaseID)
	assert.Equal(t, []polarfake.Request{{Method: http.MethodPatch, SubscriptionID: f.sub.ID, Body: nextPeriod(9)}}, f.polar.Patches())
	f.assertStillUnresolved(purchase)
}

func Test__ScheduleMaintainsAReplacementSubscriptionWhenTheOldOneCannotBeRead(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	logs := captureLogs(t)
	old := polarfake.Subscription{ID: "sub_old_" + uuid.NewString(), Status: models.PolarSubscriptionStatusCanceled}
	f.polar.Add(old)
	start, end := f.previousPeriod()
	purchase := f.unresolvedPurchase(old.ID, 4, 5, f.now.Add(-48*time.Hour), &start, &end)
	// The reconciliation reads the old subscription first, and Polar fails.
	f.polar.FailNext(1, http.StatusNotFound, "ResourceNotFound")

	result := f.schedule()

	assert.Equal(t, ActionFixed, result.Action)
	assert.Equal(t, []polarfake.Request{{Method: http.MethodPatch, SubscriptionID: f.sub.ID, Body: nextPeriod(9)}}, f.polar.Patches())
	f.assertStillUnresolved(purchase)
	var failed []*log.Entry
	for _, entry := range logs.AllEntries() {
		if entry.Message == SeatPurchaseReconcileFailedKey {
			failed = append(failed, entry)
		}
	}
	require.Len(t, failed, 1)
	assert.Equal(t, purchase.ID.String(), failed[0].Data["seat_purchase_id"])
	assert.Equal(t, old.ID, failed[0].Data["polar_subscription_id"])
	assert.Equal(t, f.sub.ID, failed[0].Data["current_polar_subscription_id"])
}

func Test__ScheduleFailsWhenThePurchaseOfTheCurrentSubscriptionCannotBeRead(t *testing.T) {
	f := newScheduleFixture(t, 10, 9)
	purchase := f.unresolvedPurchase(f.sub.ID, 10, 11, f.now.Add(-time.Hour), new(f.sub.PeriodStart), new(f.sub.PeriodEnd))
	f.polar.FailNext(1, http.StatusServiceUnavailable, "")

	_, err := f.scheduler.Schedule(context.Background(), f.r.Organization.ID)

	require.Error(t, err, "a purchase in the current period can still race the update")
	assert.Empty(t, f.polar.Patches())
	f.assertStillUnresolved(purchase)
}

func Test__ScheduleKeepsTheGuardForPurchasesThatCanStillRace(t *testing.T) {
	tests := []struct {
		name string
		// renewedAgo is how long ago the current Polar period started.
		renewedAgo time.Duration
		// createdBeforeRenewal is how long before the current period the
		// purchase started.
		createdBeforeRenewal time.Duration
		legacy               bool
		// clearPeriod removes the period from the Polar snapshots.
		clearPeriod bool
		blocks      bool
	}{
		{name: "keyed purchase that can still reach Polar", renewedAgo: 30 * time.Second, createdBeforeRenewal: time.Minute, blocks: true},
		{name: "keyed purchase without a verified period", renewedAgo: time.Hour, createdBeforeRenewal: time.Hour, clearPeriod: true, blocks: true},
		{name: "legacy purchase whose window closed after the period started", renewedAgo: time.Hour, createdBeforeRenewal: time.Minute, legacy: true, blocks: true},
		{name: "legacy purchase whose window closed before the period started", renewedAgo: time.Hour, createdBeforeRenewal: 10 * time.Minute, legacy: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newScheduleFixture(t, 10, 9)
			renewed := f.now.Add(-tt.renewedAgo)
			f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) { sub.PeriodStart = renewed })
			start, end := renewed.AddDate(0, -1, 0), renewed
			created := renewed.Add(-tt.createdBeforeRenewal)
			var purchase *models.SeatPurchase
			if tt.legacy {
				purchase = f.unresolvedPurchase(f.sub.ID, 10, 11, created, nil, nil)
			} else {
				purchase = f.unresolvedPurchase(f.sub.ID, 10, 11, created, &start, &end)
			}
			client := Client(f.scheduler.client)
			if tt.clearPeriod {
				client = &observedClient{Client: client, afterGet: func(snapshot *polar.SubscriptionData) {
					snapshot.CurrentPeriodStart.Time = time.Time{}
				}}
			}

			result := f.scheduleWith(client)

			f.assertStillUnresolved(purchase)
			if tt.blocks {
				assert.Equal(t, ActionSkipped, result.Action)
				assert.Equal(t, SkipSeatPurchaseInFlight, result.SkipReason)
				assert.Equal(t, purchase.ID, result.SeatPurchaseID)
				assert.Empty(t, f.polar.Patches())
				return
			}
			assert.Equal(t, ActionFixed, result.Action)
			assert.Equal(t, []polarfake.Request{{Method: http.MethodPatch, SubscriptionID: f.sub.ID, Body: nextPeriod(9)}}, f.polar.Patches())
		})
	}
}

func Test__ScheduleReducesTheRenewalAfterALostPurchaseResponse(t *testing.T) {
	f := newScheduleFixture(t, 9, 10)
	// A purchase from 9 to 10 seats lost its response. Polar applied it, but
	// no snapshot settled the purchase before the renewal.
	start, end := f.sub.PeriodStart, f.sub.PeriodEnd
	purchase := f.unresolvedPurchase(f.sub.ID, 9, 10, f.now.Add(-time.Hour), &start, &end)
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) {
		sub.Seats = 10
		sub.PeriodStart = end
		sub.PeriodEnd = end.AddDate(0, 1, 0)
	})
	f.removeMember()
	f.removeMember()
	f.removeMember()

	result := f.schedule()

	assert.Equal(t, ActionFixed, result.Action)
	assert.Equal(t, new(7), result.Target)
	assert.Equal(t, []polarfake.Request{{Method: http.MethodPatch, SubscriptionID: f.sub.ID, Body: nextPeriod(7)}}, f.polar.Patches())
	f.assertOnlyNextPeriodUpdates()
	assert.Equal(t, 10, f.polar.Subscription(f.sub.ID).Seats, "the current period keeps its seats")
	assert.Equal(t, new(7), f.plan().RenewalSeatsScheduled)
	f.assertStillUnresolved(purchase)

	assert.Equal(t, ActionInSync, f.schedule().Action)
	assert.Len(t, f.polar.Patches(), 1, "maintenance never repeats the charge or the schedule")
	f.assertStillUnresolved(purchase)
}
