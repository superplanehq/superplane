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
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
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
		hook := capturePolarWebhookLogs(t)
		body := []byte(`{"type":"order.updated","data":{"id":"order_1"}}`)
		rec := signedPolarWebhook(t, secret, body)
		server.handlePolarWebhook(rec.recorder, rec.request)
		assert.Equal(t, http.StatusAccepted, rec.recorder.Code)
		requirePolarWebhookWarning(t, hook, "order.updated")
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

func Test__HandlePolarWebhookSubscriptionCycled(t *testing.T) {
	r := support.Setup(t)
	server := &Server{}
	secret := "whsec_test-secret"
	t.Setenv("POLAR_WEBHOOK_SECRET", secret)
	t.Setenv("POLAR_BUSINESS_PRODUCT_ID", "prod_business")
	require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))

	subscriptionID := "sub_cycle_" + uuid.NewString()
	firstStart := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	firstEnd := firstStart.AddDate(0, 1, 0)
	nextStart := firstEnd
	nextEnd := nextStart.AddDate(0, 1, 0)
	renewedAt := time.Now().UTC().Truncate(time.Second)
	postSubscription(t, server, secret, "subscription.updated", subscriptionID, r.Organization.ID, firstStart, firstEnd, renewedAt.Add(-time.Minute))
	welcomeBefore := creditGrantCount(t, r.Organization.ID, models.LLMCreditGrantKindWelcome)
	firstPlan := requireSavedPeriod(t, r.Organization.ID, firstStart, firstEnd)

	postSubscription(t, server, secret, "subscription.cycled", subscriptionID, r.Organization.ID, nextStart, nextEnd, renewedAt)
	requireSavedPeriod(t, r.Organization.ID, nextStart, nextEnd)
	requireIncludedGrant(t, r.Organization.ID, subscriptionID, nextEnd, firstPlan.Catalog.CreditAmountCents)
	assert.Equal(t, welcomeBefore, creditGrantCount(t, r.Organization.ID, models.LLMCreditGrantKindWelcome))

	postSubscription(t, server, secret, "subscription.cycled", subscriptionID, r.Organization.ID, nextStart, nextEnd, renewedAt)
	requireSavedPeriod(t, r.Organization.ID, nextStart, nextEnd)
	requireIncludedGrant(t, r.Organization.ID, subscriptionID, nextEnd, firstPlan.Catalog.CreditAmountCents)

	postSubscription(t, server, secret, "subscription.updated", subscriptionID, r.Organization.ID, nextStart, nextEnd, renewedAt.Add(time.Second))
	requireSavedPeriod(t, r.Organization.ID, nextStart, nextEnd)
	requireIncludedGrant(t, r.Organization.ID, subscriptionID, nextEnd, firstPlan.Catalog.CreditAmountCents)
	assert.Equal(t, welcomeBefore, creditGrantCount(t, r.Organization.ID, models.LLMCreditGrantKindWelcome))

	staleStart := nextEnd
	staleEnd := staleStart.AddDate(0, 1, 0)
	postSubscription(t, server, secret, "subscription.cycled", subscriptionID, r.Organization.ID, staleStart, staleEnd, renewedAt.Add(-time.Hour))
	requireSavedPeriod(t, r.Organization.ID, nextStart, nextEnd)
	assert.Equal(t, 0, includedGrantCount(t, r.Organization.ID, subscriptionID, staleEnd))
}

func requireSavedPeriod(t *testing.T, orgID uuid.UUID, start, end time.Time) *models.OrganizationBillingPlan {
	t.Helper()
	plan, err := models.FindOrganizationBillingPlan(database.Conn(), orgID)
	require.NoError(t, err)
	require.NotNil(t, plan.CurrentPeriodStart)
	require.NotNil(t, plan.CurrentPeriodEnd)
	assert.True(t, plan.CurrentPeriodStart.Equal(start))
	assert.True(t, plan.CurrentPeriodEnd.Equal(end))
	return plan
}

func postSubscription(
	t *testing.T,
	server *Server,
	secret, eventType, subscriptionID string,
	orgID uuid.UUID,
	start, end, modified time.Time,
) {
	t.Helper()
	body := []byte(fmt.Sprintf(`{
		"type": %q,
		"data": {
			"id": %q,
			"status": "active",
			"product_id": "prod_business",
			"modified_at": %q,
			"current_period_start": %q,
			"current_period_end": %q,
			"customer": {"id": "cust_1", "external_id": %q}
		}
	}`, eventType, subscriptionID, modified.UTC().Format(time.RFC3339), start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339), orgID.String()))
	rec := signedPolarWebhook(t, secret, body)
	server.handlePolarWebhook(rec.recorder, rec.request)
	require.Equal(t, http.StatusAccepted, rec.recorder.Code)
}

func requireIncludedGrant(t *testing.T, orgID uuid.UUID, subscriptionID string, periodEnd time.Time, amountCents int64) {
	t.Helper()
	assert.Equal(t, 1, includedGrantCount(t, orgID, subscriptionID, periodEnd))
	grants, err := models.ListOrganizationLLMCreditGrants(database.Conn(), orgID)
	require.NoError(t, err)
	key := models.IncludedGrantKey(subscriptionID, periodEnd)
	for _, grant := range grants {
		if grant.Kind != models.LLMCreditGrantKindIncluded || grant.PolarOrderID == nil || *grant.PolarOrderID != key {
			continue
		}
		assert.Equal(t, models.CentsToMicros(amountCents), grant.AmountMicros)
		require.NotNil(t, grant.ExpiresAt)
		assert.True(t, grant.ExpiresAt.Equal(periodEnd.UTC()))
		return
	}
	require.Fail(t, "expected an included grant for the new period")
}

func includedGrantCount(t *testing.T, orgID uuid.UUID, subscriptionID string, periodEnd time.Time) int {
	t.Helper()
	grants, err := models.ListOrganizationLLMCreditGrants(database.Conn(), orgID)
	require.NoError(t, err)
	key := models.IncludedGrantKey(subscriptionID, periodEnd)
	count := 0
	for _, grant := range grants {
		if grant.Kind == models.LLMCreditGrantKindIncluded && grant.PolarOrderID != nil && *grant.PolarOrderID == key {
			count++
		}
	}
	return count
}

func creditGrantCount(t *testing.T, orgID uuid.UUID, kind string) int {
	t.Helper()
	grants, err := models.ListOrganizationLLMCreditGrants(database.Conn(), orgID)
	require.NoError(t, err)
	count := 0
	for _, grant := range grants {
		if grant.Kind == kind {
			count++
		}
	}
	return count
}

func capturePolarWebhookLogs(t *testing.T) *logtest.Hook {
	t.Helper()
	logger := log.StandardLogger()
	previous := logger.ReplaceHooks(make(log.LevelHooks))
	for _, hooks := range previous {
		for _, existing := range hooks {
			logger.AddHook(existing)
		}
	}
	hook := logtest.NewGlobal()
	t.Cleanup(func() {
		logger.ReplaceHooks(previous)
	})
	return hook
}

func requirePolarWebhookWarning(t *testing.T, hook *logtest.Hook, eventType string) {
	t.Helper()
	for _, entry := range hook.AllEntries() {
		if entry.Level != log.WarnLevel {
			continue
		}
		if entry.Message != "ignored unsupported polar webhook event" {
			continue
		}
		assert.Contains(t, fmt.Sprint(entry.Data["error"]), eventType)
		assert.Equal(t, eventType, entry.Data["polar_event_type"])
		return
	}
	require.Fail(t, "expected a warning for the unsupported polar event")
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
