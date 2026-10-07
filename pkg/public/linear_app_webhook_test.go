package public

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__HandleLinearAppWebhook__DeliversIssueToTheTeamTrigger(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	const appSecret = "linear-app-webhook-secret"
	t.Setenv(config.EnvLinearOAuthClientID, "hosted-client")
	t.Setenv(config.EnvLinearOAuthClientSecret, "hosted-secret")
	t.Setenv(config.EnvLinearOAuthWebhookSecret, appSecret)

	r.Registry.Triggers["linear.onIssue"] = &linear.OnIssue{}

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

	integration := readyLinearIntegration(t, r, "org-1", "acme")
	webhookID := readyLinearAppWebhook(t, r, integration, "team-1")

	body := []byte(`{
		"action": "create",
		"type": "Issue",
		"organizationId": "org-1",
		"url": "https://linear.app/acme/issue/ENG-1",
		"data": {
			"id": "issue-1",
			"identifier": "ENG-1",
			"title": "Deploy failed",
			"teamId": "team-1",
			"team": {"id": "team-1", "key": "ENG"}
		}
	}`)

	response := postLinearAppWebhook(server, body, linearSignature(body, appSecret))
	require.Equal(t, http.StatusOK, response.Code)

	var events []models.CanvasEvent
	require.NoError(t, database.Conn().Where("node_id = ?", "trigger-1").Find(&events).Error)
	require.Len(t, events, 1)
	assert.Equal(t, "default", events[0].Channel)

	var receipts []models.LinearWebhookReceipt
	require.NoError(t, database.Conn().Where("webhook_id = ?", webhookID).Find(&receipts).Error)
	require.Len(t, receipts, 1)
	assert.Equal(t, models.LinearWebhookOutcomeAccepted, receipts[0].Outcome)

	otherTeam := []byte(`{
		"action": "create",
		"type": "Issue",
		"organizationId": "org-1",
		"url": "https://linear.app/acme/issue/OPS-1",
		"data": {"id": "issue-2", "teamId": "team-2", "team": {"id": "team-2", "key": "OPS"}}
	}`)
	ignored := postLinearAppWebhook(server, otherTeam, linearSignature(otherTeam, appSecret))
	require.Equal(t, http.StatusOK, ignored.Code)
	require.NoError(t, database.Conn().Where("node_id = ?", "trigger-1").Find(&events).Error)
	assert.Len(t, events, 1)

	rejected := postLinearAppWebhook(server, body, linearSignature(body, "wrong-secret"))
	require.Equal(t, http.StatusForbidden, rejected.Code)
}

func readyLinearIntegration(t *testing.T, r *support.ResourceRegistry, organizationID, urlKey string) *models.Integration {
	t.Helper()

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "linear", "linear-"+uuid.NewString()[:8], nil)
	require.NoError(t, err)
	require.NoError(t, database.Conn().Model(integration).Updates(map[string]any{
		"state": models.IntegrationStateReady,
		"metadata": datatypes.NewJSONType(map[string]any{
			"organizationId": organizationID,
			"urlKey":         urlKey,
			"hostedOAuth":    true,
			"teams":          []map[string]any{{"id": "team-1", "key": "ENG", "name": "Engineering"}},
		}),
	}).Error)
	return integration
}

func readyLinearAppWebhook(t *testing.T, r *support.ResourceRegistry, integration *models.Integration, teamID string) uuid.UUID {
	t.Helper()

	webhookID := uuid.New()
	require.NoError(t, database.Conn().Create(&models.Webhook{
		ID:                webhookID,
		State:             models.WebhookStateReady,
		Secret:            []byte("not-the-app-secret"),
		AppInstallationID: &integration.ID,
		Configuration: datatypes.NewJSONType[any](map[string]any{
			"teamId":       teamID,
			"resourceType": linear.IssueResourceType,
		}),
		Metadata: datatypes.NewJSONType[any](map[string]any{"appLevel": true}),
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
				Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "linear.onIssue"}}),
				Configuration: datatypes.NewJSONType(map[string]any{"team": teamID, "actions": []string{"create"}}),
			},
		},
		nil,
	)
	require.NoError(t, database.Conn().
		Model(&models.CanvasNode{}).
		Where("workflow_id = ?", canvas.ID).
		Where("node_id = ?", nodeID).
		Updates(map[string]any{
			"webhook_id":          webhookID,
			"app_installation_id": integration.ID,
		}).
		Error)
	return webhookID
}

func postLinearAppWebhook(server *Server, body []byte, signature string) *httptest.ResponseRecorder {
	return execRequest(server, requestParams{
		method: "POST",
		path:   "/linear/webhook",
		body:   body,
		headers: map[string]string{
			linear.EventHeader:     linear.IssueResourceType,
			linear.SignatureHeader: signature,
		},
	})
}

func linearSignature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
