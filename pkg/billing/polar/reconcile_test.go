package polar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListAndReconcileOrdersGrantsPaidPackOnce(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orderID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/orders/", req.URL.Path)
		require.NoError(t, json.NewEncoder(w).Encode(paidPackListPayload(orderID, r.Organization.ID.String())))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	orders, err := client.ListAndReconcileOrders(context.Background(), db, r.Organization.ID)
	require.NoError(t, err)
	require.Len(t, orders, 1)
	assert.Equal(t, orderID, orders[0].ID)

	orders, err = client.ListAndReconcileOrders(context.Background(), db, r.Organization.ID)
	require.NoError(t, err)
	require.Len(t, orders, 1)

	grant, err := models.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(10000), grant.AmountMicros)

	summary, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(models.DefaultWelcomeGrantCents)+models.CentsToMicros(10000), summary.GrantMicros)
}

func Test__ListAndReconcileOrdersHydratesThinListedOrder(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orderID := uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/orders/"+orderID {
			require.NoError(t, json.NewEncoder(w).Encode(paidPackOrderPayload(orderID, r.Organization.ID.String())))
			return
		}
		assert.Equal(t, "/orders/", req.URL.Path)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":           orderID,
					"created_at":   "2026-09-08T12:00:00Z",
					"status":       "paid",
					"total_amount": 10000,
					"product":      map[string]any{"name": "$100 pack"},
				},
			},
			"pagination": map[string]any{"max_page": 1},
		}))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err := client.ListAndReconcileOrders(context.Background(), db, r.Organization.ID)
	require.NoError(t, err)

	grant, err := models.FindLLMCreditGrantByPolarOrderID(db, orderID)
	require.NoError(t, err)
	assert.Equal(t, models.CentsToMicros(10000), grant.AmountMicros)
}

func Test__ListAndReconcileOrdersAppliesRefund(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	orderID := uuid.NewString()
	require.NoError(t, ApplyOrderPaid(context.Background(), db, paidPackEvent(r.Organization.ID, orderID, 10000), nil))
	before, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/orders/", req.URL.Path)
		payload := paidPackListPayload(orderID, r.Organization.ID.String())
		item := payload["items"].([]map[string]any)[0]
		item["status"] = "refunded"
		item["refunded_amount"] = int64(10000)
		require.NoError(t, json.NewEncoder(w).Encode(payload))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL, "oat_test", server.Client())
	_, err = client.ListAndReconcileOrders(context.Background(), db, r.Organization.ID)
	require.NoError(t, err)

	after, err := models.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, before.GrantMicros-models.CentsToMicros(10000), after.GrantMicros)
}

func paidPackListPayload(orderID, orgID string) map[string]any {
	return map[string]any{
		"items": []map[string]any{
			paidPackOrderPayload(orderID, orgID),
		},
		"pagination": map[string]any{"max_page": 1},
	}
}

func paidPackOrderPayload(orderID, orgID string) map[string]any {
	return map[string]any{
		"id":             orderID,
		"created_at":     "2026-09-08T12:00:00Z",
		"status":         "paid",
		"billing_reason": "purchase",
		"total_amount":   10000,
		"product_price":  map[string]any{"amount_type": "fixed", "price_amount": 10000},
		"customer": map[string]any{
			"id":          "cust_polar_1",
			"external_id": orgID,
		},
		"product": map[string]any{
			"id":   "prod_100",
			"name": "$100 pack",
			"metadata": map[string]any{
				"superplane_credit_pack": true,
			},
			"prices": []map[string]any{
				{"amount_type": "fixed", "price_amount": 10000},
			},
		},
	}
}
