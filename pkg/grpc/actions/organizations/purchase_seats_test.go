package organizations

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/billing/seatrenewal"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

const seatPriceCents = 2000

// seatPurchaseFixture is a Team organization with an active Polar subscription
// and a fake Polar server that records every call.
type seatPurchaseFixture struct {
	t              *testing.T
	r              *support.ResourceRegistry
	catalog        models.BillingPlan
	subscriptionID string
	product        string
	periodStart    time.Time
	periodEnd      time.Time

	mu sync.Mutex
	// patchBodies are the seat purchase updates. Renewal schedules are in schedules.
	patchBodies []map[string]any
	getCount    int
	// onPatch writes the seat purchase response. The default applies the update.
	onPatch func(w http.ResponseWriter, f *seatPurchaseFixture, seats int)
	// polarSeats is the seat count that GET reports.
	polarSeats int
	// schedules are the next_period updates that schedule the renewal seat count.
	schedules []map[string]any
	// pendingSeats is the seat count that Polar renews with, when it differs
	// from polarSeats.
	pendingSeats *int
	// modifiedAt is the subscription modification time that snapshots report.
	// The zero value makes each snapshot report the current time.
	modifiedAt time.Time
	// replays counts webhook replays so each snapshot is newer than the last.
	replays int
	// pastDueAt is sent as past_due_at in a past_due replay when set.
	pastDueAt *time.Time
}

func newSeatPurchaseFixture(t *testing.T, purchased int, seatLimit *int) *seatPurchaseFixture {
	t.Helper()
	r := support.Setup(t)
	t.Cleanup(r.Close)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeatureInTransaction(db, r.Organization.ID, features.FeatureNewBilling))

	product := "prod_team_" + uuid.NewString()
	catalog := models.BillingPlan{
		Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private",
		PerSeat: true, PriceCents: seatPriceCents, SeatLimit: seatLimit, PolarProductID: &product,
		CreditFrequency: "none", CreditExpiration: "never",
	}
	require.NoError(t, models.SaveBillingPlan(db, &catalog))

	now := time.Now().UTC().Truncate(time.Second)
	f := &seatPurchaseFixture{
		t: t, r: r, catalog: catalog, product: product,
		subscriptionID: "sub_" + uuid.NewString(),
		periodStart:    now.Add(-7 * 24 * time.Hour),
		periodEnd:      now.Add(23 * 24 * time.Hour),
		polarSeats:     purchased,
	}
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Updates(map[string]any{
		"plan_id": catalog.ID, "plan": catalog.Key, "plan_source": models.BillingPlanSourcePolar,
		"polar_subscription_id": f.subscriptionID, "polar_subscription_status": models.PolarSubscriptionStatusActive,
		"purchased_seats": purchased, "current_period_start": f.periodStart, "current_period_end": f.periodEnd,
		"cancel_at_period_end": false,
	}).Error)
	f.onPatch = func(w http.ResponseWriter, f *seatPurchaseFixture, seats int) {
		// An applied seat update replaces the scheduled one.
		f.polarSeats = seats
		f.pendingSeats = nil
		f.writeJSON(w, http.StatusOK, f.snapshot(seats))
	}

	server := polarAPIServer(t, f.serve)
	usePolarTestServer(t, server)
	// Background renewal schedules must end before the server and the
	// database close.
	t.Cleanup(seatrenewal.Wait)
	return f
}

func (f *seatPurchaseFixture) serve(w http.ResponseWriter, req *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if req.URL.Path == "/customers/external/"+f.r.Organization.ID.String()+"/payment-methods" {
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
		seats := int(body["seats"].(float64))
		if body["proration_behavior"] == "next_period" {
			// Like Polar, an update back to the current seats clears the scheduled one.
			f.schedules = append(f.schedules, body)
			f.pendingSeats = &seats
			if seats == f.polarSeats {
				f.pendingSeats = nil
			}
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
		"cancel_at_period_end": false, "modified_at": modifiedAt.UTC().Format(time.RFC3339),
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

func (f *seatPurchaseFixture) purchase(additional, expected int32) (*pb.PurchaseSeatsResponse, error) {
	return PurchaseSeats(context.Background(), f.r.Organization.ID.String(),
		&pb.PurchaseSeatsRequest{AdditionalSeats: additional, ExpectedPurchasedSeats: expected}, f.r.User.String())
}

func (f *seatPurchaseFixture) preview(additional int32) (*pb.PreviewSeatPurchaseResponse, error) {
	return PreviewSeatPurchase(context.Background(), f.r.Organization.ID.String(), &pb.PreviewSeatPurchaseRequest{AdditionalSeats: additional})
}

func (f *seatPurchaseFixture) plan() *models.OrganizationBillingPlan {
	f.t.Helper()
	plan, err := models.FindOrganizationBillingPlan(database.Conn(), f.r.Organization.ID)
	require.NoError(f.t, err)
	return plan
}

func (f *seatPurchaseFixture) purchasedSeats() int {
	f.t.Helper()
	plan := f.plan()
	require.NotNil(f.t, plan.PurchasedSeats)
	return *plan.PurchasedSeats
}

func (f *seatPurchaseFixture) latestPurchase() *models.SeatPurchase {
	f.t.Helper()
	purchase, err := models.FindLatestSeatPurchase(database.Conn(), f.r.Organization.ID)
	require.NoError(f.t, err)
	require.NotNil(f.t, purchase)
	return purchase
}

func (f *seatPurchaseFixture) updateBillingPlan(values map[string]any) {
	f.t.Helper()
	require.NoError(f.t, database.Conn().Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", f.r.Organization.ID).Updates(values).Error)
}

func (f *seatPurchaseFixture) applyWebhook(seats int, modifiedAt time.Time) {
	f.t.Helper()
	data := f.snapshot(seats)
	data["modified_at"] = modifiedAt.UTC().Format(time.RFC3339)
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
	assert.True(t, preview.PeriodEnd.AsTime().Equal(f.periodEnd))
	assert.WithinDuration(t, time.Now(), preview.PeriodStart.AsTime(), time.Minute)
	require.NotNil(t, preview.PurchasableSeats)
	assert.EqualValues(t, 47, *preview.PurchasableSeats)
	assert.True(t, preview.Estimate)
	assert.True(t, preview.ExcludesTax)

	assert.Zero(t, f.patchCount())
	assert.Equal(t, 3, f.purchasedSeats())
	latest, err := models.FindLatestSeatPurchase(database.Conn(), f.r.Organization.ID)
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

	assert.Equal(t, models.SeatPurchaseStatusSucceeded, response.Purchase.Status)
	assert.EqualValues(t, 3, response.Purchase.FromSeats)
	assert.EqualValues(t, 4, response.Purchase.ToSeats)
	assert.Empty(t, response.Purchase.Message)
	require.NotNil(t, response.Billing)
	assert.EqualValues(t, 4, response.Billing.GetPurchasedSeats())
	assert.Equal(t, response.Purchase.Id, response.Billing.LastSeatPurchase.Id)

	stored := f.latestPurchase()
	require.NotNil(t, stored.UserID)
	assert.Equal(t, f.r.User, *stored.UserID)

	expected, err := models.FindExpectedSeatCount(database.Conn(), f.r.Organization.ID, f.subscriptionID)
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
				require.NoError(t, models.SetOrganizationSeatLimitOverride(database.Conn(), f.r.Organization.ID, tt.override))
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
	require.NoError(t, models.SetOrganizationSeatLimitOverride(database.Conn(), f.r.Organization.ID, nil))

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
			flat := models.BillingPlan{Key: "flat_" + uuid.NewString()[:8], Name: "Flat", Category: "paid", Visibility: "private", PriceCents: seatPriceCents, PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
			require.NoError(t, models.SaveBillingPlan(database.Conn(), &flat))
			f.updateBillingPlan(map[string]any{"plan_id": flat.ID, "plan": flat.Key})
		}, message: "Seat purchases need an active Team subscription."},
		{name: "admin plan", change: func(f *seatPurchaseFixture) {
			f.updateBillingPlan(map[string]any{"plan_source": models.BillingPlanSourceAdmin})
		}, message: "Seat purchases need an active Team subscription."},
		{name: "subscription is not active", change: func(f *seatPurchaseFixture) {
			f.updateBillingPlan(map[string]any{"polar_subscription_status": models.PolarSubscriptionStatusPastDue})
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
			tt.change(f)

			_, err := f.purchase(1, 3)
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

func Test__PurchaseSeatsRequiresNewBilling(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	require.NoError(t, models.DisableExperimentalFeatureInTransaction(database.Conn(), f.r.Organization.ID, features.FeatureNewBilling))

	_, err := f.purchase(1, 3)
	requireGRPCCode(t, codes.FailedPrecondition, err)
	_, err = f.preview(1)
	requireGRPCCode(t, codes.FailedPrecondition, err)

	assert.Zero(t, f.patchCount())
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
		{"payment failed", http.StatusPaymentRequired, "PaymentFailed", codes.FailedPrecondition, models.SeatPurchaseStatusFailed,
			"The payment failed. No seats were added."},
		{"payment needs bank confirmation", http.StatusPaymentRequired, "PaymentActionRequired", codes.FailedPrecondition, models.SeatPurchaseStatusActionRequired,
			"Your bank needs to confirm this payment. Update the payment method and try again."},
		{"subscription locked", http.StatusConflict, "SubscriptionLocked", codes.Aborted, models.SeatPurchaseStatusFailed,
			"Try again shortly."},
		{"subscription canceled", http.StatusForbidden, "AlreadyCanceledSubscription", codes.FailedPrecondition, models.SeatPurchaseStatusFailed,
			"The subscription cannot add seats. Check that it is active and not set to cancel."},
		{"subscription inactive", http.StatusForbidden, "InactiveSubscription", codes.FailedPrecondition, models.SeatPurchaseStatusFailed,
			"The subscription cannot add seats. Check that it is active and not set to cancel."},
		{"seat count rejected", http.StatusUnprocessableEntity, "HTTPValidationError", codes.InvalidArgument, models.SeatPurchaseStatusFailed,
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
	assert.Equal(t, models.SeatPurchaseStatusFailed, f.latestPurchase().Status)
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
			assert.Equal(t, models.SeatPurchaseStatusUnknown, f.latestPurchase().Status)
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
	assert.Equal(t, models.SeatPurchaseStatusSucceeded, f.latestPurchase().Status)
	assert.Equal(t, 1, f.patchCount())
}

func Test__WebhookWithUnchangedSeatsFailsAnUnconfirmedPurchase(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.dropConnection()
	_, err := f.purchase(1, 3)
	require.NoError(t, err)

	f.applyWebhook(3, time.Now().Add(time.Minute))

	assert.Equal(t, 3, f.purchasedSeats())
	stored := f.latestPurchase()
	assert.Equal(t, models.SeatPurchaseStatusFailed, stored.Status)
	assert.Equal(t, models.SeatPurchaseErrorNotApplied, stored.ErrorCode)

	f.onPatch = func(w http.ResponseWriter, f *seatPurchaseFixture, seats int) {
		f.writeJSON(w, http.StatusOK, f.snapshot(seats))
	}
	_, err = f.purchase(1, 3)
	require.NoError(t, err, "a failed purchase no longer blocks a new one")
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
	assert.Equal(t, new(1), f.pendingRenewalSeats(), "reconciliation restores the reduction without another purchase")
	assert.Equal(t, 4, f.purchasedSeats())
	assert.Equal(t, models.SeatPurchaseStatusSucceeded, f.latestPurchase().Status)
}

func Test__RetryAfterRenewalSettlesAnUnconfirmedPurchase(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.dropConnection()
	_, err := f.purchase(1, 3)
	require.NoError(t, err)

	// Polar applied the purchase and renewed the subscription, but the
	// renewal webhook has not arrived: the saved period already ended.
	f.updateBillingPlan(map[string]any{
		"current_period_start": time.Now().Add(-31 * 24 * time.Hour),
		"current_period_end":   time.Now().Add(-time.Hour),
	})
	f.periodStart = time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	f.periodEnd = time.Now().Add(29 * 24 * time.Hour).UTC().Truncate(time.Second)
	f.polarSeats = 4

	_, err = f.purchase(1, 3)

	requireGRPCCode(t, codes.Aborted, err)
	assert.ErrorContains(t, err, "Seat count changed. Refresh and try again.")
	assert.Equal(t, 1, f.patchCount(), "the retry must not send a second update")
	assert.Equal(t, 4, f.purchasedSeats())
	assert.Equal(t, models.SeatPurchaseStatusSucceeded, f.latestPurchase().Status)
	require.NotNil(t, f.plan().CurrentPeriodEnd)
	assert.True(t, f.plan().CurrentPeriodEnd.Equal(f.periodEnd), "the retry loads the new period from Polar")
}

func Test__RetryAfterRenewalBuysWhenPolarNeverAppliedThePurchase(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.dropConnection()
	_, err := f.purchase(1, 3)
	require.NoError(t, err)
	first := f.latestPurchase()

	// Polar renewed without applying the purchase. The saved period ended,
	// and the renewal proves that Polar did not apply the lost update.
	stale := time.Now().Add(-time.Minute)
	require.NoError(t, database.Conn().Model(&models.SeatPurchase{}).Where("id = ?", first.ID).Update("created_at", stale).Error)
	f.updateBillingPlan(map[string]any{
		"current_period_start": time.Now().Add(-31 * 24 * time.Hour),
		"current_period_end":   time.Now().Add(-time.Hour),
	})
	f.periodStart = time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	f.periodEnd = time.Now().Add(29 * 24 * time.Hour).UTC().Truncate(time.Second)
	f.onPatch = func(w http.ResponseWriter, f *seatPurchaseFixture, seats int) {
		f.polarSeats = seats
		f.writeJSON(w, http.StatusOK, f.snapshot(seats))
	}

	_, err = f.purchase(1, 3)

	require.NoError(t, err, "the settled failure and the refreshed period let the retry buy")
	assert.Equal(t, 2, f.patchCount())
	assert.Equal(t, 4, f.purchasedSeats())
	require.NoError(t, database.Conn().First(first, "id = ?", first.ID).Error)
	assert.Equal(t, models.SeatPurchaseStatusFailed, first.Status)
	assert.Equal(t, models.SeatPurchaseErrorNotApplied, first.ErrorCode)
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

func Test__PurchaseRecoversAnUnconfirmedPurchaseBelowTheTarget(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.dropConnection()
	_, err := f.purchase(2, 3)
	require.NoError(t, err)
	first := f.latestPurchase()
	stale := time.Now().Add(-2 * polar.SeatPurchaseConfirmationWindow)
	require.NoError(t, database.Conn().Model(&models.SeatPurchase{}).Where("id = ?", first.ID).Update("created_at", stale).Error)
	f.polarSeats = 4 // Polar never applied 5; an outside change set 4 instead
	f.onPatch = func(w http.ResponseWriter, f *seatPurchaseFixture, seats int) {
		f.polarSeats = seats
		f.writeJSON(w, http.StatusOK, f.snapshot(seats))
	}

	_, err = f.purchase(1, 4)

	require.NoError(t, err, "the expired purchase below its target must not block purchases")
	assert.Equal(t, 2, f.patchCount())
	assert.Equal(t, 5, f.purchasedSeats())
	require.NoError(t, database.Conn().First(first, "id = ?", first.ID).Error)
	assert.Equal(t, models.SeatPurchaseStatusFailed, first.Status)
	assert.Equal(t, models.SeatPurchaseErrorNotApplied, first.ErrorCode)
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
	assert.Equal(t, models.SeatPurchaseStatusSucceeded, f.latestPurchase().Status)
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
	assert.Equal(t, models.SeatPurchaseStatusUnknown, f.latestPurchase().Status)
}

func Test__PurchaseSettlesAnUnconfirmedPurchaseThatPolarNeverApplied(t *testing.T) {
	f := newSeatPurchaseFixture(t, 3, new(50))
	f.dropConnection()
	_, err := f.purchase(1, 3)
	require.NoError(t, err)
	stale := time.Now().Add(-2 * polar.SeatPurchaseConfirmationWindow)
	require.NoError(t, database.Conn().Model(&models.SeatPurchase{}).Where("organization_id = ?", f.r.Organization.ID).Update("created_at", stale).Error)
	f.onPatch = func(w http.ResponseWriter, f *seatPurchaseFixture, seats int) {
		f.polarSeats = seats
		f.writeJSON(w, http.StatusOK, f.snapshot(seats))
	}

	_, err = f.purchase(1, 3)

	require.NoError(t, err)
	assert.Equal(t, 2, f.patchCount())
	assert.Equal(t, 4, f.purchasedSeats())
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
	assert.Equal(t, models.SeatPurchaseStatusFailed, after.LastSeatPurchase.Status)
	assert.Equal(t, "The payment failed. No seats were added.", after.LastSeatPurchase.Message)
	assert.EqualValues(t, 4, after.LastSeatPurchase.ToSeats)
}

func Test__SyncBillingReconcilesExpiredUnknownSeatPurchases(t *testing.T) {
	for _, polarAvailable := range []bool{true, false} {
		name := "unchanged subscription fails the expired purchase"
		if !polarAvailable {
			name = "unavailable Polar preserves the unconfirmed purchase"
		}
		t.Run(name, func(t *testing.T) {
			f := newSeatPurchaseFixture(t, 3, new(50))
			purchase := models.NewSeatPurchase(f.r.Organization.ID, &f.r.User, f.subscriptionID, 3, 4)
			purchase.Status = models.SeatPurchaseStatusUnknown
			purchase.CreatedAt = time.Now().Add(-2 * polar.SeatPurchaseConfirmationWindow)
			require.NoError(t, purchase.Create(database.DB(t.Context())))
			unchanged := f.snapshot(3)
			unchanged["modified_at"] = purchase.CreatedAt.Add(-time.Minute).UTC().Format(time.RFC3339)
			var purchaseUpdates atomic.Int32
			server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
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
			usePolarTestServer(t, server)

			billing, err := SyncOrganizationBilling(t.Context(), f.r.Organization.ID.String(), &pb.SyncOrganizationBillingRequest{})

			require.NoError(t, err)
			require.NotNil(t, billing.LastSeatPurchase)
			assert.EqualValues(t, 3, billing.GetPurchasedSeats())
			assert.Equal(t, 3, f.purchasedSeats())
			assert.Zero(t, purchaseUpdates.Load(), "refreshing billing cannot submit a purchase")
			stored := f.latestPurchase()
			if !polarAvailable {
				assert.Equal(t, models.SeatPurchaseStatusUnknown, stored.Status)
				assert.Equal(t, seatPurchaseAwaitingConfirmation, billing.LastSeatPurchase.Status)
				return
			}
			assert.Equal(t, models.SeatPurchaseStatusFailed, stored.Status)
			assert.Equal(t, models.SeatPurchaseErrorNotApplied, stored.ErrorCode)
			assert.Equal(t, models.SeatPurchaseStatusFailed, billing.LastSeatPurchase.Status)
			assert.Equal(t, "Polar did not apply the seat purchase. No seats were added.", billing.LastSeatPurchase.Message)
		})
	}
}
