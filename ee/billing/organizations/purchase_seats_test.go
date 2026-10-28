package organizations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	polartest "github.com/superplanehq/superplane/ee/billing/testsupport/polar"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const seatPriceCents = 2000

// seatPurchaseFixture is a Team organization with an active Polar subscription
// and a fake Polar server that records every call.
type seatPurchaseFixture struct {
	t              *testing.T
	r              *support.ResourceRegistry
	catalog        billingmodels.BillingPlan
	subscriptionID string
	product        string
	periodStart    time.Time
	periodEnd      time.Time

	mu sync.Mutex
	// patchBodies are the seat purchase updates. Next-period updates are in
	// schedules.
	patchBodies []map[string]any
	getCount    int
	// onPatch writes the seat purchase response. The default applies the update.
	onPatch func(w http.ResponseWriter, f *seatPurchaseFixture, seats int)
	// polarSeats is the seat count that GET reports.
	polarSeats int
	// schedules are the next_period updates. SuperPlane must send none.
	schedules []map[string]any
	// pendingSeats is a next-period seat count that the retired automatic
	// scheduler left in Polar, when it differs from polarSeats.
	pendingSeats *int
	// modifiedAt is the subscription modification time that snapshots report.
	// The zero value makes each snapshot report the current time.
	modifiedAt time.Time
	// replays counts webhook replays so each snapshot is newer than the last.
	replays int
	// pastDueAt is sent as past_due_at in a past_due replay when set.
	pastDueAt *time.Time
	// cancelAtPeriodEnd is the cancel_at_period_end that snapshots report.
	cancelAtPeriodEnd bool
}

func newSeatPurchaseFixture(t *testing.T, purchased int, seatLimit *int) *seatPurchaseFixture {
	t.Helper()
	r := support.Setup(t)
	t.Cleanup(r.Close)
	db := database.DB(t.Context())

	product := "prod_team_" + uuid.NewString()
	catalog := billingmodels.BillingPlan{
		Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private",
		PerSeat: true, PriceCents: seatPriceCents, SeatLimit: seatLimit, PolarProductID: &product,
		CreditFrequency: "none", CreditExpiration: "never",
	}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &catalog))

	now := time.Now().UTC().Truncate(time.Second)
	f := &seatPurchaseFixture{
		t: t, r: r, catalog: catalog, product: product,
		subscriptionID: "sub_" + uuid.NewString(),
		periodStart:    now.Add(-7 * 24 * time.Hour),
		periodEnd:      now.Add(23 * 24 * time.Hour),
		polarSeats:     purchased,
	}
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Updates(map[string]any{
		"plan_id": catalog.ID, "plan": catalog.Key, "plan_source": billingmodels.BillingPlanSourcePolar,
		"polar_subscription_id": f.subscriptionID, "polar_subscription_status": billingmodels.PolarSubscriptionStatusActive,
		"purchased_seats": purchased, "current_period_start": f.periodStart, "current_period_end": f.periodEnd,
		"cancel_at_period_end": false,
	}).Error)
	f.onPatch = func(w http.ResponseWriter, f *seatPurchaseFixture, seats int) {
		// An applied seat update replaces the scheduled one.
		f.polarSeats = seats
		f.pendingSeats = nil
		f.writeJSON(w, http.StatusOK, f.snapshot(seats))
	}

	server := polartest.APIServer(t, f.serve)
	polartest.UseServer(t, server)
	return f
}

func (f *seatPurchaseFixture) serve(w http.ResponseWriter, req *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if req.URL.Path == "/customers/external/"+f.r.Organization.ID.String()+"/payment-methods" {
		f.writeJSON(w, http.StatusOK, map[string]any{"items": []any{}, "pagination": map[string]int{"max_page": 1}})
		return
	}
	if req.URL.Path == "/checkouts/" && req.Method == http.MethodGet {
		f.writeJSON(w, http.StatusOK, map[string]any{"items": []any{}, "pagination": map[string]int{"max_page": 1}})
		return
	}
	if req.URL.Path == "/subscriptions/" && req.Method == http.MethodGet {
		f.writeJSON(w, http.StatusOK, map[string]any{"items": []any{f.snapshot(f.polarSeats)}, "pagination": map[string]int{"max_page": 1}})
		return
	}
	if req.URL.Path != "/subscriptions/"+f.subscriptionID {
		f.t.Errorf("unexpected Polar request %s %s", req.Method, req.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	switch req.Method {
	case http.MethodGet:
		f.getCount++
		f.writeJSON(w, http.StatusOK, f.snapshot(f.polarSeats))
	case http.MethodPatch:
		var body map[string]any
		require.NoError(f.t, json.NewDecoder(req.Body).Decode(&body))
		if cancelAtPeriodEnd, ok := body["cancel_at_period_end"].(bool); ok {
			f.cancelAtPeriodEnd = cancelAtPeriodEnd
			f.writeJSON(w, http.StatusOK, f.snapshot(f.polarSeats))
			return
		}
		seats := int(body["seats"].(float64))
		if body["proration_behavior"] == "next_period" {
			f.schedules = append(f.schedules, body)
			f.writeJSON(w, http.StatusOK, f.snapshot(f.polarSeats))
			return
		}
		f.patchBodies = append(f.patchBodies, body)
		f.onPatch(w, f, seats)
	default:
		f.t.Errorf("unexpected Polar method %s", req.Method)
	}
}

func (f *seatPurchaseFixture) snapshot(seats int) map[string]any {
	modifiedAt := f.modifiedAt
	if modifiedAt.IsZero() {
		modifiedAt = time.Now()
	}
	snapshot := map[string]any{
		"id": f.subscriptionID, "product_id": f.product, "status": "active", "seats": seats,
		"cancel_at_period_end": f.cancelAtPeriodEnd, "modified_at": modifiedAt.UTC().Format(time.RFC3339Nano),
		"current_period_start": f.periodStart.Format(time.RFC3339), "current_period_end": f.periodEnd.Format(time.RFC3339),
		"customer_id": "cust_1", "external_customer_id": f.r.Organization.ID.String(),
		"pending_update": nil,
	}
	if f.pendingSeats != nil {
		snapshot["pending_update"] = map[string]any{"seats": *f.pendingSeats, "applies_at": f.periodEnd.Format(time.RFC3339)}
	}
	return snapshot
}

func (f *seatPurchaseFixture) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	require.NoError(f.t, json.NewEncoder(w).Encode(body))
}

func (f *seatPurchaseFixture) rejectWith(status int, code string) {
	f.onPatch = func(w http.ResponseWriter, f *seatPurchaseFixture, _ int) {
		f.writeJSON(w, status, map[string]any{"error": code, "detail": "details"})
	}
}

func (f *seatPurchaseFixture) dropConnection() {
	f.onPatch = func(w http.ResponseWriter, _ *seatPurchaseFixture, _ int) {
		conn, _, err := w.(http.Hijacker).Hijack()
		require.NoError(f.t, err)
		require.NoError(f.t, conn.Close())
	}
}

func (f *seatPurchaseFixture) patchCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.patchBodies)
}

// purchase sends a new purchase request with a new operation key, confirmed
// for the subscription and the period that a preview shows now.
func (f *seatPurchaseFixture) purchase(additional, expected int32) (*pb.PurchaseSeatsResponse, error) {
	return f.purchaseWithKey(uuid.NewString(), additional, expected)
}

// purchaseWithKey sends a purchase request with the given operation key,
// confirmed for the subscription and the period that a preview shows now.
func (f *seatPurchaseFixture) purchaseWithKey(key string, additional, expected int32) (*pb.PurchaseSeatsResponse, error) {
	return f.purchaseConfirmed(key, f.confirmation(), additional, expected)
}

// purchaseConfirmed sends a purchase request with the given operation key
// and confirmed preview context. A repeat of the same key and request is a
// retry of the same purchase.
func (f *seatPurchaseFixture) purchaseConfirmed(key string, confirmed seatPurchaseConfirmation, additional, expected int32) (*pb.PurchaseSeatsResponse, error) {
	return PurchaseSeats(context.Background(), f.r.Organization.ID.String(), f.request(key, confirmed, additional, expected), f.r.User.String())
}

// seatPurchaseConfirmation is the preview context that a client confirms.
type seatPurchaseConfirmation struct {
	subscriptionID string
	periodStart    time.Time
	periodEnd      time.Time
}

// confirmation returns the subscription and the billing period that a
// preview shows now: the ones in the local plan.
func (f *seatPurchaseFixture) confirmation() seatPurchaseConfirmation {
	f.t.Helper()
	plan := f.plan()
	var confirmed seatPurchaseConfirmation
	if plan.PolarSubscriptionID != nil {
		confirmed.subscriptionID = *plan.PolarSubscriptionID
	}
	if plan.CurrentPeriodStart != nil {
		confirmed.periodStart = *plan.CurrentPeriodStart
	}
	if plan.CurrentPeriodEnd != nil {
		confirmed.periodEnd = *plan.CurrentPeriodEnd
	}
	return confirmed
}

func (f *seatPurchaseFixture) request(key string, confirmed seatPurchaseConfirmation, additional, expected int32) *pb.PurchaseSeatsRequest {
	return &pb.PurchaseSeatsRequest{
		AdditionalSeats:         additional,
		ExpectedPurchasedSeats:  expected,
		OperationKey:            key,
		ConfirmedSubscriptionId: confirmed.subscriptionID,
		ConfirmedPeriodStart:    timestamppb.New(confirmed.periodStart),
		ConfirmedPeriodEnd:      timestamppb.New(confirmed.periodEnd),
	}
}

func (f *seatPurchaseFixture) preview(additional int32) (*pb.PreviewSeatPurchaseResponse, error) {
	return PreviewSeatPurchase(context.Background(), f.r.Organization.ID.String(), &pb.PreviewSeatPurchaseRequest{AdditionalSeats: additional})
}

// purchaseAt runs a purchase with a controlled clock.
func (f *seatPurchaseFixture) purchaseAt(clock seatPurchaseClock, additional, expected int32) (*pb.PurchaseSeatsResponse, error) {
	return purchaseSeats(context.Background(), f.r.Organization.ID.String(),
		f.request(uuid.NewString(), f.confirmation(), additional, expected), f.r.User.String(), clock)
}

// setPeriodEnd moves the end of the billing period, both locally and in Polar.
func (f *seatPurchaseFixture) setPeriodEnd(periodEnd time.Time) {
	f.t.Helper()
	f.mu.Lock()
	f.periodEnd = periodEnd
	f.mu.Unlock()
	f.updateBillingPlan(map[string]any{"current_period_end": periodEnd})
}

// renew starts the next billing period in Polar with the given seats, as when
// Polar renews with a scheduled seat count. The local plan keeps the old
// period until a webhook or a refresh applies the new one.
func (f *seatPurchaseFixture) renew(seats int) {
	f.t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	f.periodStart = time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	f.periodEnd = f.periodStart.Add(30 * 24 * time.Hour)
	f.polarSeats = seats
	f.pendingSeats = nil
}

// replaceSubscription replaces the subscription with another one that has
// the same seats and period, both locally and in Polar.
func (f *seatPurchaseFixture) replaceSubscription() {
	f.t.Helper()
	f.mu.Lock()
	f.subscriptionID = "sub_" + uuid.NewString()
	subscriptionID := f.subscriptionID
	f.mu.Unlock()
	f.updateBillingPlan(map[string]any{"polar_subscription_id": subscriptionID})
}

func (f *seatPurchaseFixture) scheduleCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.schedules)
}

func fixedSeatPurchaseClock(now time.Time) seatPurchaseClock {
	return func() time.Time { return now }
}

func (f *seatPurchaseFixture) plan() *billingmodels.OrganizationBillingPlan {
	f.t.Helper()
	plan, err := billingmodels.FindOrganizationBillingPlan(database.Conn(), f.r.Organization.ID)
	require.NoError(f.t, err)
	return plan
}

func (f *seatPurchaseFixture) purchasedSeats() int {
	f.t.Helper()
	plan := f.plan()
	require.NotNil(f.t, plan.PurchasedSeats)
	return *plan.PurchasedSeats
}

func (f *seatPurchaseFixture) latestPurchase() *billingmodels.SeatPurchase {
	f.t.Helper()
	purchase, err := billingmodels.FindLatestSeatPurchase(database.Conn(), f.r.Organization.ID)
	require.NoError(f.t, err)
	require.NotNil(f.t, purchase)
	return purchase
}

func (f *seatPurchaseFixture) updateBillingPlan(values map[string]any) {
	f.t.Helper()
	require.NoError(f.t, database.Conn().Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", f.r.Organization.ID).Updates(values).Error)
}

func (f *seatPurchaseFixture) applyWebhook(seats int, modifiedAt time.Time) {
	f.t.Helper()
	data := f.snapshot(seats)
	data["modified_at"] = modifiedAt.UTC().Format(time.RFC3339Nano)
	f.applySubscriptionEvent(data)
}

// applySubscriptionEvent delivers a subscription.updated webhook with the
// snapshot data.
func (f *seatPurchaseFixture) applySubscriptionEvent(data map[string]any) {
	f.t.Helper()
	body, err := json.Marshal(map[string]any{"type": "subscription.updated", "data": data})
	require.NoError(f.t, err)
	var event polar.SubscriptionWebhookEvent
	require.NoError(f.t, json.Unmarshal(body, &event))
	require.NoError(f.t, polar.ApplySubscriptionEvent(context.Background(), database.Conn(), &event))
}

func requireGRPCCode(t *testing.T, want codes.Code, err error) {
	t.Helper()
	require.Error(t, err)
	require.Equal(t, want, grpcerrors.Code(err), err.Error())
}

func Test__ProratedChargeCentsFollowsThePolarFormula(t *testing.T) {
	now := time.Now()
	start := now.Add(-7 * 24 * time.Hour)
	end := start.Add(30 * 24 * time.Hour)

	assert.EqualValues(t, 1533, proratedChargeCents(1, 2000, start, end, now), "23 of 30 days of a $20 seat")
	assert.EqualValues(t, 4600, proratedChargeCents(3, 2000, start, end, now))
	assert.EqualValues(t, 2000, proratedChargeCents(1, 2000, start, end, start))
	assert.EqualValues(t, 1, proratedChargeCents(1, 2000, start, end, end.Add(-12*time.Minute)), "0.56 cents rounds up")
	assert.EqualValues(t, 0, proratedChargeCents(1, 2000, start, end, end.Add(-5*time.Minute)), "0.23 cents rounds down")
	assert.EqualValues(t, 0, proratedChargeCents(1, 2000, start, end, end))
}

func Test__PreviewSeatPurchaseEstimatesWithoutChangingAnything(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))

	preview, err := f.preview(1)
	require.NoError(t, err)

	assert.EqualValues(t, 3, preview.PurchasedSeats)
	assert.EqualValues(t, 4, preview.TargetSeats)
	assert.EqualValues(t, seatPriceCents, preview.SeatPriceCents)
	assert.EqualValues(t, 1533, preview.ProratedChargeCents, "23 of 30 days of a $20 seat")
	assert.EqualValues(t, 4*seatPriceCents, preview.NewMonthlyTotalCents)
	assert.Equal(t, f.subscriptionID, preview.SubscriptionId)
	assert.True(t, preview.BillingPeriodStart.AsTime().Equal(f.periodStart))
	assert.True(t, preview.PeriodEnd.AsTime().Equal(f.periodEnd))
	assert.WithinDuration(t, time.Now(), preview.PeriodStart.AsTime(), time.Minute, "period_start stays the coverage start")
	require.NotNil(t, preview.PurchasableSeats)
	assert.EqualValues(t, 47, *preview.PurchasableSeats)
	assert.True(t, preview.Estimate)
	assert.True(t, preview.ExcludesTax)

	assert.Zero(t, f.patchCount())
	assert.Equal(t, 3, f.purchasedSeats())
	latest, err := billingmodels.FindLatestSeatPurchase(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)
	assert.Nil(t, latest)
}

func Test__PurchaseSeatsChargesOnceAndAddsCapacityAfterPolarConfirms(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))

	response, err := f.purchase(1, 3)
	require.NoError(t, err)

	require.Equal(t, 1, f.patchCount())
	assert.Equal(t, map[string]any{"seats": float64(4), "proration_behavior": "invoice"}, f.patchBodies[0])
	assert.Equal(t, 4, f.purchasedSeats())
	assert.True(t, f.plan().CurrentPeriodEnd.Equal(f.periodEnd), "the renewal date does not change")

	assert.Equal(t, billingmodels.SeatPurchaseStatusSucceeded, response.Purchase.Status)
	assert.EqualValues(t, 3, response.Purchase.FromSeats)
	assert.EqualValues(t, 4, response.Purchase.ToSeats)
	assert.Empty(t, response.Purchase.Message)
	require.NotNil(t, response.Billing)
	assert.EqualValues(t, 4, response.Billing.GetPurchasedSeats())
	assert.Equal(t, response.Purchase.Id, response.Billing.LastSeatPurchase.Id)

	stored := f.latestPurchase()
	require.NotNil(t, stored.UserID)
	assert.Equal(t, f.r.User, *stored.UserID)

	expected, err := billingmodels.FindExpectedSeatCount(database.Conn(), f.r.Organization.ID, f.subscriptionID, f.periodStart)
	require.NoError(t, err)
	require.NotNil(t, expected)
	assert.Equal(t, 4, *expected, "an outside-change alert can recognize this purchase")
}

func Test__PurchaseSeatsRejectsRepeatedSubmission(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))

	_, err := f.purchase(1, 3)
	require.NoError(t, err)
	_, err = f.purchase(1, 3)

	requireGRPCCode(t, codes.Aborted, err)
	assert.Equal(t, 1, f.patchCount())
	assert.Equal(t, 4, f.purchasedSeats())
}

func Test__PurchaseSeatsRejectsAStaleSeatCount(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))

	_, err := f.purchase(1, 2)

	requireGRPCCode(t, codes.Aborted, err)
	assert.ErrorContains(t, err, "Seat count changed. Refresh and try again.")
	assert.Zero(t, f.patchCount())
}

func Test__PurchaseSeatsAboveTheMaximumNeverReachesPolar(t *testing.T) {
	tests := []struct {
		name      string
		purchased int
		limit     *int
		override  *int
		hasOver   bool
		additions int32
		message   string
	}{
		{name: "above the catalog maximum", purchased: 3, limit: new(5), additions: 3, message: "The Team plan allows up to 5 seats."},
		{name: "outside change left no room", purchased: 6, limit: new(5), additions: 1, message: "The Team plan allows up to 5 seats."},
		{name: "organization override replaces the catalog maximum", purchased: 3, limit: new(50), override: new(4), hasOver: true, additions: 2, message: "The Team plan allows up to 4 seats."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSeatPurchaseFixture(t, tt.purchased, tt.limit)
			if tt.hasOver {
				require.NoError(t, billingmodels.SetOrganizationSeatLimitOverride(database.Conn(), f.r.Organization.ID, tt.override))
			}

			_, err := f.purchase(tt.additions, int32(tt.purchased))
			requireGRPCCode(t, codes.InvalidArgument, err)
			assert.ErrorContains(t, err, tt.message)
			_, err = f.preview(tt.additions)
			requireGRPCCode(t, codes.InvalidArgument, err)

			assert.Zero(t, f.patchCount())
			assert.Equal(t, tt.purchased, f.purchasedSeats())
		})
	}
}

func Test__PurchaseSeatsAllowsAnyCountWithAnUnlimitedOverride(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(5))
	require.NoError(t, billingmodels.SetOrganizationSeatLimitOverride(database.Conn(), f.r.Organization.ID, nil))

	_, err := f.purchase(100, 3)

	require.NoError(t, err)
	assert.Equal(t, 103, f.purchasedSeats())
}

func Test__PurchaseSeatsRequiresAtLeastOneSeat(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))

	_, err := f.purchase(0, 3)

	requireGRPCCode(t, codes.InvalidArgument, err)
	assert.Zero(t, f.patchCount())
}

func Test__PurchaseSeatsRejectsSubscriptionsThatCannotTakeSeats(t *testing.T) {
	tests := []struct {
		name    string
		change  func(f *seatPurchaseFixture)
		message string
	}{
		{name: "plan is not per seat", change: func(f *seatPurchaseFixture) {
			product := "prod_flat_" + uuid.NewString()
			flat := billingmodels.BillingPlan{Key: "flat_" + uuid.NewString()[:8], Name: "Flat", Category: "paid", Visibility: "private", PriceCents: seatPriceCents, PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
			require.NoError(t, billingmodels.SaveBillingPlan(database.Conn(), &flat))
			f.updateBillingPlan(map[string]any{"plan_id": flat.ID, "plan": flat.Key})
		}, message: "Seat purchases need an active Team subscription."},
		{name: "admin plan", change: func(f *seatPurchaseFixture) {
			f.updateBillingPlan(map[string]any{"plan_source": billingmodels.BillingPlanSourceAdmin})
		}, message: "Seat purchases need an active Team subscription."},
		{name: "subscription is not active", change: func(f *seatPurchaseFixture) {
			f.updateBillingPlan(map[string]any{"polar_subscription_status": billingmodels.PolarSubscriptionStatusPastDue})
		}, message: "Seat purchases need an active Team subscription."},
		{name: "no subscription id", change: func(f *seatPurchaseFixture) {
			f.updateBillingPlan(map[string]any{"polar_subscription_id": nil})
		}, message: "Seat purchases need an active Team subscription."},
		{name: "subscription is set to cancel", change: func(f *seatPurchaseFixture) {
			f.updateBillingPlan(map[string]any{"cancel_at_period_end": true})
		}, message: "Resume the subscription before buying seats."},
		{name: "period has ended", change: func(f *seatPurchaseFixture) {
			f.updateBillingPlan(map[string]any{"current_period_end": time.Now().Add(-time.Minute)})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSeatPurchaseFixture(t, 3, new(50))
			// The user confirmed a preview before the subscription changed.
			confirmed := f.confirmation()
			tt.change(f)

			_, err := f.purchaseConfirmed(uuid.NewString(), confirmed, 1, 3)
			requireGRPCCode(t, codes.FailedPrecondition, err)
			if tt.message != "" {
				assert.ErrorContains(t, err, tt.message)
			}
			_, err = f.preview(1)
			requireGRPCCode(t, codes.FailedPrecondition, err)

			assert.Zero(t, f.patchCount())
			assert.Equal(t, 0, f.getCount)
		})
	}
}

func Test__PurchaseSeatsRejectedByPolarLeavesCapacityUnchanged(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		polarCode  string
		wantCode   codes.Code
		wantStatus string
		message    string
	}{
		{"payment failed", http.StatusPaymentRequired, "PaymentFailed", codes.FailedPrecondition, billingmodels.SeatPurchaseStatusFailed,
			"The payment failed. No seats were added."},
		{"payment needs bank confirmation", http.StatusPaymentRequired, "PaymentActionRequired", codes.FailedPrecondition, billingmodels.SeatPurchaseStatusActionRequired,
			"Your bank needs to confirm this payment. Update the payment method and try again."},
		{"subscription locked", http.StatusConflict, "SubscriptionLocked", codes.Aborted, billingmodels.SeatPurchaseStatusFailed,
			"Try again shortly."},
		{"subscription canceled", http.StatusForbidden, "AlreadyCanceledSubscription", codes.FailedPrecondition, billingmodels.SeatPurchaseStatusFailed,
			"The subscription cannot add seats. Check that it is active and not set to cancel."},
		{"subscription inactive", http.StatusForbidden, "InactiveSubscription", codes.FailedPrecondition, billingmodels.SeatPurchaseStatusFailed,
			"The subscription cannot add seats. Check that it is active and not set to cancel."},
		{"seat count rejected", http.StatusUnprocessableEntity, "HTTPValidationError", codes.InvalidArgument, billingmodels.SeatPurchaseStatusFailed,
			"Polar did not accept the seat purchase. No seats were added."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSeatPurchaseFixture(t, 3, new(50))
			f.rejectWith(tt.status, tt.polarCode)

			_, err := f.purchase(1, 3)

			requireGRPCCode(t, tt.wantCode, err)
			assert.ErrorContains(t, err, tt.message)
			assert.NotContains(t, err.Error(), "scope")
			assert.Equal(t, 3, f.purchasedSeats())
			stored := f.latestPurchase()
			assert.Equal(t, tt.wantStatus, stored.Status)
			assert.Equal(t, 1, f.patchCount())

			f.onPatch = func(w http.ResponseWriter, f *seatPurchaseFixture, seats int) {
				f.polarSeats = seats
				f.writeJSON(w, http.StatusOK, f.snapshot(seats))
			}
			_, err = f.purchase(1, 3)
			require.NoError(t, err, "a settled failure does not block a new purchase")
			assert.Equal(t, 4, f.purchasedSeats())
		})
	}
}

func Test__PurchaseSeatsReportsAMissingPolarScopeForUnrelatedForbiddenErrors(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.rejectWith(http.StatusForbidden, "NotPermitted")

	_, err := f.purchase(1, 3)

	requireGRPCCode(t, codes.FailedPrecondition, err)
	assert.ErrorContains(t, err, "missing a required Polar scope")
	assert.Equal(t, billingmodels.SeatPurchaseStatusFailed, f.latestPurchase().Status)
}

func Test__PurchaseSeatsWithoutAnAnswerWaitsForConfirmation(t *testing.T) {
	tests := map[string]func(f *seatPurchaseFixture){
		"connection drops": func(f *seatPurchaseFixture) { f.dropConnection() },
		"server error":     func(f *seatPurchaseFixture) { f.rejectWith(http.StatusGatewayTimeout, "") },
	}
	for name, fail := range tests {
		t.Run(name, func(t *testing.T) {
			f := newSeatPurchaseFixture(t, 3, new(50))
			fail(f)

			response, err := f.purchase(1, 3)

			require.NoError(t, err)
			assert.Equal(t, "awaiting_confirmation", response.Purchase.Status)
			assert.NotEmpty(t, response.Purchase.Message)
			assert.Equal(t, 3, f.purchasedSeats(), "capacity changes only from a Polar snapshot")
			assert.Equal(t, billingmodels.SeatPurchaseStatusUnknown, f.latestPurchase().Status)
			assert.EqualValues(t, 3, response.Billing.GetPurchasedSeats())
			assert.Equal(t, "awaiting_confirmation", response.Billing.LastSeatPurchase.Status)
		})
	}
}

func Test__WebhookAddsCapacityForAnUnconfirmedPurchase(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.dropConnection()
	_, err := f.purchase(1, 3)
	require.NoError(t, err)

	f.applyWebhook(4, time.Now())

	assert.Equal(t, 4, f.purchasedSeats())
	assert.Equal(t, billingmodels.SeatPurchaseStatusSucceeded, f.latestPurchase().Status)
	assert.Equal(t, 1, f.patchCount())
}

// The renewal schedule read 3 seats before the purchase was recorded. Its
// next_period update reaches Polar after the purchase was recorded, and the
// webhook of that update arrives late. Polar then charges the purchase, but
// the response is lost. The delayed event reports the original seats after
// the purchase, but it is about another update, so it proves nothing.
func Test__DelayedScheduleEventLeavesALostPurchaseUnknownUntilPolarProvesIt(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	var delayedSchedule map[string]any
	f.onPatch = func(w http.ResponseWriter, f *seatPurchaseFixture, seats int) {
		f.pendingSeats = new(1)
		delayedSchedule = f.snapshot(f.polarSeats)
		f.polarSeats, f.pendingSeats = seats, nil
		conn, _, err := w.(http.Hijacker).Hijack()
		require.NoError(f.t, err)
		require.NoError(f.t, conn.Close())
	}
	key := uuid.NewString()
	confirmed := f.confirmation()

	first, err := f.purchaseConfirmed(key, confirmed, 2, 3)
	require.NoError(t, err)
	require.Equal(t, seatPurchaseAwaitingConfirmation, first.Purchase.Status)

	f.applySubscriptionEvent(delayedSchedule)

	stored := f.latestPurchase()
	assert.Equal(t, first.Purchase.Id, stored.ID.String())
	assert.Equal(t, billingmodels.SeatPurchaseStatusUnknown, stored.Status, "the delayed schedule event must not fail the purchase")
	assert.Empty(t, stored.ErrorCode)
	assert.Equal(t, 3, f.purchasedSeats(), "capacity follows the applied snapshot")

	// The next read from Polar shows the target in the purchase period after
	// the purchase was recorded.
	_, err = f.preview(1)
	require.NoError(t, err)

	stored = f.latestPurchase()
	assert.Equal(t, first.Purchase.Id, stored.ID.String(), "the original operation settles")
	assert.Equal(t, billingmodels.SeatPurchaseStatusSucceeded, stored.Status)
	assert.Equal(t, 5, f.purchasedSeats())
	assert.Equal(t, 1, f.patchCount(), "the settlement does not charge again")

	// A duplicate delivery of the delayed event does not replace the outcome.
	f.applySubscriptionEvent(delayedSchedule)
	replay, err := f.purchaseConfirmed(key, confirmed, 2, 3)
	require.NoError(t, err)
	assert.Equal(t, first.Purchase.Id, replay.Purchase.Id)
	assert.Equal(t, billingmodels.SeatPurchaseStatusSucceeded, replay.Purchase.Status)
	assert.Equal(t, 5, f.purchasedSeats())
	assert.Equal(t, 1, f.patchCount(), "the replay does not charge again")
}

func Test__RetryAfterAnUnconfirmedPurchaseDoesNotChargeAgain(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.dropConnection()
	_, err := f.purchase(1, 3)
	require.NoError(t, err)
	f.polarSeats = 4 // Polar applied the update even though the answer was lost.

	_, err = f.purchase(1, 3)

	requireGRPCCode(t, codes.Aborted, err)
	assert.Equal(t, 1, f.patchCount(), "the retry must not send a second update")
	assert.Zero(t, f.scheduleCount(), "reconciliation schedules no seat count")
	assert.Equal(t, 4, f.purchasedSeats())
	assert.Equal(t, billingmodels.SeatPurchaseStatusSucceeded, f.latestPurchase().Status)
}

func Test__PurchaseSeatsRequiresAValidOperationKey(t *testing.T) {
	tests := map[string]string{
		"missing":         "",
		"with a space":    "key with space",
		"non-ASCII":       "schlüssel",
		"control char":    "key\n",
		"longer than 128": strings.Repeat("k", 129),
	}
	for name, key := range tests {
		t.Run(name, func(t *testing.T) {
			f := newSeatPurchaseFixture(t, 3, new(50))

			_, err := f.purchaseWithKey(key, 1, 3)

			requireGRPCCode(t, codes.InvalidArgument, err)
			assert.Zero(t, f.patchCount())
			assert.Zero(t, f.getCount, "the request never reaches Polar")
			latest, findErr := billingmodels.FindLatestSeatPurchase(database.Conn(), f.r.Organization.ID)
			require.NoError(t, findErr)
			assert.Nil(t, latest)
		})
	}
}

func Test__PurchaseSeatsStoresTheOperationAndThePeriod(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	key := strings.Repeat("k", 128)

	_, err := f.purchaseWithKey(key, 1, 3)
	require.NoError(t, err)

	stored := f.latestPurchase()
	require.NotNil(t, stored.OperationKey)
	assert.Equal(t, key, *stored.OperationKey)
	require.NotNil(t, stored.RequestFingerprint)
	assert.Equal(t, billingmodels.SeatPurchaseBoundFingerprint(1, 3, f.subscriptionID, f.periodStart, f.periodEnd), *stored.RequestFingerprint)
	assert.Equal(t, f.subscriptionID, stored.PolarSubscriptionID)
	require.NotNil(t, stored.PeriodStart)
	require.NotNil(t, stored.PeriodEnd)
	assert.True(t, stored.PeriodStart.Equal(f.periodStart))
	assert.True(t, stored.PeriodEnd.Equal(f.periodEnd))
}

func Test__PurchaseSeatsReplayReturnsTheStoredPurchaseWithoutCharging(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	key := uuid.NewString()
	confirmed := f.confirmation()
	first, err := f.purchaseConfirmed(key, confirmed, 1, 3)
	require.NoError(t, err)

	replay, err := f.purchaseConfirmed(key, confirmed, 1, 3)
	require.NoError(t, err)
	assert.Equal(t, first.Purchase.Id, replay.Purchase.Id)
	assert.Equal(t, billingmodels.SeatPurchaseStatusSucceeded, replay.Purchase.Status)
	assert.Equal(t, 1, f.patchCount())

	// Polar renews and applies a scheduled reduction back to 3 seats. The
	// request matches the new seat count again, but it is the same purchase.
	f.renew(3)
	f.applyWebhook(3, time.Now())

	replay, err = f.purchaseConfirmed(key, confirmed, 1, 3)
	require.NoError(t, err)
	assert.Equal(t, first.Purchase.Id, replay.Purchase.Id)
	assert.Equal(t, 1, f.patchCount(), "a replay after the renewal does not charge again")
	assert.Equal(t, 3, f.purchasedSeats())

	// Another subscription replaces the confirmed one.
	f.replaceSubscription()

	replay, err = f.purchaseConfirmed(key, confirmed, 1, 3)
	require.NoError(t, err)
	assert.Equal(t, first.Purchase.Id, replay.Purchase.Id)
	assert.Equal(t, 1, f.patchCount(), "a replay after the replacement does not charge again")
}

func Test__PurchaseSeatsReplaysALegacyOperationByItsSeatCounts(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	key := uuid.NewString()
	// A purchase recorded before the binding has only the v1 fingerprint.
	legacy := billingmodels.NewSeatPurchase(f.r.Organization.ID, &f.r.User, f.subscriptionID, 3, 4, &billingmodels.SeatPurchaseOperation{
		Key: key, Fingerprint: billingmodels.SeatPurchaseFingerprint(1, 3), PeriodStart: f.periodStart, PeriodEnd: f.periodEnd,
	})
	legacy.Status = billingmodels.SeatPurchaseStatusSucceeded
	require.NoError(t, legacy.Create(database.DB(t.Context())))

	for name, request := range map[string]*pb.PurchaseSeatsRequest{
		"without a binding": {AdditionalSeats: 1, ExpectedPurchasedSeats: 3, OperationKey: key},
		"with a binding":    f.request(key, f.confirmation(), 1, 3),
	} {
		t.Run(name, func(t *testing.T) {
			replay, err := PurchaseSeats(t.Context(), f.r.Organization.ID.String(), request, f.r.User.String())

			require.NoError(t, err)
			assert.Equal(t, legacy.ID.String(), replay.Purchase.Id)
			assert.Zero(t, f.patchCount())
			assert.Zero(t, f.getCount, "a replay never reaches Polar")
		})
	}

	_, err := PurchaseSeats(t.Context(), f.r.Organization.ID.String(),
		&pb.PurchaseSeatsRequest{AdditionalSeats: 2, ExpectedPurchasedSeats: 3, OperationKey: key}, f.r.User.String())
	requireGRPCCode(t, codes.FailedPrecondition, err)
	assert.ErrorContains(t, err, "This operation key was used for another seat purchase.")
}

func Test__PurchaseSeatsWithoutACompleteConfirmationNeverReachesPolar(t *testing.T) {
	tests := map[string]func(req *pb.PurchaseSeatsRequest){
		"no confirmation": func(req *pb.PurchaseSeatsRequest) {
			req.ConfirmedSubscriptionId, req.ConfirmedPeriodStart, req.ConfirmedPeriodEnd = "", nil, nil
		},
		"no subscription": func(req *pb.PurchaseSeatsRequest) { req.ConfirmedSubscriptionId = "" },
		"no period start": func(req *pb.PurchaseSeatsRequest) { req.ConfirmedPeriodStart = nil },
		"no period end":   func(req *pb.PurchaseSeatsRequest) { req.ConfirmedPeriodEnd = nil },
		"invalid time":    func(req *pb.PurchaseSeatsRequest) { req.ConfirmedPeriodEnd = &timestamppb.Timestamp{Nanos: -1} },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			f := newSeatPurchaseFixture(t, 3, new(50))
			req := f.request(uuid.NewString(), f.confirmation(), 1, 3)
			change(req)

			_, err := PurchaseSeats(t.Context(), f.r.Organization.ID.String(), req, f.r.User.String())

			requireGRPCCode(t, codes.InvalidArgument, err)
			assert.ErrorContains(t, err, "Preview the purchase again and confirm it.")
			assert.Zero(t, f.patchCount())
			assert.Zero(t, f.getCount, "the request never reaches Polar")
			latest, findErr := billingmodels.FindLatestSeatPurchase(database.Conn(), f.r.Organization.ID)
			require.NoError(t, findErr)
			assert.Nil(t, latest)
		})
	}
}

func Test__PurchaseSeatsRejectsAStaleConfirmationWithTheSameSeatCounts(t *testing.T) {
	tests := map[string]func(f *seatPurchaseFixture){
		"renewal with the same seats": func(f *seatPurchaseFixture) {
			f.renew(3)
			f.applyWebhook(3, time.Now())
		},
		"replacement subscription with the same counts and dates": func(f *seatPurchaseFixture) {
			f.replaceSubscription()
		},
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			f := newSeatPurchaseFixture(t, 3, new(50))
			stale := f.confirmation()
			change(f)

			_, err := f.purchaseConfirmed(uuid.NewString(), stale, 1, 3)

			requireGRPCCode(t, codes.Aborted, err)
			assert.ErrorContains(t, err, "The subscription or billing period changed. Review the new price and confirm again.")
			assert.Zero(t, f.patchCount())
			latest, findErr := billingmodels.FindLatestSeatPurchase(database.Conn(), f.r.Organization.ID)
			require.NoError(t, findErr)
			assert.Nil(t, latest, "no attempt is recorded")

			// A fresh preview and confirmation buys the seats.
			_, err = f.purchase(1, 3)
			require.NoError(t, err)
			assert.Equal(t, 1, f.patchCount())
			assert.Equal(t, f.subscriptionID, f.latestPurchase().PolarSubscriptionID)
		})
	}
}

func Test__PurchaseSeatsRejectsAKeyReusedForAnotherRequest(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	key := uuid.NewString()
	_, err := f.purchaseWithKey(key, 1, 3)
	require.NoError(t, err)

	confirmed := f.confirmation()
	otherSubscription, otherStart, otherEnd := confirmed, confirmed, confirmed
	otherSubscription.subscriptionID = "sub_other"
	otherStart.periodStart = confirmed.periodStart.Add(time.Second)
	otherEnd.periodEnd = confirmed.periodEnd.Add(time.Second)
	for name, request := range map[string]*pb.PurchaseSeatsRequest{
		"more seats from another count": f.request(key, confirmed, 2, 4),
		"another expected count":        f.request(key, confirmed, 1, 4),
		"more seats":                    f.request(key, confirmed, 2, 3),
		"another subscription":          f.request(key, otherSubscription, 1, 3),
		"another period start":          f.request(key, otherStart, 1, 3),
		"another period end":            f.request(key, otherEnd, 1, 3),
		"no binding":                    {AdditionalSeats: 1, ExpectedPurchasedSeats: 3, OperationKey: key},
	} {
		_, err = PurchaseSeats(t.Context(), f.r.Organization.ID.String(), request, f.r.User.String())

		requireGRPCCode(t, codes.FailedPrecondition, err)
		assert.ErrorContains(t, err, "This operation key was used for another seat purchase.", name)
	}
	assert.Equal(t, 1, f.patchCount())
	assert.Equal(t, 4, f.purchasedSeats())
}

func Test__ConcurrentIdenticalPurchasesRecordOnePurchase(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.onPatch = func(w http.ResponseWriter, f *seatPurchaseFixture, seats int) {
		time.Sleep(200 * time.Millisecond)
		f.polarSeats = seats
		f.writeJSON(w, http.StatusOK, f.snapshot(seats))
	}
	key := uuid.NewString()

	start := make(chan struct{})
	type result struct {
		response *pb.PurchaseSeatsResponse
		err      error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			<-start
			response, err := f.purchaseWithKey(key, 1, 3)
			results <- result{response, err}
		}()
	}
	close(start)

	ids := map[string]bool{}
	for range 2 {
		got := <-results
		require.NoError(t, got.err)
		ids[got.response.Purchase.Id] = true
	}
	assert.Len(t, ids, 1, "both requests describe the same purchase")
	var count int64
	require.NoError(t, database.Conn().Model(&billingmodels.SeatPurchase{}).Where("organization_id = ?", f.r.Organization.ID).Count(&count).Error)
	assert.EqualValues(t, 1, count)
	assert.LessOrEqual(t, f.patchCount(), 1)
}

func Test__ConcurrentIdenticalPurchaseReplaysBeforeReturningPreLockErrors(t *testing.T) {
	for _, test := range []struct {
		name          string
		seatLimit     int
		rejectRefresh bool
	}{
		{name: "purchase reaches the seat limit", seatLimit: 5},
		{name: "Polar refresh fails", seatLimit: 50, rejectRefresh: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newSeatPurchaseFixture(t, 3, new(test.seatLimit))
			var rejectRefresh atomic.Bool
			server := polartest.APIServer(t, func(w http.ResponseWriter, req *http.Request) {
				if rejectRefresh.Load() && req.Method == http.MethodGet && req.URL.Path == "/subscriptions/"+f.subscriptionID {
					http.Error(w, "Polar unavailable", http.StatusServiceUnavailable)
					return
				}
				f.serve(w, req)
			})
			polartest.UseServer(t, server)
			key := uuid.NewString()
			confirmed := f.confirmation()

			// Pause after the initial operation lookup, before the local check.
			checking := make(chan struct{})
			resume := make(chan struct{})
			var release sync.Once
			var pause sync.Once
			done := make(chan struct{})
			t.Cleanup(func() {
				release.Do(func() { close(resume) })
				<-done
			})
			var replay *pb.PurchaseSeatsResponse
			var replayErr error
			go func() {
				defer close(done)
				replay, replayErr = purchaseSeats(t.Context(), f.r.Organization.ID.String(),
					f.request(key, confirmed, 2, 3), f.r.User.String(), func() time.Time {
						pause.Do(func() {
							close(checking)
							<-resume
						})
						return time.Now()
					})
			}()
			<-checking

			first, err := f.purchaseWithKey(key, 2, 3)
			require.NoError(t, err)
			rejectRefresh.Store(test.rejectRefresh)
			release.Do(func() { close(resume) })
			<-done

			require.NoError(t, replayErr)
			require.NotNil(t, replay)
			assert.Equal(t, first.Purchase.Id, replay.Purchase.Id)
			assert.Equal(t, billingmodels.SeatPurchaseStatusSucceeded, replay.Purchase.Status)
			assert.Equal(t, 1, f.patchCount(), "the replay must not send a second purchase")
			assert.Equal(t, 5, f.purchasedSeats())
		})
	}
}

func Test__ReplayAfterALostResponseAndARenewalReductionDoesNotChargeAgain(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	// Polar applies and charges the purchase, but the response is lost.
	f.onPatch = func(w http.ResponseWriter, f *seatPurchaseFixture, seats int) {
		f.polarSeats = seats
		conn, _, err := w.(http.Hijacker).Hijack()
		require.NoError(f.t, err)
		require.NoError(f.t, conn.Close())
	}
	key := uuid.NewString()
	confirmed := f.confirmation()
	first, err := f.purchaseConfirmed(key, confirmed, 1, 3)
	require.NoError(t, err)
	assert.Equal(t, seatPurchaseAwaitingConfirmation, first.Purchase.Status)

	// The renewal applies a scheduled reduction back to the original 3 seats.
	f.renew(3)
	f.applyWebhook(3, time.Now())

	replay, err := f.purchaseConfirmed(key, confirmed, 1, 3)

	require.NoError(t, err)
	assert.Equal(t, first.Purchase.Id, replay.Purchase.Id)
	assert.Equal(t, seatPurchaseAwaitingConfirmation, replay.Purchase.Status)
	assert.Equal(t, 1, f.patchCount(), "the replay sends no second purchase")
	var count int64
	require.NoError(t, database.Conn().Model(&billingmodels.SeatPurchase{}).Where("organization_id = ?", f.r.Organization.ID).Count(&count).Error)
	assert.EqualValues(t, 1, count)
}

// After a renewal, the seat count of the new period cannot prove whether
// Polar charged an unknown purchase. The purchase keeps waiting for a person
// to resolve it, and it keeps blocking new purchases.
func Test__UnknownPurchaseAcrossARenewalKeepsAwaitingConfirmation(t *testing.T) {
	for _, renewedSeats := range []int{3, 4, 7} {
		t.Run(fmt.Sprintf("renewed with %d seats", renewedSeats), func(t *testing.T) {
			f := newSeatPurchaseFixture(t, 3, new(50))
			f.dropConnection()
			_, err := f.purchase(1, 3)
			require.NoError(t, err)
			unknown := f.latestPurchase()
			stale := time.Now().Add(-2 * polar.SeatPurchaseConfirmationWindow)
			require.NoError(t, database.Conn().Model(&billingmodels.SeatPurchase{}).Where("id = ?", unknown.ID).Update("created_at", stale).Error)
			f.renew(renewedSeats)

			requireAwaitingConfirmation := func(path string) {
				t.Helper()
				stored := f.latestPurchase()
				assert.Equal(t, unknown.ID, stored.ID)
				assert.Equal(t, billingmodels.SeatPurchaseStatusUnknown, stored.Status, "%s must not settle the purchase", path)
			}

			f.applyWebhook(renewedSeats, time.Now())
			requireAwaitingConfirmation("the webhook")
			assert.Equal(t, renewedSeats, f.purchasedSeats(), "the renewal still updates the plan")

			require.NoError(t, polar.ReconcileSeatPurchase(t.Context(), database.Conn(), polar.NewClientFromEnv(), f.r.Organization.ID))
			requireAwaitingConfirmation("reconciliation")

			require.NoError(t, polar.RefreshSeatPurchaseState(t.Context(), database.Conn(), polar.NewClientFromEnv(), f.r.Organization.ID, f.subscriptionID))
			requireAwaitingConfirmation("the refresh")

			billing, err := SyncOrganizationBilling(t.Context(), f.r.Organization.ID.String(), &pb.SyncOrganizationBillingRequest{})
			require.NoError(t, err)
			requireAwaitingConfirmation("the billing sync")
			assert.Equal(t, seatPurchaseAwaitingConfirmation, billing.LastSeatPurchase.Status)

			_, err = f.purchase(1, int32(renewedSeats))
			requireGRPCCode(t, codes.Aborted, err)
			assert.ErrorContains(t, err, "A seat purchase is already in progress.")
			assert.Equal(t, 1, f.patchCount(), "a new key does not charge while the purchase is unknown")
			requireAwaitingConfirmation("a new purchase")
		})
	}
}

func Test__PurchaseSeatsKeepsSeatsAddedElsewhere(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.polarSeats = 6 // an admin added seats in Polar before the webhook arrived

	_, err := f.purchase(1, 3)

	requireGRPCCode(t, codes.Aborted, err)
	assert.ErrorContains(t, err, "Seat count changed. Refresh and try again.")
	assert.Zero(t, f.patchCount(), "no update may send an absolute count below the Polar seats")
	assert.Equal(t, 6, f.purchasedSeats(), "the Polar snapshot refreshes the local count")

	_, err = f.purchase(1, 6)
	require.NoError(t, err)
	require.Equal(t, 1, f.patchCount())
	assert.Equal(t, map[string]any{"seats": float64(7), "proration_behavior": "invoice"}, f.patchBodies[0])
	assert.Equal(t, 7, f.purchasedSeats())
}

// Neither the time since an unknown purchase nor a seat count below its
// target proves that Polar did not charge. The purchase stays unknown and
// keeps blocking new purchases. The verified snapshot still repairs the
// capacity.
func Test__UnknownPurchaseWithoutEvidenceBlocksNewPurchasesWhileCapacityIsRepaired(t *testing.T) {
	for name, polarSeats := range map[string]int{
		"original seats":                  3,
		"outside change below the target": 4,
	} {
		t.Run(name, func(t *testing.T) {
			f := newSeatPurchaseFixture(t, 3, new(50))
			f.dropConnection()
			_, err := f.purchase(2, 3)
			require.NoError(t, err)
			unknown := f.latestPurchase()
			stale := time.Now().Add(-2 * polar.SeatPurchaseConfirmationWindow)
			require.NoError(t, database.Conn().Model(&billingmodels.SeatPurchase{}).Where("id = ?", unknown.ID).Update("created_at", stale).Error)
			f.polarSeats = polarSeats
			f.onPatch = func(w http.ResponseWriter, f *seatPurchaseFixture, seats int) {
				f.polarSeats = seats
				f.writeJSON(w, http.StatusOK, f.snapshot(seats))
			}

			_, err = f.purchase(1, int32(polarSeats))

			requireGRPCCode(t, codes.Aborted, err)
			assert.ErrorContains(t, err, "A seat purchase is already in progress.")
			assert.Equal(t, 1, f.patchCount(), "a new purchase does not charge while the first one is unknown")
			assert.Equal(t, polarSeats, f.purchasedSeats(), "the verified snapshot repairs the capacity")
			stored := f.latestPurchase()
			assert.Equal(t, unknown.ID, stored.ID)
			assert.Equal(t, billingmodels.SeatPurchaseStatusUnknown, stored.Status)
			assert.Empty(t, stored.ErrorCode)
		})
	}
}

func Test__PreviewSettlesAnUnconfirmedPurchaseFromPolar(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.dropConnection()
	_, err := f.purchase(1, 3)
	require.NoError(t, err)
	f.polarSeats = 4

	preview, err := f.preview(1)

	require.NoError(t, err)
	assert.EqualValues(t, 4, preview.PurchasedSeats)
	assert.EqualValues(t, 5, preview.TargetSeats)
	assert.Equal(t, billingmodels.SeatPurchaseStatusSucceeded, f.latestPurchase().Status)
	assert.Equal(t, 1, f.patchCount())
}

func Test__PurchaseWaitsWhileAnUnconfirmedPurchaseIsOpen(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.dropConnection()
	_, err := f.purchase(1, 3)
	require.NoError(t, err)

	_, err = f.purchase(1, 3)

	requireGRPCCode(t, codes.Aborted, err)
	assert.ErrorContains(t, err, "A seat purchase is already in progress.")
	assert.Equal(t, 1, f.patchCount())
	assert.Equal(t, billingmodels.SeatPurchaseStatusUnknown, f.latestPurchase().Status)
}

func Test__ConcurrentPurchasesThatExceedTheMaximumSendOneUpdate(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(5))
	f.onPatch = func(w http.ResponseWriter, f *seatPurchaseFixture, seats int) {
		time.Sleep(200 * time.Millisecond)
		f.polarSeats = seats
		f.writeJSON(w, http.StatusOK, f.snapshot(seats))
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			_, err := f.purchase(2, 3)
			results <- err
		}()
	}
	close(start)

	succeeded, aborted := 0, 0
	for range 2 {
		err := <-results
		if err == nil {
			succeeded++
			continue
		}
		require.Equal(t, codes.Aborted, grpcerrors.Code(err), err.Error())
		aborted++
	}

	assert.Equal(t, 1, succeeded)
	assert.Equal(t, 1, aborted)
	assert.Equal(t, 1, f.patchCount())
	assert.Equal(t, 5, f.purchasedSeats())
}

func Test__DescribeBillingShowsTheLatestSeatPurchase(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))

	before, err := DescribeOrganizationBilling(context.Background(), f.r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
	require.NoError(t, err)
	assert.Nil(t, before.LastSeatPurchase)

	f.rejectWith(http.StatusPaymentRequired, "PaymentFailed")
	_, err = f.purchase(1, 3)
	require.Error(t, err)

	after, err := DescribeOrganizationBilling(context.Background(), f.r.Organization.ID.String(), &pb.DescribeOrganizationBillingRequest{})
	require.NoError(t, err)
	require.NotNil(t, after.LastSeatPurchase)
	assert.Equal(t, billingmodels.SeatPurchaseStatusFailed, after.LastSeatPurchase.Status)
	assert.Equal(t, "The payment failed. No seats were added.", after.LastSeatPurchase.Message)
	assert.EqualValues(t, 4, after.LastSeatPurchase.ToSeats)
}

func Test__SyncBillingKeepsAnExpiredUnknownSeatPurchaseWithoutEvidence(t *testing.T) {
	for _, polarAvailable := range []bool{true, false} {
		name := "unchanged subscription proves nothing"
		if !polarAvailable {
			name = "unavailable Polar preserves the unconfirmed purchase"
		}
		t.Run(name, func(t *testing.T) {
			f := newSeatPurchaseFixture(t, 3, new(50))
			purchase := billingmodels.NewSeatPurchase(f.r.Organization.ID, &f.r.User, f.subscriptionID, 3, 4, nil)
			purchase.Status = billingmodels.SeatPurchaseStatusUnknown
			purchase.CreatedAt = time.Now().Add(-2 * polar.SeatPurchaseConfirmationWindow)
			require.NoError(t, purchase.Create(database.DB(t.Context())))
			unchanged := f.snapshot(3)
			unchanged["modified_at"] = purchase.CreatedAt.Add(-time.Minute).UTC().Format(time.RFC3339)
			var purchaseUpdates atomic.Int32
			server := polartest.APIServer(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodPatch {
					purchaseUpdates.Add(1)
					http.Error(w, "sync must not buy seats", http.StatusInternalServerError)
					return
				}
				if !polarAvailable {
					http.Error(w, "Polar unavailable", http.StatusServiceUnavailable)
					return
				}
				switch req.URL.Path {
				case "/subscriptions/":
					f.writeJSON(w, http.StatusOK, map[string]any{"items": []any{unchanged}, "pagination": map[string]int{"max_page": 1}})
				case "/subscriptions/" + f.subscriptionID:
					f.writeJSON(w, http.StatusOK, unchanged)
				case "/customers/external/" + f.r.Organization.ID.String() + "/payment-methods":
					f.writeJSON(w, http.StatusOK, map[string]any{"items": []any{}, "pagination": map[string]int{"max_page": 1}})
				default:
					t.Errorf("unexpected Polar request %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusInternalServerError)
				}
			})
			polartest.UseServer(t, server)

			billing, err := SyncOrganizationBilling(t.Context(), f.r.Organization.ID.String(), &pb.SyncOrganizationBillingRequest{})

			require.NoError(t, err)
			require.NotNil(t, billing.LastSeatPurchase)
			assert.EqualValues(t, 3, billing.GetPurchasedSeats())
			assert.Equal(t, 3, f.purchasedSeats())
			assert.Zero(t, purchaseUpdates.Load(), "refreshing billing cannot submit a purchase")
			assert.Equal(t, billingmodels.SeatPurchaseStatusUnknown, f.latestPurchase().Status)
			assert.Equal(t, seatPurchaseAwaitingConfirmation, billing.LastSeatPurchase.Status)
		})
	}
}

func Test__PurchaseSeatsInsideTheRenewalWindowNeverReachesPolar(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.setPeriodEnd(time.Now().Add(seatPurchaseBoundaryWindow - time.Minute))

	_, err := f.purchase(1, 3)

	requireGRPCCode(t, codes.FailedPrecondition, err)
	assert.ErrorContains(t, err, "The billing period renews soon. Buy seats after the renewal.")
	assert.Zero(t, f.patchCount())
	assert.Zero(t, f.scheduleCount())
	assert.Equal(t, 3, f.purchasedSeats())
	assert.Equal(t, 0, f.getCount, "the local check rejects the request before the Polar refresh")
	latest, findErr := billingmodels.FindLatestSeatPurchase(database.Conn(), f.r.Organization.ID)
	require.NoError(t, findErr)
	assert.Nil(t, latest)
}

func Test__PreviewSeatPurchaseInsideTheRenewalWindowReturnsTheSameError(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.setPeriodEnd(time.Now().Add(seatPurchaseBoundaryWindow - time.Minute))

	_, err := f.preview(1)

	requireGRPCCode(t, codes.FailedPrecondition, err)
	assert.ErrorContains(t, err, "The billing period renews soon. Buy seats after the renewal.")
}

func Test__PurchaseSeatsEdgeOfTheRenewalWindow(t *testing.T) {
	tests := []struct {
		name         string
		untilRenewal time.Duration
		wantErr      bool
	}{
		{name: "just outside the window", untilRenewal: seatPurchaseBoundaryWindow + time.Second},
		{name: "exactly at the window edge", untilRenewal: seatPurchaseBoundaryWindow, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSeatPurchaseFixture(t, 3, new(50))
			now := time.Now().UTC().Truncate(time.Second)
			f.setPeriodEnd(now.Add(tt.untilRenewal))

			_, err := f.purchaseAt(fixedSeatPurchaseClock(now), 1, 3)

			if tt.wantErr {
				requireGRPCCode(t, codes.FailedPrecondition, err)
				assert.Zero(t, f.patchCount())
				assert.Equal(t, 3, f.purchasedSeats())
				return
			}
			require.NoError(t, err)
			require.Equal(t, 1, f.patchCount())
			assert.Equal(t, 4, f.purchasedSeats())
		})
	}
}

func Test__PurchaseSeatsAfterTheRenewalAppliedAScheduledReductionIsRejected(t *testing.T) {
	f := newSeatPurchaseFixture(t, 10, new(50))
	// The server reads 10 seats. Polar renews to 7 before the refresh.
	renewedStart := f.periodEnd
	f.mu.Lock()
	f.periodStart = renewedStart
	f.periodEnd = renewedStart.Add(30 * 24 * time.Hour)
	f.polarSeats = 7
	f.mu.Unlock()

	_, err := f.purchase(1, 10)

	requireGRPCCode(t, codes.Aborted, err)
	assert.ErrorContains(t, err, "The billing period changed. Refresh and try again.")
	assert.Zero(t, f.patchCount(), "no update can buy 11 seats")
	assert.Equal(t, 7, f.purchasedSeats(), "the refresh records the renewed seats")
	latest, findErr := billingmodels.FindLatestSeatPurchase(database.Conn(), f.r.Organization.ID)
	require.NoError(t, findErr)
	assert.Nil(t, latest, "no attempt is recorded")
}

func Test__PurchaseSeatsRejectsAPeriodThatChangedAfterTheFirstCheck(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	// The seat count stays the same, so only the period differs.
	f.mu.Lock()
	f.periodStart = f.periodEnd
	f.periodEnd = f.periodEnd.Add(30 * 24 * time.Hour)
	f.mu.Unlock()

	_, err := f.purchase(1, 3)

	requireGRPCCode(t, codes.Aborted, err)
	assert.ErrorContains(t, err, "The billing period changed. Refresh and try again.")
	assert.Zero(t, f.patchCount())
	assert.Equal(t, 3, f.purchasedSeats())
	latest, findErr := billingmodels.FindLatestSeatPurchase(database.Conn(), f.r.Organization.ID)
	require.NoError(t, findErr)
	assert.Nil(t, latest, "no attempt stays in flight")

	// A new request works from the refreshed period.
	_, err = f.purchase(1, 3)
	require.NoError(t, err)
	assert.Equal(t, 1, f.patchCount())
	assert.Equal(t, 4, f.purchasedSeats())
}

func Test__PurchaseSeatsStopsAtTheFinalCheckWhenTheClockEntersTheRenewalWindow(t *testing.T) {
	f := newSeatPurchaseFixture(t, 10, new(50))
	f.pendingSeats = new(7)

	// Time is normal until the purchase is recorded. Then the clock reaches
	// the renewal window, as when the request waited for a slow dependency.
	var recorded atomic.Bool
	clock := func() time.Time {
		if !recorded.Load() {
			var count int64
			require.NoError(t, database.Conn().Model(&billingmodels.SeatPurchase{}).Where("organization_id = ?", f.r.Organization.ID).Count(&count).Error)
			recorded.Store(count > 0)
		}
		if recorded.Load() {
			return f.periodEnd.Add(-time.Minute)
		}
		return time.Now()
	}

	_, err := f.purchaseAt(clock, 1, 10)

	requireGRPCCode(t, codes.FailedPrecondition, err)
	assert.ErrorContains(t, err, "The billing period renews soon. Buy seats after the renewal.")
	assert.Zero(t, f.patchCount(), "no update can buy 11 seats")
	assert.Zero(t, f.scheduleCount(), "the scheduled reduction stays as it is")
	assert.Equal(t, 10, f.purchasedSeats())

	failed := f.latestPurchase()
	assert.Equal(t, billingmodels.SeatPurchaseStatusFailed, failed.Status)
	assert.Equal(t, billingmodels.SeatPurchaseErrorRenewalBoundary, failed.ErrorCode)
	assert.Contains(t, seatPurchaseMessage(failed), "No seats were added.")
	inFlight, err := billingmodels.FindInFlightSeatPurchase(database.Conn(), f.r.Organization.ID)
	require.NoError(t, err)
	assert.Nil(t, inFlight, "the stopped attempt does not block a later purchase")

	// After the renewal, a new purchase starts from the renewed period.
	f.mu.Lock()
	f.periodStart = f.periodEnd
	f.periodEnd = f.periodEnd.Add(30 * 24 * time.Hour)
	f.polarSeats = 7
	f.pendingSeats = nil
	f.mu.Unlock()
	_, err = f.purchase(1, 10)
	requireGRPCCode(t, codes.Aborted, err)
	_, err = f.purchase(1, 7)
	require.NoError(t, err)
	require.Equal(t, 1, f.patchCount())
	assert.Equal(t, float64(8), f.patchBodies[0]["seats"])
}
