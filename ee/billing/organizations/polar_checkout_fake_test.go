package organizations

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	polartest "github.com/superplanehq/superplane/ee/billing/testsupport/polar"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// fakePolarCheckouts is a stateful Polar fake for the checkout endpoints and
// the customer lookup of one organization. It records every checkout that a
// create call makes, so a test can see a checkout that the client never heard of.
type fakePolarCheckouts struct {
	t     *testing.T
	orgID uuid.UUID

	mu    sync.Mutex
	items []*fakePolarCheckout
	posts int

	// createDelay keeps a create call open, so concurrent requests overlap.
	createDelay time.Duration
	// onCreate inspects the body of each create call.
	onCreate func(body map[string]any)
	// rejectStatus answers a create call with an error before Polar records anything.
	rejectStatus int
	// loseResponse ("502" or "drop") makes the client lose the answer of a
	// create call. recordLost tells whether Polar kept the checkout.
	loseResponse string
	recordLost   bool
	// requireUnlockedOrganization fails the test when a Polar call arrives
	// while any connection holds the organization row lock.
	requireUnlockedOrganization bool
}

type fakePolarCheckout struct {
	ID        string
	Status    string
	ProductID string
	Seats     *int
	Metadata  map[string]any
	ExpiresAt time.Time
}

func newFakePolarCheckouts(t *testing.T, orgID uuid.UUID) *fakePolarCheckouts {
	t.Helper()
	return &fakePolarCheckouts{t: t, orgID: orgID, recordLost: true}
}

// serve starts a Polar server. Requests that are not about checkouts or the
// customer go to other, which can be nil.
func (f *fakePolarCheckouts) serve(other http.HandlerFunc) {
	f.t.Helper()
	server := polartest.APIServer(f.t, func(w http.ResponseWriter, r *http.Request) {
		if f.handle(w, r) {
			return
		}
		if other == nil {
			f.t.Errorf("unexpected Polar request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		other(w, r)
	})
	polartest.UseServer(f.t, server)
}

func (f *fakePolarCheckouts) handle(w http.ResponseWriter, r *http.Request) bool {
	switch {
	case r.URL.Path == "/customers/external/"+f.orgID.String():
		writeFakeJSON(f.t, w, map[string]any{"id": "cust_fake", "external_id": f.orgID.String()})
	case r.URL.Path == "/checkouts/" && r.Method == http.MethodPost:
		f.assertOrganizationUnlocked()
		f.create(w, r)
	case r.URL.Path == "/checkouts/" && r.Method == http.MethodGet:
		f.assertOrganizationUnlocked()
		f.list(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/checkouts/"):
		f.assertOrganizationUnlocked()
		f.read(w, strings.TrimPrefix(r.URL.Path, "/checkouts/"))
	default:
		return false
	}
	return true
}

func (f *fakePolarCheckouts) create(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	assert.NoError(f.t, json.NewDecoder(r.Body).Decode(&body))
	f.mu.Lock()
	f.posts++
	f.mu.Unlock()
	if f.createDelay > 0 {
		time.Sleep(f.createDelay)
	}
	if f.onCreate != nil {
		f.onCreate(body)
	}
	if f.rejectStatus != 0 {
		http.Error(w, `{"error":"rejected"}`, f.rejectStatus)
		return
	}

	checkout := &fakePolarCheckout{
		ID:        "chk_" + uuid.NewString(),
		Status:    "open",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if products, ok := body["products"].([]any); ok && len(products) == 1 {
		checkout.ProductID, _ = products[0].(string)
	}
	if seats, ok := body["seats"].(float64); ok {
		checkout.Seats = new(int(seats))
	}
	checkout.Metadata, _ = body["metadata"].(map[string]any)

	if f.loseResponse != "" && !f.recordLost {
		f.loseCreateResponse(w)
		return
	}
	f.mu.Lock()
	f.items = append(f.items, checkout)
	f.mu.Unlock()
	if f.loseResponse != "" {
		f.loseCreateResponse(w)
		return
	}
	writeFakeJSON(f.t, w, f.payload(checkout))
}

func (f *fakePolarCheckouts) loseCreateResponse(w http.ResponseWriter) {
	if f.loseResponse == "drop" {
		conn, _, err := w.(http.Hijacker).Hijack()
		require.NoError(f.t, err)
		require.NoError(f.t, conn.Close())
		return
	}
	w.WriteHeader(http.StatusBadGateway)
}

func (f *fakePolarCheckouts) read(w http.ResponseWriter, id string) {
	f.mu.Lock()
	index := slices.IndexFunc(f.items, func(item *fakePolarCheckout) bool { return item.ID == id })
	var checkout *fakePolarCheckout
	if index >= 0 {
		checkout = f.items[index]
	}
	f.mu.Unlock()
	if checkout == nil {
		http.Error(w, `{"error":"ResourceNotFound"}`, http.StatusNotFound)
		return
	}
	writeFakeJSON(f.t, w, f.payload(checkout))
}

func (f *fakePolarCheckouts) list(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	assert.Equal(f.t, f.orgID.String(), query.Get("external_customer_id"))
	f.mu.Lock()
	items := []any{}
	for _, checkout := range slices.Backward(f.items) {
		if product := query.Get("product_id"); product != "" && product != checkout.ProductID {
			continue
		}
		items = append(items, f.payload(checkout))
	}
	f.mu.Unlock()
	writeFakeJSON(f.t, w, map[string]any{"items": items, "pagination": map[string]int{"max_page": 1, "total_count": len(items)}})
}

func (f *fakePolarCheckouts) payload(checkout *fakePolarCheckout) map[string]any {
	return map[string]any{
		"id":                   checkout.ID,
		"url":                  "https://polar.sh/checkout/" + checkout.ID,
		"status":               checkout.Status,
		"product_id":           checkout.ProductID,
		"seats":                checkout.Seats,
		"metadata":             checkout.Metadata,
		"expires_at":           checkout.ExpiresAt.UTC().Format(time.RFC3339Nano),
		"customer_id":          "cust_fake",
		"external_customer_id": f.orgID.String(),
	}
}

// assertOrganizationUnlocked proves that no transaction holds the organization
// row lock while a request is in flight to Polar.
func (f *fakePolarCheckouts) assertOrganizationUnlocked() {
	if !f.requireUnlockedOrganization {
		return
	}
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		return tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).
			First(&models.Organization{}, "id = ?", f.orgID).Error
	})
	assert.NoError(f.t, err, "the organization row lock must not be held during a Polar call")
}

// Counters and state changes for tests.

func (f *fakePolarCheckouts) createCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.posts
}

func (f *fakePolarCheckouts) checkouts() []*fakePolarCheckout {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.items)
}

// seed adds a checkout that SuperPlane never created, such as a session from
// before intents existed.
func (f *fakePolarCheckouts) seed(productID string, seats *int, metadata map[string]any) *fakePolarCheckout {
	checkout := &fakePolarCheckout{
		ID:        "chk_seeded_" + uuid.NewString(),
		Status:    "open",
		ProductID: productID,
		Seats:     seats,
		Metadata:  metadata,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	f.mu.Lock()
	f.items = append(f.items, checkout)
	f.mu.Unlock()
	return checkout
}

func (f *fakePolarCheckouts) setStatus(checkout *fakePolarCheckout, status string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	checkout.Status = status
}

func writeFakeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	assert.NoError(t, json.NewEncoder(w).Encode(payload))
}
