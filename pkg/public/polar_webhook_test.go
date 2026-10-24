package public

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/billing/polar"
	"github.com/superplanehq/superplane/pkg/billing/seatrenewal"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/polarfake"
)

func Test__WebhookRestoresRenewalReductionAfterLostPurchaseResponse(t *testing.T) {
	r := support.Setup(t)
	t.Cleanup(r.Close)
	t.Setenv(models.BillingEnabledEnv, "yes")
	t.Setenv("POLAR_WEBHOOK_SECRET", "whsec_test-secret")
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeatureInTransaction(db, r.Organization.ID, features.FeatureNewBilling))
	provider := polarfake.NewServer(t)
	t.Cleanup(seatrenewal.Wait)
	sub := provider.AddSeatPlan(db, r.Organization.ID, 3, time.Now().AddDate(0, 1, 0))
	provider.Update(sub.ID, func(sub *polarfake.Subscription) { sub.PendingSeats = new(1) })
	provider.LoseNextPurchaseResponse()
	response, err := organizations.PurchaseSeats(t.Context(), r.Organization.ID.String(), &pb.PurchaseSeatsRequest{
		AdditionalSeats: 1, ExpectedPurchasedSeats: 3,
	}, r.User.String())
	require.NoError(t, err)
	require.Equal(t, "awaiting_confirmation", response.Purchase.Status)
	require.Nil(t, provider.Subscription(sub.ID).PendingSeats, "the applied purchase cleared the reduction")

	snapshot, err := polar.NewClientFromEnv().GetSubscription(t.Context(), sub.ID)
	require.NoError(t, err)
	stale := *snapshot
	stale.ModifiedAt.Time = time.Now().Add(-time.Hour)
	staleBody, err := json.Marshal(map[string]any{"type": "subscription.updated", "data": stale})
	require.NoError(t, err)
	staleWebhook := signedPolarWebhook(t, "whsec_test-secret", staleBody)
	(&Server{}).handlePolarWebhook(staleWebhook.recorder, staleWebhook.request)
	require.Equal(t, http.StatusAccepted, staleWebhook.recorder.Code)
	seatrenewal.Wait()
	purchase, err := models.FindLatestSeatPurchase(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, models.SeatPurchaseStatusUnknown, purchase.Status, "a stale snapshot cannot settle the purchase")
	require.Nil(t, provider.Subscription(sub.ID).PendingSeats)

	body, err := json.Marshal(map[string]any{"type": "subscription.updated", "data": snapshot})
	require.NoError(t, err)
	webhook := signedPolarWebhook(t, "whsec_test-secret", body)
	(&Server{}).handlePolarWebhook(webhook.recorder, webhook.request)
	require.Equal(t, http.StatusAccepted, webhook.recorder.Code)
	seatrenewal.Wait()
	purchase, err = models.FindLatestSeatPurchase(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, models.SeatPurchaseStatusSucceeded, purchase.Status)
	assert.Equal(t, new(1), provider.Subscription(sub.ID).PendingSeats)
	plan, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Equal(t, new(1), plan.RenewalSeatsScheduled)
	// A duplicate delivery checks the state again without sending a new update.
	patchCount := len(provider.Patches())
	replay := signedPolarWebhook(t, "whsec_test-secret", body)
	(&Server{}).handlePolarWebhook(replay.recorder, replay.request)
	require.Equal(t, http.StatusAccepted, replay.recorder.Code)
	seatrenewal.Wait()
	assert.Len(t, provider.Patches(), patchCount)
}

func Test__HandlePolarWebhook(t *testing.T) {
	r := support.Setup(t)
	server := &Server{}
	secret := "whsec_test-secret"

	t.Run("missing secret", func(t *testing.T) {
		t.Setenv("POLAR_WEBHOOK_SECRET", "")
		body := []byte(`{"type":"order.paid","data":{"id":"order_1"}}`)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/polar/webhooks", bytes.NewReader(body))
		server.handlePolarWebhook(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("invalid signature", func(t *testing.T) {
		t.Setenv("POLAR_WEBHOOK_SECRET", secret)
		body := []byte(`{"type":"order.paid","data":{"id":"order_1"}}`)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/polar/webhooks", bytes.NewReader(body))
		req.Header.Set("Webhook-Id", "msg_1")
		req.Header.Set("Webhook-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
		req.Header.Set("Webhook-Signature", "v1,AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
		server.handlePolarWebhook(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("unsupported event", func(t *testing.T) {
		t.Setenv("POLAR_WEBHOOK_SECRET", secret)
		body := []byte(`{"type":"order.updated","data":{"id":"order_1"}}`)
		rec := signedPolarWebhook(t, secret, body)
		server.handlePolarWebhook(rec.recorder, rec.request)
		assert.Equal(t, http.StatusAccepted, rec.recorder.Code)
	})

	t.Run("subscription cycled advances the billing period", func(t *testing.T) {
		t.Setenv("POLAR_WEBHOOK_SECRET", secret)
		require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
		firstEnd := time.Now().UTC().Truncate(time.Second).AddDate(0, 1, 0)
		renewedEnd := firstEnd.AddDate(0, 1, 0)
		subscriptionBody := func(eventType string, start, end time.Time) []byte {
			return []byte(fmt.Sprintf(`{
				"type": %q,
				"data": {
					"id": "sub_cycle",
					"status": "active",
					"product_id": "prod_business",
					"modified_at": %q,
					"current_period_start": %q,
					"current_period_end": %q,
					"customer": {"id": "cust_1", "external_id": %q}
				}
			}`, eventType, start.Format(time.RFC3339), start.Format(time.RFC3339), end.Format(time.RFC3339), r.Organization.ID.String()))
		}

		for _, body := range [][]byte{
			subscriptionBody("subscription.created", firstEnd.AddDate(0, -1, 0), firstEnd),
			subscriptionBody("subscription.cycled", firstEnd, renewedEnd),
		} {
			rec := signedPolarWebhook(t, secret, body)
			server.handlePolarWebhook(rec.recorder, rec.request)
			require.Equal(t, http.StatusAccepted, rec.recorder.Code)
		}

		plan, err := models.FindOrganizationBillingPlan(database.Conn(), r.Organization.ID)
		require.NoError(t, err)
		require.NotNil(t, plan.CurrentPeriodStart)
		require.NotNil(t, plan.CurrentPeriodEnd)
		assert.True(t, firstEnd.Equal(*plan.CurrentPeriodStart))
		assert.True(t, renewedEnd.Equal(*plan.CurrentPeriodEnd))
	})

	t.Run("unusable payload", func(t *testing.T) {
		t.Setenv("POLAR_WEBHOOK_SECRET", secret)
		body := []byte(`{"type":"order.paid","data":{"id":""}}`)
		rec := signedPolarWebhook(t, secret, body)
		server.handlePolarWebhook(rec.recorder, rec.request)
		assert.Equal(t, http.StatusAccepted, rec.recorder.Code)
	})

	t.Run("subscription updated stores purchased seats", func(t *testing.T) {
		t.Setenv("POLAR_WEBHOOK_SECRET", secret)
		db := database.Conn()
		product := "product_" + uuid.NewString()
		catalog := models.BillingPlan{Key: "team_" + uuid.NewString()[:8], Name: "Team", Category: "paid", Visibility: "private", PerSeat: true, SeatLimit: new(50), PolarProductID: &product, CreditFrequency: "none", CreditExpiration: "never"}
		require.NoError(t, models.SaveBillingPlan(db, &catalog))
		body := []byte(fmt.Sprintf(`{"type":"subscription.updated","data":{"id":"sub_seats","status":"active","product_id":%q,"seats":3,"customer":{"external_id":%q}}}`, product, r.Organization.ID.String()))
		for range 2 {
			rec := signedPolarWebhook(t, secret, body)
			server.handlePolarWebhook(rec.recorder, rec.request)
			require.Equal(t, http.StatusAccepted, rec.recorder.Code)
		}
		stored, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
		require.NoError(t, err)
		require.NotNil(t, stored.PurchasedSeats)
		require.Equal(t, 3, *stored.PurchasedSeats)
	})

	t.Run("grants credit for paid order", func(t *testing.T) {
		t.Setenv("POLAR_WEBHOOK_SECRET", secret)
		orderID := uuid.NewString()
		body := []byte(fmt.Sprintf(`{
			"type": "order.paid",
			"data": {
				"id": %q,
				"customer": {"id": "cust_1", "external_id": %q},
				"product": {
					"id": "prod_1",
					"name": "Hosted credit 25",
					"metadata": {"superplane_credit_pack": true},
					"prices": [{"amount_type": "fixed", "price_amount": 2500}]
				}
			}
		}`, orderID, r.Organization.ID.String()))
		rec := signedPolarWebhook(t, secret, body)
		server.handlePolarWebhook(rec.recorder, rec.request)
		assert.Equal(t, http.StatusAccepted, rec.recorder.Code)

		var payload map[string]string
		require.NoError(t, json.Unmarshal(rec.recorder.Body.Bytes(), &payload))
		assert.Equal(t, "accepted", payload["status"])

		grant, err := models.FindLLMCreditGrantByPolarOrderID(database.Conn(), orderID)
		require.NoError(t, err)
		assert.Equal(t, models.CentsToMicros(2500), grant.AmountMicros)
	})

	t.Run("grants credit when the webhook api version differs", func(t *testing.T) {
		t.Setenv("POLAR_WEBHOOK_SECRET", secret)
		t.Setenv("POLAR_API_VERSION", "2026-10")
		orderID := uuid.NewString()
		body := []byte(fmt.Sprintf(`{
			"type": "order.paid",
			"api_version": "2026-04",
			"data": {
				"id": %q,
				"customer": {"id": "cust_1", "external_id": %q},
				"product": {
					"id": "prod_1",
					"name": "Hosted credit 25",
					"metadata": {"superplane_credit_pack": true},
					"prices": [{"amount_type": "fixed", "price_amount": 2500}]
				}
			}
		}`, orderID, r.Organization.ID.String()))
		rec := signedPolarWebhook(t, secret, body)
		rec.request.Header.Set("Webhook-Api-Version", "2026-04")
		server.handlePolarWebhook(rec.recorder, rec.request)
		assert.Equal(t, http.StatusAccepted, rec.recorder.Code)

		grant, err := models.FindLLMCreditGrantByPolarOrderID(database.Conn(), orderID)
		require.NoError(t, err)
		assert.Equal(t, models.CentsToMicros(2500), grant.AmountMicros)
	})

	t.Run("grants quoted credit from signed order metadata excluding fees and tax", func(t *testing.T) {
		t.Setenv("POLAR_WEBHOOK_SECRET", secret)
		orderID := uuid.NewString()
		body := []byte(fmt.Sprintf(`{
			"type": "order.paid",
			"data": {
				"id": %q,
				"billing_reason": "purchase",
				"currency": "usd",
				"net_amount": 10579,
				"tax_amount": 2116,
				"total_amount": 12695,
				"metadata": {
					"superplane_credit_purchase_version": "1",
					"superplane_credit_amount_cents": "10000",
					"superplane_credit_subtotal_cents": "10579"
				},
				"customer": {"id": "cust_1", "external_id": %q},
				"product": {"id": "prod_1", "metadata": {"superplane_credit_pack": true}},
				"items": [{"amount": 10579, "product_price_id": "price_adhoc"}]
			}
		}`, orderID, r.Organization.ID.String()))
		rec := signedPolarWebhook(t, secret, body)
		server.handlePolarWebhook(rec.recorder, rec.request)
		require.Equal(t, http.StatusAccepted, rec.recorder.Code)
		grant, err := models.FindLLMCreditGrantByPolarOrderID(database.Conn(), orderID)
		require.NoError(t, err)
		assert.Equal(t, models.CentsToMicros(10000), grant.AmountMicros)
	})

	t.Run("apply failure", func(t *testing.T) {
		t.Setenv("POLAR_WEBHOOK_SECRET", secret)
		body := []byte(`{
			"type": "order.paid",
			"data": {
				"id": "order_bad_org",
				"customer": {"id": "cust_1", "external_id": "not-a-uuid"},
				"product": {
					"id": "prod_1",
					"metadata": {"superplane_credit_pack": true},
					"prices": [{"amount_type": "fixed", "price_amount": 2500}]
				}
			}
		}`)
		rec := signedPolarWebhook(t, secret, body)
		server.handlePolarWebhook(rec.recorder, rec.request)
		assert.Equal(t, http.StatusAccepted, rec.recorder.Code)
	})

	t.Run("unknown organization is accepted", func(t *testing.T) {
		t.Setenv("POLAR_WEBHOOK_SECRET", secret)
		body := []byte(fmt.Sprintf(`{
			"type": "order.paid",
			"data": {
				"id": %q,
				"customer": {"id": "cust_1", "external_id": %q},
				"product": {
					"id": "prod_1",
					"metadata": {"superplane_credit_pack": true},
					"prices": [{"amount_type": "fixed", "price_amount": 2500}]
				}
			}
		}`, uuid.NewString(), uuid.NewString()))
		rec := signedPolarWebhook(t, secret, body)
		server.handlePolarWebhook(rec.recorder, rec.request)
		assert.Equal(t, http.StatusAccepted, rec.recorder.Code)
	})

	t.Run("refund reverses grant", func(t *testing.T) {
		t.Setenv("POLAR_WEBHOOK_SECRET", secret)
		orderID := uuid.NewString()
		paid := []byte(fmt.Sprintf(`{
			"type": "order.paid",
			"data": {
				"id": %q,
				"customer": {"id": "cust_1", "external_id": %q},
				"product": {
					"id": "prod_1",
					"metadata": {"superplane_credit_pack": true},
					"prices": [{"amount_type": "fixed", "price_amount": 2500}]
				}
			}
		}`, orderID, r.Organization.ID.String()))
		rec := signedPolarWebhook(t, secret, paid)
		server.handlePolarWebhook(rec.recorder, rec.request)
		require.Equal(t, http.StatusAccepted, rec.recorder.Code)

		refund := []byte(fmt.Sprintf(`{
			"type": "order.refunded",
			"data": {
				"id": %q,
				"status": "refunded",
				"refunded_amount": 2500,
				"customer": {"id": "cust_1", "external_id": %q},
				"product": {
					"id": "prod_1",
					"metadata": {"superplane_credit_pack": true},
					"prices": [{"amount_type": "fixed", "price_amount": 2500}]
				}
			}
		}`, orderID, r.Organization.ID.String()))
		rec = signedPolarWebhook(t, secret, refund)
		server.handlePolarWebhook(rec.recorder, rec.request)
		assert.Equal(t, http.StatusAccepted, rec.recorder.Code)

		_, err := models.FindLLMCreditRefundByPolarRefundID(database.Conn(), orderID+":full")
		require.NoError(t, err)
	})

	t.Run("refund before grant is retried", func(t *testing.T) {
		t.Setenv("POLAR_WEBHOOK_SECRET", secret)
		orderID := uuid.NewString()
		refund := []byte(fmt.Sprintf(`{
			"type": "order.refunded",
			"data": {
				"id": %q,
				"status": "refunded",
				"refunded_amount": 2500,
				"customer": {"id": "cust_1", "external_id": %q},
				"product": {
					"id": "prod_1",
					"metadata": {"superplane_credit_pack": true},
					"prices": [{"amount_type": "fixed", "price_amount": 2500}]
				}
			}
		}`, orderID, r.Organization.ID.String()))
		rec := signedPolarWebhook(t, secret, refund)
		server.handlePolarWebhook(rec.recorder, rec.request)
		assert.Equal(t, http.StatusInternalServerError, rec.recorder.Code)
	})
}

type signedPolarWebhookCall struct {
	request  *http.Request
	recorder *httptest.ResponseRecorder
}

func signedPolarWebhook(t *testing.T, secret string, body []byte) signedPolarWebhookCall {
	t.Helper()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("msg_1"))
	mac.Write([]byte("."))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	signature := fmt.Sprintf("v1,%s", base64.StdEncoding.EncodeToString(mac.Sum(nil)))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/polar/webhooks", bytes.NewReader(body))
	req.Header.Set("Webhook-Id", "msg_1")
	req.Header.Set("Webhook-Timestamp", timestamp)
	req.Header.Set("Webhook-Signature", signature)
	return signedPolarWebhookCall{
		request:  req,
		recorder: httptest.NewRecorder(),
	}
}
