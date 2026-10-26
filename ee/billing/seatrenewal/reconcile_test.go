package seatrenewal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/ee/billing/entitlements"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/ee/billing/testsupport/polarfake"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

type reconcileFixture struct {
	t          *testing.T
	r          *support.ResourceRegistry
	polar      *polarfake.Server
	sub        polarfake.Subscription
	reconciler *Reconciler
	now        time.Time
}

// newReconcileFixture is an organization with the owner and members-1 other
// members on an active per-seat Polar subscription with the given seats.
func newReconcileFixture(t *testing.T, seats, members int) *reconcileFixture {
	t.Helper()
	r := support.Setup(t)
	t.Cleanup(r.Close)
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	f := &reconcileFixture{t: t, r: r, polar: polarfake.NewServer(t), now: time.Now().UTC().Truncate(time.Second)}
	f.sub = f.polar.AddSeatPlan(database.Conn(), r.Organization.ID, seats, f.now.Add(10*24*time.Hour))
	for range members - 1 {
		support.CreateUser(t, r, r.Organization.ID)
	}
	f.reconciler = NewReconciler(database.Conn(), polar.NewClientFromEnv())
	return f
}

func (f *reconcileFixture) reconcile() Result {
	f.t.Helper()
	return f.reconcileWith(f.reconciler.client)
}

func (f *reconcileFixture) reconcileWith(client Client) Result {
	f.t.Helper()
	result, err := NewReconciler(database.Conn(), client).Reconcile(context.Background(), f.r.Organization.ID)
	require.NoError(f.t, err)
	return result
}

func (f *reconcileFixture) updatePlan(values map[string]any) {
	f.t.Helper()
	require.NoError(f.t, database.Conn().Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", f.r.Organization.ID).Updates(values).Error)
}

func (f *reconcileFixture) plan() *billingmodels.OrganizationBillingPlan {
	f.t.Helper()
	plan, err := billingmodels.FindOrganizationBillingPlan(database.Conn(), f.r.Organization.ID)
	require.NoError(f.t, err)
	return plan
}

// renew moves the fake subscription to the next period with the given seats,
// as Polar does at renewal, without a webhook.
func (f *reconcileFixture) renew(seats int) time.Time {
	f.t.Helper()
	periodEnd := f.polar.Subscription(f.sub.ID).PeriodEnd
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) {
		sub.Seats = seats
		sub.PendingSeats = nil
		sub.PeriodStart = periodEnd
		sub.PeriodEnd = periodEnd.AddDate(0, 1, 0)
	})
	return periodEnd
}

// removeMember removes one member that is not the owner.
func (f *reconcileFixture) removeMember() {
	f.t.Helper()
	require.NoError(f.t, database.Conn().Model(&models.User{}).
		Where("id = (SELECT id FROM users WHERE organization_id = ? AND is_owner = false AND deleted_at IS NULL LIMIT 1)", f.r.Organization.ID).
		Update("deleted_at", time.Now()).Error)
}

func Test__ReconcileRetainsSpareCapacityWithoutAQuantityWrite(t *testing.T) {
	// Ten paid seats and seven members: three spare seats stay paid.
	f := newReconcileFixture(t, 10, 7)
	f.removeMember()

	result := f.reconcile()

	assert.Equal(t, ActionReconciled, result.Action)
	assert.Equal(t, new(10), result.CurrentSeats)
	assert.Empty(t, f.polar.Patches(), "reconciliation never writes a seat count")
	assert.Nil(t, f.polar.Subscription(f.sub.ID).PendingSeats, "renewal keeps the paid seats")
	assert.Equal(t, new(10), f.plan().PurchasedSeats)

	renewedStart := f.renew(10)
	f.reconcile()

	plan := f.plan()
	assert.Equal(t, new(10), plan.PurchasedSeats, "the renewed period keeps every paid seat")
	assert.True(t, plan.CurrentPeriodStart.Equal(renewedStart))
	assert.Empty(t, f.polar.Patches())
}

func Test__ReconcileLeavesALegacyScheduledReductionUntouched(t *testing.T) {
	f := newReconcileFixture(t, 10, 9)
	// The retired scheduler sent this reduction before the change.
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) { sub.PendingSeats = new(9) })

	f.reconcile()

	assert.Empty(t, f.polar.Patches(), "legacy cleanup owns the existing reduction")
	assert.Equal(t, new(9), f.polar.Subscription(f.sub.ID).PendingSeats)
}

func Test__ReconcileSkipsPlansThatPolarDoesNotManage(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]any
		reason string
	}{
		{name: "admin plan", values: map[string]any{"plan_source": billingmodels.BillingPlanSourceAdmin}, reason: SkipAdminPlan},
		{name: "no subscription", values: map[string]any{"polar_subscription_id": nil}, reason: SkipNoSubscription},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newReconcileFixture(t, 10, 9)
			f.updatePlan(tt.values)

			result := f.reconcile()

			assert.Equal(t, ActionSkipped, result.Action)
			assert.Equal(t, tt.reason, result.SkipReason)
			assert.Empty(t, f.polar.Requests(), "a plan that Polar does not manage needs no Polar request")
		})
	}
}

func Test__ReconcileStoresThePolarStatusOfEveryLifecycle(t *testing.T) {
	tests := []struct {
		name              string
		status            string
		cancelAtPeriodEnd bool
	}{
		{name: "past due", status: billingmodels.PolarSubscriptionStatusPastDue},
		{name: "paused", status: billingmodels.PolarSubscriptionStatusPaused},
		{name: "unpaid", status: billingmodels.PolarSubscriptionStatusUnpaid},
		{name: "canceled", status: billingmodels.PolarSubscriptionStatusCanceled},
		{name: "set to cancel", status: billingmodels.PolarSubscriptionStatusActive, cancelAtPeriodEnd: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newReconcileFixture(t, 10, 9)
			f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) {
				sub.Status = tt.status
				sub.CancelAtPeriodEnd = tt.cancelAtPeriodEnd
			})

			f.reconcile()

			plan := f.plan()
			assert.Equal(t, tt.status, plan.PolarSubscriptionStatus)
			assert.Equal(t, tt.cancelAtPeriodEnd, plan.CancelAtPeriodEnd)
			assert.Equal(t, new(10), plan.PurchasedSeats)
			assert.Empty(t, f.polar.Patches())
		})
	}
}

func Test__ReconcileKeepsPaidSeatsThroughAPastDueCycleAndRecovery(t *testing.T) {
	f := newReconcileFixture(t, 10, 8)
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) { sub.Status = billingmodels.PolarSubscriptionStatusPastDue })
	f.reconcile()
	usage, err := entitlements.ResolveSeats(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)
	assert.True(t, usage.PaymentBlocked, "no member can join while the payment fails")

	// Polar renews while it retries the payment, and then the payment recovers.
	renewedStart := f.renew(10)
	f.reconcile()
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) { sub.Status = billingmodels.PolarSubscriptionStatusActive })
	f.reconcile()

	plan := f.plan()
	assert.Equal(t, billingmodels.PolarSubscriptionStatusActive, plan.PolarSubscriptionStatus)
	assert.True(t, plan.Catalog.PerSeat, "recovery restores the Team plan")
	assert.Equal(t, new(10), plan.PurchasedSeats)
	assert.True(t, plan.CurrentPeriodStart.Equal(renewedStart))
	assert.Empty(t, f.polar.Patches())
}

// observedClient runs a hook around each Polar read that Reconcile makes.
type observedClient struct {
	Client
	beforeGet func()
	afterGet  func(snapshot *polar.SubscriptionData)
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

func Test__ReconcileDoesNotLetAnOlderReadOverwriteANewerWebhook(t *testing.T) {
	f := newReconcileFixture(t, 10, 9)
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) { sub.ModifiedAt = new(f.now.Add(-time.Hour)) })
	// A payment failure webhook arrives after Polar answered the read.
	client := &observedClient{Client: f.reconciler.client, afterGet: func(*polar.SubscriptionData) {
		event := polar.SubscriptionWebhookEvent{}
		require.NoError(t, json.Unmarshal([]byte(fmt.Sprintf(
			`{"type":"subscription.updated","data":{"id":%q,"product_id":%q,"status":"past_due","seats":10,"external_customer_id":%q,
			"current_period_start":%q,"current_period_end":%q,"modified_at":%q}}`,
			f.sub.ID, f.sub.ProductID, f.r.Organization.ID,
			f.sub.PeriodStart.Format(time.RFC3339), f.sub.PeriodEnd.Format(time.RFC3339), time.Now().Add(time.Minute).Format(time.RFC3339),
		)), &event))
		require.NoError(t, polar.ApplySubscriptionEvent(context.Background(), database.Conn(), &event))
	}}

	f.reconcileWith(client)

	assert.Equal(t, billingmodels.PolarSubscriptionStatusPastDue, f.plan().PolarSubscriptionStatus, "the older active read must not undo the failure")
	assert.Empty(t, f.polar.Patches())
}

func Test__ReconcileMakesNoPolarRequestWhileTheOrganizationRowIsLocked(t *testing.T) {
	f := newReconcileFixture(t, 10, 9)
	purchase := billingmodels.NewSeatPurchase(f.r.Organization.ID, nil, f.sub.ID, 10, 12, nil)
	purchase.Status = billingmodels.SeatPurchaseStatusUnknown
	require.NoError(t, purchase.Create(database.Conn()))
	calls := 0
	client := &observedClient{Client: f.reconciler.client, beforeGet: func() {
		calls++
		// A different connection can lock the row only when Reconcile holds no lock.
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			return tx.Exec("SELECT id FROM organizations WHERE id = ? FOR UPDATE NOWAIT", f.r.Organization.ID).Error
		})
		assert.NoError(t, err, "the organization row is locked during a Polar request")
	}}

	f.reconcileWith(client)

	assert.Equal(t, 2, calls, "the purchase read and the subscription read were both checked")
}

// unresolvedPurchase records an unknown purchase from fromSeats to toSeats on
// the subscription. A nil period records a purchase from before operation
// keys.
func (f *reconcileFixture) unresolvedPurchase(subscriptionID string, fromSeats, toSeats int, createdAt time.Time, periodStart, periodEnd *time.Time) *billingmodels.SeatPurchase {
	f.t.Helper()
	var operation *billingmodels.SeatPurchaseOperation
	if periodStart != nil {
		operation = &billingmodels.SeatPurchaseOperation{
			Key: uuid.NewString(), Fingerprint: billingmodels.SeatPurchaseFingerprint(toSeats-fromSeats, fromSeats),
			PeriodStart: *periodStart, PeriodEnd: *periodEnd,
		}
	}
	purchase := billingmodels.NewSeatPurchase(f.r.Organization.ID, nil, subscriptionID, fromSeats, toSeats, operation)
	purchase.Status = billingmodels.SeatPurchaseStatusUnknown
	purchase.CreatedAt = createdAt
	require.NoError(f.t, purchase.Create(database.Conn()))
	return purchase
}

// assertStillUnresolved fails when Reconcile changed the purchase.
func (f *reconcileFixture) assertStillUnresolved(purchase *billingmodels.SeatPurchase) {
	f.t.Helper()
	var stored billingmodels.SeatPurchase
	require.NoError(f.t, database.Conn().First(&stored, "id = ?", purchase.ID).Error)
	assert.Equal(f.t, billingmodels.SeatPurchaseStatusUnknown, stored.Status, "reconciliation never settles a purchase without evidence")
	assert.Empty(f.t, stored.ErrorCode)
	assert.Equal(f.t, purchase.OperationKey, stored.OperationKey, "the purchase keeps its identity")
}

func Test__ReconcileSettlesOnlyAPurchaseThatPolarProves(t *testing.T) {
	t.Run("snapshot proves the purchase", func(t *testing.T) {
		f := newReconcileFixture(t, 10, 9)
		purchase := f.unresolvedPurchase(f.sub.ID, 10, 12, time.Now().Add(-time.Minute), new(f.sub.PeriodStart), new(f.sub.PeriodEnd))
		f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) { sub.Seats = 12 })

		f.reconcile()

		var stored billingmodels.SeatPurchase
		require.NoError(t, database.Conn().First(&stored, "id = ?", purchase.ID).Error)
		assert.Equal(t, billingmodels.SeatPurchaseStatusSucceeded, stored.Status)
		assert.Equal(t, new(12), f.plan().PurchasedSeats)
		assert.Empty(t, f.polar.Patches(), "settlement never sends another charge")
	})

	t.Run("snapshot proves nothing", func(t *testing.T) {
		f := newReconcileFixture(t, 10, 9)
		purchase := f.unresolvedPurchase(f.sub.ID, 10, 12, time.Now().Add(-time.Minute), new(f.sub.PeriodStart), new(f.sub.PeriodEnd))

		f.reconcile()

		f.assertStillUnresolved(purchase)
		assert.Empty(t, f.polar.Patches())
	})
}

func Test__ReconcileRestoresCapacityAfterALostPurchaseResponseWithoutAnotherCharge(t *testing.T) {
	f := newReconcileFixture(t, 9, 9)
	// A purchase from 9 to 10 seats lost its response. Polar applied it, but
	// no snapshot proved the purchase before the renewal.
	start, end := f.sub.PeriodStart, f.sub.PeriodEnd
	purchase := f.unresolvedPurchase(f.sub.ID, 9, 10, f.now.Add(-time.Hour), &start, &end)
	f.renew(10)

	f.reconcile()

	assert.Equal(t, new(10), f.plan().PurchasedSeats, "the verified read restores the committed capacity")
	assert.Empty(t, f.polar.Patches(), "reconciliation never repeats the charge")
	f.assertStillUnresolved(purchase)
}

func captureLogs(t *testing.T) *logtest.Hook {
	t.Helper()
	hook := logtest.NewGlobal()
	t.Cleanup(func() { log.StandardLogger().ReplaceHooks(make(log.LevelHooks)) })
	return hook
}

func Test__ReconcileMaintainsAReplacementSubscriptionWhenTheOldOneCannotBeRead(t *testing.T) {
	f := newReconcileFixture(t, 10, 9)
	logs := captureLogs(t)
	old := polarfake.Subscription{ID: "sub_old_" + uuid.NewString(), Status: billingmodels.PolarSubscriptionStatusCanceled}
	f.polar.Add(old)
	current := f.polar.Subscription(f.sub.ID).PeriodStart
	start, end := current.AddDate(0, -1, 0), current
	purchase := f.unresolvedPurchase(old.ID, 4, 5, f.now.Add(-48*time.Hour), &start, &end)
	// The reconciliation reads the old subscription first, and Polar fails.
	f.polar.FailNext(1, http.StatusNotFound, "ResourceNotFound")
	f.polar.Update(f.sub.ID, func(sub *polarfake.Subscription) { sub.Seats = 11 })

	result := f.reconcile()

	assert.Equal(t, ActionReconciled, result.Action)
	assert.Equal(t, new(11), f.plan().PurchasedSeats, "the current subscription is still reconciled")
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

func Test__ReconcileFailsWhenThePurchaseOfTheCurrentSubscriptionCannotBeRead(t *testing.T) {
	f := newReconcileFixture(t, 10, 9)
	purchase := f.unresolvedPurchase(f.sub.ID, 10, 11, f.now.Add(-time.Hour), new(f.sub.PeriodStart), new(f.sub.PeriodEnd))
	f.polar.FailNext(1, http.StatusServiceUnavailable, "")

	_, err := f.reconciler.Reconcile(context.Background(), f.r.Organization.ID)

	require.Error(t, err)
	f.assertStillUnresolved(purchase)
}
