package public

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/datadog"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

func findDatadogReceipt(items []adminDatadogWebhookReceipt, id string) *adminDatadogWebhookReceipt {
	for i := range items {
		if items[i].ID == id {
			return &items[i]
		}
	}
	return nil
}

func TestAdminDatadogWebhooks(t *testing.T) {
	server, registry, token := setupAdminTestServer(t)

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", "regular-datadog-webhooks@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/datadog/webhooks",
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("lists stored receipts without a payload", func(t *testing.T) {
		integrationID := uuid.New()
		receiptID, err := models.CreateDatadogWebhookReceipt(database.Conn(), models.DatadogWebhookReceipt{
			ID:                uuid.New(),
			ReceivedAt:        time.Now().UTC().Add(-time.Hour),
			IntegrationID:     integrationID,
			OrganizationID:    registry.Organization.ID,
			EventType:         "error_tracking_alert",
			AlertTransition:   "Triggered",
			AlertID:           "867",
			Service:           "checkout",
			IssueID:           "11111111-1111-4111-8111-111111111111",
			HTTPStatus:        http.StatusOK,
			Outcome:           models.DatadogWebhookOutcomeAccepted,
			SubscriptionCount: 1,
		})
		require.NoError(t, err)
		taskID := uuid.New()
		require.NoError(t, models.AppendDatadogWebhookTask(database.Conn(), receiptID, taskID))
		require.NoError(t, models.AppendDatadogWebhookTask(database.Conn(), receiptID, taskID))

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       "/admin/api/datadog/webhooks",
			authCookie: token,
		})
		assert.Equal(t, http.StatusOK, response.Code)
		assert.NotContains(t, response.Body.String(), "webhook-token")
		assert.NotContains(t, response.Body.String(), "payload")

		var body adminDatadogWebhooksResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		receipt := findDatadogReceipt(body.Items, receiptID.String())
		require.NotNil(t, receipt)
		assert.Equal(t, "error_tracking_alert", receipt.EventType)
		assert.Equal(t, "Triggered", receipt.AlertTransition)
		assert.Equal(t, "checkout", receipt.Service)
		assert.Equal(t, models.DatadogWebhookOutcomeAccepted, receipt.Outcome)
		assert.Equal(t, http.StatusOK, receipt.HTTPStatus)
		assert.Equal(t, []string{taskID.String()}, receipt.TaskIDs)
	})

	t.Run("stores a Datadog call without the body or token", func(t *testing.T) {
		integration, err := models.CreateIntegration(uuid.New(), registry.Organization.ID, "datadog", "Datadog", map[string]any{
			"site": "datadoghq.com",
		})
		require.NoError(t, err)
		integrationContext := contexts.NewIntegrationContext(database.Conn(), nil, integration, server.encryptor, server.registry, nil)
		require.NoError(t, integrationContext.SetSecret(datadog.WebhookSecretName, []byte("webhook-token")))

		issueID := "11111111-1111-4111-8111-111111111111"
		response := execRequest(server, requestParams{
			method: "POST",
			path:   "/integrations/" + integration.ID.String() + "/events",
			body: []byte(`{
				"event_type":"error_tracking_alert",
				"alert_transition":"Triggered",
				"alert_id":"867",
				"title":"secret-title",
				"body":"payload-secret-body",
				"tags":"service:checkout",
				"link":"https://app.datadoghq.com/error-tracking/issue/` + issueID + `"
			}`),
			headers: map[string]string{
				datadog.WebhookHeaderName: "webhook-token",
			},
		})
		assert.Equal(t, http.StatusOK, response.Code)

		var receipts []models.DatadogWebhookReceipt
		require.NoError(t, database.Conn().Where("integration_id = ?", integration.ID).Find(&receipts).Error)
		require.Len(t, receipts, 1)
		assert.Equal(t, "error_tracking_alert", receipts[0].EventType)
		assert.Equal(t, "Triggered", receipts[0].AlertTransition)
		assert.Equal(t, "867", receipts[0].AlertID)
		assert.Equal(t, "checkout", receipts[0].Service)
		assert.Equal(t, issueID, receipts[0].IssueID)
		assert.Equal(t, models.DatadogWebhookOutcomeNoSubscription, receipts[0].Outcome)
		assert.Equal(t, http.StatusOK, receipts[0].HTTPStatus)
		assert.NotContains(t, receipts[0].EventType+receipts[0].AlertTransition+receipts[0].Service+receipts[0].IssueID, "payload-secret-body")
		assert.NotContains(t, receipts[0].EventType+receipts[0].AlertID+receipts[0].Service, "webhook-token")
		assert.NotContains(t, receipts[0].AlertTransition+receipts[0].Service+receipts[0].IssueID, "secret-title")

		rejected := execRequest(server, requestParams{
			method: "POST",
			path:   "/integrations/" + integration.ID.String() + "/events",
			body:   []byte(`{"event_type":"error_tracking_alert","body":"payload-secret-body"}`),
			headers: map[string]string{
				datadog.WebhookHeaderName: "wrong-token",
			},
		})
		assert.Equal(t, http.StatusForbidden, rejected.Code)

		var rejectedReceipt models.DatadogWebhookReceipt
		require.NoError(t, database.Conn().
			Where("integration_id = ? AND outcome = ?", integration.ID, models.DatadogWebhookOutcomeRejected).
			First(&rejectedReceipt).Error)
		assert.Equal(t, http.StatusForbidden, rejectedReceipt.HTTPStatus)
		assert.Equal(t, "error_tracking_alert", rejectedReceipt.EventType)
		assert.NotContains(t, rejectedReceipt.EventType+rejectedReceipt.Service+rejectedReceipt.AlertID, "payload-secret-body")
	})

	t.Run("limits rejected receipts and does not read the body after the limit", func(t *testing.T) {
		integration, err := models.CreateIntegration(uuid.New(), registry.Organization.ID, "datadog", "Datadog limit", map[string]any{
			"site": "datadoghq.com",
		})
		require.NoError(t, err)
		integrationContext := contexts.NewIntegrationContext(database.Conn(), nil, integration, server.encryptor, server.registry, nil)
		require.NoError(t, integrationContext.SetSecret(datadog.WebhookSecretName, []byte("webhook-token")))

		path := "/integrations/" + integration.ID.String() + "/events"
		for range rejectedDatadogReceiptLimit {
			response := execRequest(server, requestParams{
				method: "POST",
				path:   path,
				body:   []byte(`{"event_type":"error_tracking_alert","body":"payload-secret-body"}`),
				headers: map[string]string{
					datadog.WebhookHeaderName: "wrong-token",
				},
			})
			assert.Equal(t, http.StatusForbidden, response.Code)
		}

		var stored int64
		require.NoError(t, database.Conn().Model(&models.DatadogWebhookReceipt{}).Where("integration_id = ?", integration.ID).Count(&stored).Error)
		assert.Equal(t, int64(rejectedDatadogReceiptLimit), stored)

		bodyRead := false
		request := httptest.NewRequest(http.MethodPost, path, readFailReader{onRead: func() { bodyRead = true }})
		request.Header.Set(datadog.WebhookHeaderName, "wrong-token")
		response := httptest.NewRecorder()
		server.Router.ServeHTTP(response, request)
		assert.Equal(t, http.StatusForbidden, response.Code)
		assert.False(t, bodyRead)

		require.NoError(t, database.Conn().Model(&models.DatadogWebhookReceipt{}).Where("integration_id = ?", integration.ID).Count(&stored).Error)
		assert.Equal(t, int64(rejectedDatadogReceiptLimit), stored)

		accepted := execRequest(server, requestParams{
			method: "POST",
			path:   path,
			body:   []byte(`{"event_type":"error_tracking_alert","alert_transition":"Triggered"}`),
			headers: map[string]string{
				datadog.WebhookHeaderName: "webhook-token",
			},
		})
		assert.Equal(t, http.StatusOK, accepted.Code)
		require.NoError(t, database.Conn().Model(&models.DatadogWebhookReceipt{}).Where("integration_id = ?", integration.ID).Count(&stored).Error)
		assert.Equal(t, int64(rejectedDatadogReceiptLimit+1), stored)
	})

	t.Run("does not wait on the receipt lock when the limit is full", func(t *testing.T) {
		integration, err := models.CreateIntegration(uuid.New(), registry.Organization.ID, "datadog", "Datadog lock", map[string]any{
			"site": "datadoghq.com",
		})
		require.NoError(t, err)
		integrationContext := contexts.NewIntegrationContext(database.Conn(), nil, integration, server.encryptor, server.registry, nil)
		require.NoError(t, integrationContext.SetSecret(datadog.WebhookSecretName, []byte("webhook-token")))
		for range rejectedDatadogReceiptLimit {
			_, err := models.CreateDatadogWebhookReceipt(database.Conn(), models.DatadogWebhookReceipt{
				IntegrationID:  integration.ID,
				OrganizationID: integration.OrganizationID,
				HTTPStatus:     http.StatusForbidden,
				Outcome:        models.DatadogWebhookOutcomeRejected,
			})
			require.NoError(t, err)
		}

		blocker := database.Conn().Begin()
		require.NoError(t, blocker.Error)
		t.Cleanup(func() { blocker.Rollback() })
		require.NoError(t, blocker.Exec(
			"SELECT pg_advisory_xact_lock(?)",
			models.RejectedDatadogWebhookReceiptLockKey(integration.ID),
		).Error)

		bodyRead := false
		request := httptest.NewRequest(
			http.MethodPost,
			"/integrations/"+integration.ID.String()+"/events",
			readFailReader{onRead: func() { bodyRead = true }},
		)
		request.Header.Set(datadog.WebhookHeaderName, "wrong-token")
		response := httptest.NewRecorder()
		started := time.Now()
		server.Router.ServeHTTP(response, request)

		assert.Equal(t, http.StatusForbidden, response.Code)
		assert.False(t, bodyRead)
		assert.Less(t, time.Since(started), time.Second)

		var stored int64
		require.NoError(t, database.Conn().Model(&models.DatadogWebhookReceipt{}).Where("integration_id = ?", integration.ID).Count(&stored).Error)
		assert.Equal(t, int64(rejectedDatadogReceiptLimit), stored)
	})

	t.Run("rejects an authenticated body larger than the webhook limit", func(t *testing.T) {
		integration, err := models.CreateIntegration(uuid.New(), registry.Organization.ID, "datadog", "Datadog size", map[string]any{
			"site": "datadoghq.com",
		})
		require.NoError(t, err)
		integrationContext := contexts.NewIntegrationContext(database.Conn(), nil, integration, server.encryptor, server.registry, nil)
		require.NoError(t, integrationContext.SetSecret(datadog.WebhookSecretName, []byte("webhook-token")))

		logs := captureDatadogWebhookLogs(t)
		response := execRequest(server, requestParams{
			method: "POST",
			path:   "/integrations/" + integration.ID.String() + "/events",
			body:   bytes.Repeat([]byte("a"), MaxEventSize+1),
			headers: map[string]string{
				datadog.WebhookHeaderName: "webhook-token",
			},
		})
		assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)

		var stored int64
		require.NoError(t, database.Conn().Model(&models.DatadogWebhookReceipt{}).Where("integration_id = ?", integration.ID).Count(&stored).Error)
		assert.Equal(t, int64(0), stored)

		lines := datadogWebhookLogLines(t, logs.String())
		require.Len(t, lines, 1)
		assert.Equal(t, "error", lines[0]["level"])
		assert.Equal(t, "Datadog webhook failed", lines[0]["msg"])
		assert.Equal(t, logging.WebhookLogType, lines[0]["type"])
		assert.Equal(t, logging.DatadogIntegration, lines[0]["integration"])
		assert.Equal(t, models.DatadogWebhookOutcomeFailed, lines[0]["outcome"])
		assert.Equal(t, "unknown", lines[0]["event_type"])
		assert.Equal(t, registry.Organization.ID.String(), lines[0]["organization_id"])
		assert.Equal(t, integration.ID.String(), lines[0]["integration_id"])
		assert.Contains(t, lines[0]["error"], "request body is too large")
	})
}

func TestRejectedDatadogReceiptLimitCache(t *testing.T) {
	cache := rejectedReceiptLimitCache{}
	id := uuid.New()
	now := time.Now().UTC()

	assert.False(t, cache.blocked(id, now))
	cache.markFull(id, now.Add(time.Minute))
	assert.True(t, cache.blocked(id, now))
	assert.False(t, cache.blocked(id, now.Add(2*time.Minute)))
	assert.False(t, cache.blocked(id, now.Add(2*time.Minute)))
}

func captureDatadogWebhookLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	logger := logging.DatadogWebhookLogger()
	previous := logger.Out
	buffer := &bytes.Buffer{}
	logger.SetOutput(buffer)
	t.Cleanup(func() {
		logger.SetOutput(previous)
	})
	return buffer
}

func datadogWebhookLogLines(t *testing.T, raw string) []map[string]any {
	t.Helper()

	lines := []map[string]any{}
	for _, line := range bytes.Split([]byte(raw), []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		payload := map[string]any{}
		require.NoError(t, json.Unmarshal(line, &payload))
		lines = append(lines, payload)
	}
	return lines
}

type readFailReader struct {
	onRead func()
}

func (r readFailReader) Read(_ []byte) (int, error) {
	if r.onRead != nil {
		r.onRead()
	}
	return 0, io.EOF
}
