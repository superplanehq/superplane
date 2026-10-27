package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/e2e/session"
)

// Two pages of one browser context are two tabs: each page has its own
// JavaScript state, and both pages share same-origin storage. The test
// holds each purchase request in the browser until the test releases it,
// so the order of confirmations and answers is fixed without sleeps.
func TestSeatPurchaseTabs(t *testing.T) {
	t.Run("different confirmations keep their identities across a lost answer, a renewal, and a reload", func(t *testing.T) {
		s := startSeatPurchaseTabs(t)
		first := s.firstTab.confirm(1)
		second := s.secondTab.confirm(2)
		require.NotEqual(t, first.body.OperationKey, second.body.OperationKey)
		s.requireConfirmedFor(first.body, s.polar.period())
		s.requireConfirmedFor(second.body, s.polar.period())

		first.loseResponse()
		second.loseRequest()
		require.Equal(t, []int{5}, s.polar.charges())

		s.renewWithSeats(4)
		s.firstTab.reload()
		s.secondTab.reload()

		// The tab that did not confirm the first purchase retries its counts.
		retry := s.secondTab.confirm(1)
		require.Equal(t, first.body, retry.body)
		require.Equal(t, http.StatusOK, retry.deliver())
		s.secondTab.assertStatus("Seats added.")
		require.Equal(t, []int{5}, s.polar.charges(), "the retry replays the purchase that Polar charged")

		// The request that never reached the server keeps its own key and
		// confirmed period. The server rejects that stale period.
		stale := s.firstTab.confirm(2)
		require.Equal(t, second.body, stale.body)
		require.Equal(t, http.StatusConflict, stale.deliver())
		require.Equal(t, []int{5}, s.polar.charges())

		next := s.firstTab.confirm(2)
		require.NotContains(t, []string{first.body.OperationKey, second.body.OperationKey}, next.body.OperationKey)
		s.requireConfirmedFor(next.body, s.polar.period())
		require.Equal(t, http.StatusOK, next.deliver())
		require.Equal(t, []int{5, 6}, s.polar.charges())
	})

	t.Run("a settled purchase in one tab keeps the unresolved purchase of another tab", func(t *testing.T) {
		s := startSeatPurchaseTabs(t)
		first := s.firstTab.confirm(1)
		second := s.secondTab.confirm(2)

		// The first tab settles and forgets its key while the second
		// confirmation is in flight.
		require.Equal(t, http.StatusOK, first.deliver())
		s.firstTab.assertStatus("Seats added.")
		second.loseRequest()
		require.Equal(t, []int{5}, s.polar.charges())

		s.renewWithSeats(4)
		s.firstTab.reload()
		s.secondTab.reload()

		retry := s.firstTab.confirm(2)
		require.Equal(t, second.body, retry.body)
		require.Equal(t, http.StatusConflict, retry.deliver())
		require.Equal(t, []int{5}, s.polar.charges(), "the lost request never reached Polar")

		// The settled key is gone, so the same counts start a new purchase.
		next := s.secondTab.confirm(1)
		require.NotEqual(t, first.body.OperationKey, next.body.OperationKey)
		s.requireConfirmedFor(next.body, s.polar.period())
		require.Equal(t, http.StatusOK, next.deliver())
		require.Equal(t, []int{5, 5}, s.polar.charges())
	})

	t.Run("concurrent confirmations of the same purchase share one identity and charge once", func(t *testing.T) {
		s := startSeatPurchaseTabs(t)
		first := s.firstTab.confirm(1)
		second := s.secondTab.confirm(1)
		require.Equal(t, first.body, second.body)

		require.Equal(t, http.StatusOK, first.deliver())
		require.Equal(t, http.StatusOK, second.deliver())
		s.firstTab.assertStatus("Seats added.")
		s.secondTab.assertStatus("Seats added.")
		require.Equal(t, []int{5}, s.polar.charges())
	})
}

type seatPurchaseTabsSteps struct {
	t         *testing.T
	session   *session.TestSession
	polar     *seatPurchasePolar
	firstTab  *seatPurchaseTab
	secondTab *seatPurchaseTab
}

func startSeatPurchaseTabs(t *testing.T) *seatPurchaseTabsSteps {
	s := &seatPurchaseTabsSteps{t: t}
	s.session = ctx.NewSession(t)
	s.session.Start()
	s.polar = newSeatPurchasePolar(t, s.session.OrgID.String())
	s.givenATeamSubscriptionWithFourSeats()
	s.session.Login()

	path := s.billingPath()
	s.firstTab = openSeatPurchaseTab(t, s.session.Page(), s.session.BaseURL+path)
	second, err := s.session.Page().Context().NewPage()
	require.NoError(t, err)
	t.Cleanup(func() { _ = second.Close() })
	s.secondTab = openSeatPurchaseTab(t, second, s.session.BaseURL+path)
	return s
}

func (s *seatPurchaseTabsSteps) givenATeamSubscriptionWithFourSeats() {
	db := database.DB(s.t.Context())
	require.NoError(s.t, models.EnableExperimentalFeature(s.session.OrgID, features.FeatureFactories))
	maximum := 50
	plan := billingmodels.BillingPlan{
		Key: "seat_tabs_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private",
		PerSeat: true, PriceCents: 2000, SeatLimit: &maximum, PolarProductID: &s.polar.product,
		CreditFrequency: "none", CreditExpiration: "never",
	}
	require.NoError(s.t, billingmodels.SaveBillingPlan(db, &plan))
	period := s.polar.period()
	require.NoError(s.t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", s.session.OrgID).Updates(map[string]any{
		"plan_id": plan.ID, "plan": plan.Key, "plan_source": billingmodels.BillingPlanSourcePolar,
		"polar_subscription_id": s.polar.subscriptionID, "polar_subscription_status": billingmodels.PolarSubscriptionStatusActive,
		"purchased_seats": 4, "current_period_start": period.start, "current_period_end": period.end,
		"cancel_at_period_end": false,
	}).Error)
}

func (s *seatPurchaseTabsSteps) billingPath() string {
	factory, err := models.CreateFactory(database.DB(s.t.Context()), s.session.OrgID, "Seat tabs", "", "")
	require.NoError(s.t, err)
	return "/" + s.session.OrgID.String() + "/workspaces/" + factory.Key + "/settings/organization/billing"
}

// renewWithSeats starts the next billing period in Polar, as when Polar
// renews with a scheduled seat count, and delivers the webhook.
func (s *seatPurchaseTabsSteps) renewWithSeats(seats int) {
	snapshot := s.polar.renew(seats)
	body, err := json.Marshal(map[string]any{"type": "subscription.updated", "data": snapshot})
	require.NoError(s.t, err)
	var event polar.SubscriptionWebhookEvent
	require.NoError(s.t, json.Unmarshal(body, &event))
	require.NoError(s.t, polar.ApplySubscriptionEvent(context.Background(), database.Conn(), &event))
}

func (s *seatPurchaseTabsSteps) requireConfirmedFor(body seatPurchaseBody, period seatPurchasePeriod) {
	s.t.Helper()
	require.Equal(s.t, s.polar.subscriptionID, body.ConfirmedSubscriptionID)
	start, err := time.Parse(time.RFC3339Nano, body.ConfirmedPeriodStart)
	require.NoError(s.t, err)
	end, err := time.Parse(time.RFC3339Nano, body.ConfirmedPeriodEnd)
	require.NoError(s.t, err)
	require.True(s.t, start.Equal(period.start), "confirmed period start %s, want %s", start, period.start)
	require.True(s.t, end.Equal(period.end), "confirmed period end %s, want %s", end, period.end)
}

// seatPurchaseTab is one browser tab on the billing page. Its purchase
// requests wait in held until the test releases them.
type seatPurchaseTab struct {
	t    *testing.T
	page pw.Page
	url  string
	held chan pw.Route
}

func openSeatPurchaseTab(t *testing.T, page pw.Page, url string) *seatPurchaseTab {
	tab := &seatPurchaseTab{t: t, page: page, url: url, held: make(chan pw.Route, 1)}
	require.NoError(t, page.Route("**/api/v1/organizations/*/billing/seats/purchase", func(route pw.Route) {
		tab.held <- route
	}))
	_, err := page.Goto(url, pw.PageGotoOptions{WaitUntil: pw.WaitUntilStateDomcontentloaded})
	require.NoError(t, err)
	tab.waitForBuySeats()
	return tab
}

func (tab *seatPurchaseTab) reload() {
	_, err := tab.page.Reload(pw.PageReloadOptions{WaitUntil: pw.WaitUntilStateDomcontentloaded})
	require.NoError(tab.t, err)
	tab.waitForBuySeats()
}

func (tab *seatPurchaseTab) waitForBuySeats() {
	require.NoError(tab.t, tab.buySeatsButton().WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible}))
}

func (tab *seatPurchaseTab) buySeatsButton() pw.Locator {
	return tab.page.GetByRole("button", pw.PageGetByRoleOptions{Name: "Buy seats", Exact: pw.Bool(true)})
}

func (tab *seatPurchaseTab) dialog() pw.Locator {
	return tab.page.GetByRole("dialog", pw.PageGetByRoleOptions{Name: "Buy seats", Exact: pw.Bool(true)})
}

// confirm opens a new purchase dialog for the seats, confirms the new
// preview, and returns the purchase request that the tab sends.
func (tab *seatPurchaseTab) confirm(seats int) *heldSeatPurchase {
	tab.t.Helper()
	dialog := tab.dialog()
	if visible, err := dialog.IsVisible(); err == nil && visible {
		require.NoError(tab.t, dialog.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Close", Exact: pw.Bool(true)}).Click())
		require.NoError(tab.t, dialog.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateHidden}))
	}
	require.NoError(tab.t, tab.buySeatsButton().Click())
	require.NoError(tab.t, dialog.GetByLabel("Additional seats", pw.LocatorGetByLabelOptions{Exact: pw.Bool(true)}).Fill(strconv.Itoa(seats)))
	// Confirm is enabled only for a preview of the current input.
	require.NoError(tab.t, dialog.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Confirm purchase", Exact: pw.Bool(true)}).Click())

	select {
	case route := <-tab.held:
		held := &heldSeatPurchase{tab: tab, route: route}
		require.NoError(tab.t, route.Request().PostDataJSON(&held.body))
		require.Equal(tab.t, seats, held.body.AdditionalSeats)
		return held
	case <-time.After(15 * time.Second):
		tab.t.Fatal("the tab did not send the seat purchase request")
		return nil
	}
}

func (tab *seatPurchaseTab) assertStatus(text string) {
	tab.t.Helper()
	status := tab.dialog().GetByRole("status").Filter(pw.LocatorFilterOptions{HasText: text})
	require.NoError(tab.t, status.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible}))
}

// waitUntilSettled waits until the tab handled the answer, including the
// storage update and the billing recovery after it.
func (tab *seatPurchaseTab) waitUntilSettled() {
	tab.t.Helper()
	submitting := tab.page.GetByRole("button", pw.PageGetByRoleOptions{Name: "Buying seats…", Exact: pw.Bool(true)})
	require.NoError(tab.t, submitting.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateHidden}))
}

type seatPurchaseBody struct {
	AdditionalSeats         int    `json:"additionalSeats"`
	ExpectedPurchasedSeats  int    `json:"expectedPurchasedSeats"`
	OperationKey            string `json:"operationKey"`
	ConfirmedSubscriptionID string `json:"confirmedSubscriptionId"`
	ConfirmedPeriodStart    string `json:"confirmedPeriodStart"`
	ConfirmedPeriodEnd      string `json:"confirmedPeriodEnd"`
}

// heldSeatPurchase is a purchase request that the browser sent and the
// test has not released yet.
type heldSeatPurchase struct {
	tab   *seatPurchaseTab
	route pw.Route
	body  seatPurchaseBody
}

// deliver sends the request to the server and gives the answer to the tab.
func (h *heldSeatPurchase) deliver() int {
	h.tab.t.Helper()
	response, err := h.route.Fetch()
	require.NoError(h.tab.t, err)
	require.NoError(h.tab.t, h.route.Fulfill(pw.RouteFulfillOptions{Response: response}))
	h.tab.waitUntilSettled()
	return response.Status()
}

// loseResponse sends the request to the server, which processes it, and
// fails the request in the tab as if the connection dropped.
func (h *heldSeatPurchase) loseResponse() {
	h.tab.t.Helper()
	response, err := h.route.Fetch()
	require.NoError(h.tab.t, err)
	require.Equal(h.tab.t, http.StatusOK, response.Status())
	require.NoError(h.tab.t, h.route.Abort("failed"))
	h.tab.waitUntilSettled()
}

// loseRequest fails the request before it reaches the server.
func (h *heldSeatPurchase) loseRequest() {
	h.tab.t.Helper()
	require.NoError(h.tab.t, h.route.Abort("failed"))
	h.tab.waitUntilSettled()
}

type seatPurchasePeriod struct {
	start time.Time
	end   time.Time
}

// seatPurchasePolar is a fake Polar API for one Team subscription. Each
// seat update with an immediate invoice is a charge.
type seatPurchasePolar struct {
	t              *testing.T
	organizationID string
	subscriptionID string
	product        string

	mu      sync.Mutex
	seats   int
	current seatPurchasePeriod
	charged []int
}

func newSeatPurchasePolar(t *testing.T, organizationID string) *seatPurchasePolar {
	now := time.Now().UTC().Truncate(time.Second)
	f := &seatPurchasePolar{
		t:              t,
		organizationID: organizationID,
		subscriptionID: "sub_" + uuid.NewString(),
		product:        "product_" + uuid.NewString(),
		seats:          4,
		current:        seatPurchasePeriod{start: now.Add(-7 * 24 * time.Hour), end: now.Add(23 * 24 * time.Hour)},
	}
	server := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(server.Close)
	t.Setenv("POLAR_API_BASE_URL", server.URL)
	t.Setenv("POLAR_ACCESS_TOKEN", "seat-purchase-tabs-test")
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	return f
}

func (f *seatPurchasePolar) period() seatPurchasePeriod {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.current
}

// charges returns the seat counts of the charged updates in order.
func (f *seatPurchasePolar) charges() []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int{}, f.charged...)
}

func (f *seatPurchasePolar) renew(seats int) map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	start := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	f.current = seatPurchasePeriod{start: start, end: start.Add(30 * 24 * time.Hour)}
	f.seats = seats
	return f.snapshot()
}

func (f *seatPurchasePolar) serve(w http.ResponseWriter, req *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// Polar echoes the pinned API version on every response it routes.
	w.Header().Set("Polar-Version", req.Header.Get("Polar-Version"))
	switch {
	case req.URL.Path == "/subscriptions/"+f.subscriptionID && req.Method == http.MethodGet:
		writeSeatPurchasePolarJSON(w, http.StatusOK, f.snapshot())
	case req.URL.Path == "/subscriptions/"+f.subscriptionID && req.Method == http.MethodPatch:
		f.updateSeats(w, req)
	case req.URL.Path == "/subscriptions/" && req.Method == http.MethodGet:
		writeSeatPurchasePolarJSON(w, http.StatusOK, map[string]any{"items": []any{f.snapshot()}, "pagination": map[string]int{"max_page": 1}})
	case req.Method == http.MethodGet:
		// The billing page also lists orders and payment methods.
		writeSeatPurchasePolarJSON(w, http.StatusOK, map[string]any{"items": []any{}, "pagination": map[string]int{"max_page": 1}})
	default:
		f.t.Logf("fake Polar has no answer for %s %s", req.Method, req.URL.Path)
		writeSeatPurchasePolarJSON(w, http.StatusNotFound, map[string]any{"error": "ResourceNotFound", "detail": "Not found"})
	}
}

func (f *seatPurchasePolar) updateSeats(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Seats             int    `json:"seats"`
		ProrationBehavior string `json:"proration_behavior"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body.Seats == 0 {
		writeSeatPurchasePolarJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "RequestValidationError", "detail": "seats"})
		return
	}
	// A next-period update schedules the renewal seat count and charges nothing.
	if body.ProrationBehavior == "invoice" {
		f.charged = append(f.charged, body.Seats)
		f.seats = body.Seats
	}
	writeSeatPurchasePolarJSON(w, http.StatusOK, f.snapshot())
}

func (f *seatPurchasePolar) snapshot() map[string]any {
	return map[string]any{
		"id": f.subscriptionID, "product_id": f.product, "status": "active", "seats": f.seats,
		"cancel_at_period_end": false, "modified_at": time.Now().UTC().Format(time.RFC3339Nano),
		"current_period_start": f.current.start.Format(time.RFC3339), "current_period_end": f.current.end.Format(time.RFC3339),
		"customer_id": "cust_seat_tabs", "external_customer_id": f.organizationID, "pending_update": nil,
	}
}

func writeSeatPurchasePolarJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
