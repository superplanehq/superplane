package polar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
)

func checkoutPayload(id, status string) map[string]any {
	return map[string]any{
		"id":         id,
		"url":        "https://buy.polar.sh/" + id,
		"status":     status,
		"product_id": "prod_team",
		"seats":      4,
		"expires_at": time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano),
		"metadata":   map[string]any{CheckoutIntentMetadataKey: "intent-" + id},
	}
}

func Test__GetCheckoutReadsLifecycle(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour).UTC()
	server := httptest.NewServer(echoPolarVersion(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/checkouts/chk_1":
			assert.Equal(t, http.MethodGet, r.Method)
			payload := checkoutPayload("chk_1", "Confirmed")
			payload["expires_at"] = expiresAt.Format(time.RFC3339Nano)
			require.NoError(t, json.NewEncoder(w).Encode(payload))
		case "/checkouts/chk_other":
			require.NoError(t, json.NewEncoder(w).Encode(checkoutPayload("chk_1", "open")))
		case "/checkouts/chk_new_status":
			require.NoError(t, json.NewEncoder(w).Encode(checkoutPayload("chk_new_status", "reviewing")))
		default:
			http.Error(w, `{"error":"ResourceNotFound"}`, http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	client := NewClient(server.URL, "oat_test", server.Client())

	session, err := client.GetCheckout(context.Background(), "chk_1")
	require.NoError(t, err)
	assert.Equal(t, "chk_1", session.ID)
	assert.Equal(t, "https://buy.polar.sh/chk_1", session.URL)
	assert.Equal(t, CheckoutStatusConfirmed, session.Status)
	assert.True(t, session.Status.Payable(), "confirmed does not show payment success")
	require.NotNil(t, session.ExpiresAt)
	assert.True(t, session.ExpiresAt.Equal(expiresAt))
	assert.Equal(t, new(4), session.Seats)
	assert.Equal(t, "intent-chk_1", session.IntentID())

	_, err = client.GetCheckout(context.Background(), "chk_other")
	require.Error(t, err, "a response for another checkout must not be trusted")
	_, err = client.GetCheckout(context.Background(), "chk_new_status")
	require.Error(t, err, "an unknown status cannot be verified")
	_, err = client.GetCheckout(context.Background(), "chk_missing")
	assert.True(t, IsNotFound(err))
	_, err = client.GetCheckout(context.Background(), " ")
	require.Error(t, err)
}

func Test__CheckoutStatusMapsToIntentState(t *testing.T) {
	for status, want := range map[CheckoutStatus]string{
		CheckoutStatusSucceeded: billingmodels.BillingCheckoutStateSucceeded,
		CheckoutStatusExpired:   billingmodels.BillingCheckoutStateExpired,
		CheckoutStatusFailed:    billingmodels.BillingCheckoutStateFailed,
	} {
		state, final := status.IntentState()
		assert.True(t, final, status)
		assert.Equal(t, want, state, status)
		assert.False(t, status.Payable(), status)
	}
	for _, status := range []CheckoutStatus{CheckoutStatusOpen, CheckoutStatusConfirmed} {
		_, final := status.IntentState()
		assert.False(t, final, status)
		assert.True(t, status.Payable(), status)
	}
}

func Test__ListCheckoutsFiltersByCustomerAndProductAcrossPages(t *testing.T) {
	var pages []string
	server := httptest.NewServer(echoPolarVersion(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/checkouts/", r.URL.Path)
		query := r.URL.Query()
		assert.Equal(t, "org-1", query.Get("external_customer_id"))
		if len(pages) < 2 {
			assert.Equal(t, "prod_team", query.Get("product_id"))
		} else {
			assert.NotContains(t, query, "product_id")
		}
		assert.Equal(t, "100", query.Get("limit"))
		assert.Empty(t, query.Get("status"), "a lost create can leave a checkout in any status")
		pages = append(pages, query.Get("page"))
		page, err := strconv.Atoi(query.Get("page"))
		require.NoError(t, err)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"items":      []any{checkoutPayload("chk_page_"+query.Get("page"), "open")},
			"pagination": map[string]int{"max_page": 2, "total_count": 2 + page},
		}))
	}))
	t.Cleanup(server.Close)
	client := NewClient(server.URL, "oat_test", server.Client())

	sessions, err := client.ListCheckouts(context.Background(), "org-1", "prod_team")
	require.NoError(t, err)
	require.Len(t, sessions, 2)
	assert.Equal(t, []string{"1", "2"}, pages)
	assert.Equal(t, "chk_page_1", sessions[0].ID)
	assert.Equal(t, "intent-chk_page_2", sessions[1].IntentID())

	_, err = client.ListCheckouts(context.Background(), "", "prod_team")
	require.Error(t, err)
	sessions, err = client.ListCheckouts(context.Background(), "org-1", "")
	require.NoError(t, err)
	require.Len(t, sessions, 2)
}

func Test__ListCheckoutsRejectsUnverifiableItems(t *testing.T) {
	for name, items := range map[string]any{
		"missing items":  nil,
		"no id":          []any{map[string]any{"status": "open"}},
		"unknown status": []any{checkoutPayload("chk_1", "reviewing")},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(echoPolarVersion(func(w http.ResponseWriter, _ *http.Request) {
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "pagination": map[string]int{"max_page": 1}}))
			}))
			t.Cleanup(server.Close)
			client := NewClient(server.URL, "oat_test", server.Client())
			_, err := client.ListCheckouts(context.Background(), "org-1", "prod_team")
			require.Error(t, err)
		})
	}
}
