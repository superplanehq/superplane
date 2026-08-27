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

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

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

	t.Run("unusable payload", func(t *testing.T) {
		t.Setenv("POLAR_WEBHOOK_SECRET", secret)
		body := []byte(`{"type":"order.paid","data":{"id":""}}`)
		rec := signedPolarWebhook(t, secret, body)
		server.handlePolarWebhook(rec.recorder, rec.request)
		assert.Equal(t, http.StatusAccepted, rec.recorder.Code)
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
