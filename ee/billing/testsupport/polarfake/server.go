// Package polarfake is a fake Polar subscription API for seat tests. Its
// pending seat count stands for a next-period reduction that the retired
// automatic scheduler left in Polar.
package polarfake

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"gorm.io/gorm"
)

// Subscription is the state of one fake Polar subscription.
type Subscription struct {
	ID                 string
	ProductID          string
	ExternalCustomerID string
	Status             string
	Seats              int
	// NoSeats sends seats as null, as Polar does for a product that is not
	// per seat. Seats is then ignored.
	NoSeats bool
	// PendingSeats is the seat count scheduled for the next period.
	PendingSeats      *int
	CancelAtPeriodEnd bool
	PeriodStart       time.Time
	PeriodEnd         time.Time
	// ModifiedAt is the modified_at of the snapshots. When it is nil, each
	// snapshot uses the current time. Set it to send an older or newer
	// snapshot than an event that a test applied.
	ModifiedAt *time.Time
}

// Request is one request that the fake received.
type Request struct {
	Method         string
	SubscriptionID string
	Body           map[string]any
}

type failure struct {
	status int
	code   string
}

// Server serves GET and PATCH /subscriptions/{id} like Polar. A next_period
// seat update sets the pending seat count, and an update back to the current
// seats clears it. Any other seat update applies at once and drops the
// pending one. A cancel_at_period_end update changes only that flag.
type Server struct {
	t                    *testing.T
	mu                   sync.Mutex
	subs                 map[string]*Subscription
	requests             []Request
	failures             []failure
	losePurchaseResponse bool
	server               *httptest.Server
}

// NewServer starts the fake and points the Polar client at it.
func NewServer(t *testing.T) *Server {
	t.Helper()
	s := &Server{t: t, subs: map[string]*Subscription{}}
	s.server = httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(s.server.Close)
	t.Setenv("POLAR_ACCESS_TOKEN", "oat_test")
	t.Setenv("POLAR_API_BASE_URL", s.server.URL)
	return s
}

func (s *Server) URL() string {
	return s.server.URL
}

func (s *Server) Add(sub Subscription) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs[sub.ID] = &sub
}

func (s *Server) Subscription(id string) Subscription {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subs[id]
	require.True(s.t, ok, "unknown subscription %s", id)
	return *sub
}

// Update changes a subscription, for example to apply an outside change.
func (s *Server) Update(id string, change func(sub *Subscription)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	change(s.subs[id])
}

// FailNext answers the next count requests with the status and Polar error code.
func (s *Server) FailNext(count, status int, code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for range count {
		s.failures = append(s.failures, failure{status: status, code: code})
	}
}

// LoseNextPurchaseResponse applies the next purchase but closes its connection
// before returning the result, leaving the caller uncertain about the outcome.
func (s *Server) LoseNextPurchaseResponse() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.losePurchaseResponse = true
}

func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.requests...)
}

// Patches are the PATCH requests that the fake received, failed ones included.
func (s *Server) Patches() []Request {
	var patches []Request
	for _, request := range s.Requests() {
		if request.Method == http.MethodPatch {
			patches = append(patches, request)
		}
	}
	return patches
}

func (s *Server) serve(w http.ResponseWriter, req *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w.Header().Set("Polar-Version", req.Header.Get("Polar-Version"))

	id, ok := strings.CutPrefix(req.URL.Path, "/subscriptions/")
	sub := s.subs[id]
	if !ok || sub == nil {
		s.t.Errorf("unexpected Polar request %s %s", req.Method, req.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	request := Request{Method: req.Method, SubscriptionID: id}
	if req.Method == http.MethodPatch {
		require.NoError(s.t, json.NewDecoder(req.Body).Decode(&request.Body))
	}
	s.requests = append(s.requests, request)

	if len(s.failures) > 0 {
		next := s.failures[0]
		s.failures = s.failures[1:]
		s.writeJSON(w, next.status, map[string]any{"error": next.code, "detail": "details"})
		return
	}

	switch req.Method {
	case http.MethodGet:
	case http.MethodPatch:
		applyUpdate(sub, request.Body)
		if s.losePurchaseResponse && request.Body["proration_behavior"] == "invoice" {
			s.losePurchaseResponse = false
			conn, _, err := w.(http.Hijacker).Hijack()
			require.NoError(s.t, err)
			require.NoError(s.t, conn.Close())
			return
		}
	default:
		s.t.Errorf("unexpected Polar method %s", req.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.writeJSON(w, http.StatusOK, sub.snapshot())
}

func applyUpdate(sub *Subscription, body map[string]any) {
	if cancelAtPeriodEnd, ok := body["cancel_at_period_end"].(bool); ok {
		sub.CancelAtPeriodEnd = cancelAtPeriodEnd
		return
	}
	applySeatUpdate(sub, body)
}

func applySeatUpdate(sub *Subscription, body map[string]any) {
	seats := int(body["seats"].(float64))
	if body["proration_behavior"] != "next_period" {
		sub.Seats = seats
		sub.PendingSeats = nil
		return
	}
	if seats == sub.Seats {
		sub.PendingSeats = nil
		return
	}
	sub.PendingSeats = &seats
}

func (sub *Subscription) snapshot() map[string]any {
	modifiedAt := time.Now()
	if sub.ModifiedAt != nil {
		modifiedAt = *sub.ModifiedAt
	}
	snapshot := map[string]any{
		"id": sub.ID, "product_id": sub.ProductID, "status": sub.Status, "seats": sub.Seats,
		"cancel_at_period_end": sub.CancelAtPeriodEnd, "modified_at": modifiedAt.UTC().Format(time.RFC3339Nano),
		"current_period_start": sub.PeriodStart.UTC().Format(time.RFC3339),
		"current_period_end":   sub.PeriodEnd.UTC().Format(time.RFC3339),
		"external_customer_id": sub.ExternalCustomerID,
		"pending_update":       nil,
	}
	if sub.NoSeats {
		snapshot["seats"] = nil
	}
	if sub.PendingSeats != nil {
		snapshot["pending_update"] = map[string]any{
			"seats": *sub.PendingSeats, "applies_at": sub.PeriodEnd.UTC().Format(time.RFC3339),
		}
	}
	return snapshot
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	require.NoError(s.t, json.NewEncoder(w).Encode(body))
}

// AddSeatPlan puts an organization on an active per-seat Polar plan with the
// given seats and period end, and adds the matching subscription to the fake.
func (s *Server) AddSeatPlan(db *gorm.DB, orgID uuid.UUID, seats int, periodEnd time.Time) Subscription {
	s.t.Helper()
	// Match the whole-second precision of the subscription snapshots.
	periodEnd = periodEnd.UTC().Truncate(time.Second)
	product := "prod_team_" + uuid.NewString()
	catalog := billingmodels.BillingPlan{
		Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private",
		PerSeat: true, PriceCents: 2000, SeatLimit: new(50), PolarProductID: &product,
		CreditFrequency: "none", CreditExpiration: "never",
	}
	require.NoError(s.t, billingmodels.SaveBillingPlan(db, &catalog))
	_, err := billingmodels.EnsureOrganizationBillingPlan(db, orgID)
	require.NoError(s.t, err)

	sub := Subscription{
		ID: "sub_" + uuid.NewString(), ProductID: product, ExternalCustomerID: orgID.String(),
		Status: billingmodels.PolarSubscriptionStatusActive, Seats: seats,
		PeriodStart: periodEnd.AddDate(0, -1, 0), PeriodEnd: periodEnd,
	}
	require.NoError(s.t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", orgID).Updates(map[string]any{
		"plan_id": catalog.ID, "plan": catalog.Key, "plan_source": billingmodels.BillingPlanSourcePolar,
		"polar_subscription_id": sub.ID, "polar_subscription_status": billingmodels.PolarSubscriptionStatusActive,
		"purchased_seats": seats, "current_period_start": sub.PeriodStart, "current_period_end": sub.PeriodEnd,
		"cancel_at_period_end": false,
	}).Error)
	s.Add(sub)
	return sub
}
