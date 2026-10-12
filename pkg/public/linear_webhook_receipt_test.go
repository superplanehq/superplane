package public

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

func Test__HandleWebhook_RecordsLinearReceipt(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	const triggerName = "linear-receipt-trigger"
	r.Registry.Triggers[triggerName] = impl.NewDummyTrigger(impl.DummyTriggerOptions{
		Name: triggerName,
		HandleWebhookFunc: func(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
			mode, _ := ctx.Configuration.(map[string]any)["mode"].(string)
			switch mode {
			case "reject":
				return http.StatusForbidden, nil, fmt.Errorf("invalid webhook signature")
			case "ignore":
				return http.StatusOK, nil, nil
			default:
				if err := ctx.Events.Emit(linear.IssuePayloadType, map[string]any{"action": "create"}); err != nil {
					return http.StatusInternalServerError, nil, err
				}
				return http.StatusOK, nil, nil
			}
		},
	})

	signer := jwt.NewSigner("test")
	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		signer,
		support.NewOIDCProvider(),
		"",
		"http://localhost",
		"http://localhost",
		"test",
		"/app/templates",
		r.AuthService, false,
	)
	require.NoError(t, err)

	body := []byte(`{
		"action": "create",
		"type": "Issue",
		"actor": {"email": "ada@example.com"},
		"url": "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails",
		"data": {
			"id": "2174add1-f7c8-44e3-bbf3-2d60b5ea8bc9",
			"identifier": "ENG-142",
			"description": "secret-description",
			"team": {"key": "ENG"}
		}
	}`)

	t.Run("stores the event and stamps the canvas event", func(t *testing.T) {
		webhookID, canvasID, nodeID := linearReceiptFixture(t, r, triggerName, "emit")
		response := postLinearWebhook(server, webhookID, body)
		require.Equal(t, http.StatusOK, response.Code)

		receipt := requireLinearReceipt(t, webhookID)
		assert.Equal(t, models.LinearWebhookOutcomeAccepted, receipt.Outcome)
		assert.Equal(t, http.StatusOK, receipt.HTTPStatus)
		assert.Equal(t, "Issue", receipt.EventType)
		assert.Equal(t, "create", receipt.Action)
		assert.Equal(t, "ENG-142", receipt.IssueIdentifier)
		assert.Equal(t, "ENG", receipt.TeamKey)
		assert.Equal(t, "acme", receipt.WorkspaceKey)
		assert.Equal(t, 1, receipt.SubscriptionCount)
		assert.NotContains(t, receipt.IssueIdentifier+receipt.IssueID+receipt.TeamKey, "secret-description")
		assert.NotContains(t, receipt.IssueIdentifier+receipt.IssueID, "ada@example.com")

		var events []models.CanvasEvent
		require.NoError(t, database.Conn().Where("workflow_id = ? AND node_id = ?", canvasID, nodeID).Find(&events).Error)
		require.Len(t, events, 1)
		envelope, ok := events[0].Data.Data().(map[string]any)
		require.True(t, ok)
		payload, ok := envelope["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, receipt.ID.String(), payload[linear.ReceiptField])
	})

	t.Run("records a rejected call", func(t *testing.T) {
		webhookID, _, _ := linearReceiptFixture(t, r, triggerName, "reject")
		response := postLinearWebhook(server, webhookID, body)
		require.Equal(t, http.StatusForbidden, response.Code)

		receipt := requireLinearReceipt(t, webhookID)
		assert.Equal(t, models.LinearWebhookOutcomeRejected, receipt.Outcome)
		assert.Equal(t, http.StatusForbidden, receipt.HTTPStatus)
	})

	t.Run("records an ignored call", func(t *testing.T) {
		webhookID, canvasID, nodeID := linearReceiptFixture(t, r, triggerName, "ignore")
		response := postLinearWebhook(server, webhookID, body)
		require.Equal(t, http.StatusOK, response.Code)

		receipt := requireLinearReceipt(t, webhookID)
		assert.Equal(t, models.LinearWebhookOutcomeIgnored, receipt.Outcome)
		support.VerifyCanvasNodeEventsCount(t, canvasID, nodeID, 0)
	})

	t.Run("leaves other integrations unchanged", func(t *testing.T) {
		integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "github", "github-receipt", nil)
		require.NoError(t, err)
		webhookID := uuid.New()
		require.NoError(t, database.Conn().Create(&models.Webhook{
			ID:                webhookID,
			State:             models.WebhookStateReady,
			Secret:            []byte("secret"),
			AppInstallationID: &integration.ID,
		}).Error)

		response := postLinearWebhook(server, webhookID, body)
		require.Equal(t, http.StatusNotFound, response.Code)
		assert.Empty(t, linearReceipts(t, webhookID))
	})
}

func linearReceiptFixture(t *testing.T, r *support.ResourceRegistry, triggerName, mode string) (uuid.UUID, uuid.UUID, string) {
	t.Helper()

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "linear", "linear-"+mode+"-"+uuid.NewString()[:8], nil)
	require.NoError(t, err)

	webhookID := uuid.New()
	require.NoError(t, database.Conn().Create(&models.Webhook{
		ID:                webhookID,
		State:             models.WebhookStateReady,
		Secret:            []byte("secret"),
		AppInstallationID: &integration.ID,
	}).Error)

	nodeID := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID:        nodeID,
				Name:          nodeID,
				Type:          models.NodeTypeTrigger,
				Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: triggerName}}),
				Configuration: datatypes.NewJSONType(map[string]any{"mode": mode}),
			},
		},
		nil,
	)
	require.NoError(t, database.Conn().
		Model(&models.CanvasNode{}).
		Where("workflow_id = ?", canvas.ID).
		Where("node_id = ?", nodeID).
		Update("webhook_id", webhookID).
		Error)

	return webhookID, canvas.ID, nodeID
}

func postLinearWebhook(server *Server, webhookID uuid.UUID, body []byte) *httptest.ResponseRecorder {
	return execRequest(server, requestParams{
		method: "POST",
		path:   "/webhooks/" + webhookID.String(),
		body:   body,
		headers: map[string]string{
			linear.EventHeader: linear.IssueResourceType,
		},
	})
}

func requireLinearReceipt(t *testing.T, webhookID uuid.UUID) models.LinearWebhookReceipt {
	t.Helper()
	receipts := linearReceipts(t, webhookID)
	require.Len(t, receipts, 1)
	return receipts[0]
}

func linearReceipts(t *testing.T, webhookID uuid.UUID) []models.LinearWebhookReceipt {
	t.Helper()
	var receipts []models.LinearWebhookReceipt
	require.NoError(t, database.Conn().Where("webhook_id = ?", webhookID).Find(&receipts).Error)
	return receipts
}
