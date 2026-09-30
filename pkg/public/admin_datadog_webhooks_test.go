package public

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/datadog"
	"github.com/superplanehq/superplane/pkg/jwt"
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
		assert.NotContains(t, rejectedReceipt.EventType+rejectedReceipt.Service+rejectedReceipt.AlertID, "payload-secret-body")
	})
}
