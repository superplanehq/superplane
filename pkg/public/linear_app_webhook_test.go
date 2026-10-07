package public

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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
	server := linearAppWebhookServer(t, r)

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

func Test__HandleLinearAppWebhook__RoutesAttachmentByParentIssue(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	const appSecret = "linear-app-webhook-secret"
	t.Setenv(config.EnvLinearOAuthClientID, "hosted-client")
	t.Setenv(config.EnvLinearOAuthClientSecret, "hosted-secret")
	t.Setenv(config.EnvLinearOAuthWebhookSecret, appSecret)

	r.Registry.Triggers["linear.onIssueAttachment"] = &linear.OnIssueAttachment{}
	server := linearAppWebhookServer(t, r)

	body := linearAttachmentWebhookBody(t)
	var payload struct {
		OrganizationID string `json:"organizationId"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))

	integration := readyLinearIntegration(t, r, payload.OrganizationID, "acme")
	readyLinearSubscription(t, r, integration, linearSubscription{
		nodeID:       "trigger-attachment",
		teamID:       "team-1",
		resourceType: linear.AttachmentResourceType,
		triggerName:  "linear.onIssueAttachment",
		configuration: map[string]any{
			"team":    "team-1",
			"actions": []string{"create"},
		},
	})

	response := postLinearAppEvent(server, linear.AttachmentResourceType, body, linearSignature(body, appSecret))
	require.Equal(t, http.StatusOK, response.Code)

	var events []models.CanvasEvent
	require.NoError(t, database.Conn().Where("node_id = ?", "trigger-attachment").Find(&events).Error)
	require.Len(t, events, 1)
}

func Test__HandleLinearAppWebhook__IgnoresAnAttachmentWithNoAttachmentTrigger(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	const appSecret = "linear-app-webhook-secret"
	t.Setenv(config.EnvLinearOAuthClientID, "hosted-client")
	t.Setenv(config.EnvLinearOAuthClientSecret, "hosted-secret")
	t.Setenv(config.EnvLinearOAuthWebhookSecret, appSecret)

	r.Registry.Triggers["linear.onIssue"] = &linear.OnIssue{}
	server := linearAppWebhookServer(t, r)

	integration := readyLinearIntegration(t, r, "org-1", "acme")
	readyLinearAppWebhook(t, r, integration, "team-1")

	body := []byte(`{
		"action": "create",
		"type": "Attachment",
		"organizationId": "org-1",
		"url": "https://linear.app/acme/issue/ENG-1",
		"data": {"id": "attachment-1", "issueId": "issue-1"}
	}`)

	response := postLinearAppEvent(server, linear.AttachmentResourceType, body, linearSignature(body, appSecret))
	require.Equal(t, http.StatusOK, response.Code)

	var events []models.CanvasEvent
	require.NoError(t, database.Conn().Where("node_id = ?", "trigger-1").Find(&events).Error)
	assert.Empty(t, events)
}

func Test__HandleLinearAppWebhook__DoesNotRepeatASuccessfulSubscription(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	const appSecret = "linear-app-webhook-secret"
	t.Setenv(config.EnvLinearOAuthClientID, "hosted-client")
	t.Setenv(config.EnvLinearOAuthClientSecret, "hosted-secret")
	t.Setenv(config.EnvLinearOAuthWebhookSecret, appSecret)

	r.Registry.Triggers["linear.onIssue"] = &linear.OnIssue{}
	server := linearAppWebhookServer(t, r)

	integration := readyLinearIntegration(t, r, "org-1", "acme")
	goodWebhook := readyLinearAppWebhook(t, r, integration, "team-1")
	readyLinearSubscription(t, r, integration, linearSubscription{
		nodeID:       "trigger-bad",
		teamID:       "team-1",
		resourceType: linear.IssueResourceType,
		triggerName:  "linear.missing",
		configuration: map[string]any{
			"team":    "team-1",
			"actions": []string{"create"},
		},
	})

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

	first := postLinearAppWebhook(server, body, linearSignature(body, appSecret))
	require.Equal(t, http.StatusInternalServerError, first.Code)

	var events []models.CanvasEvent
	require.NoError(t, database.Conn().Where("node_id = ?", "trigger-1").Find(&events).Error)
	require.Len(t, events, 1)

	retry := postLinearAppWebhook(server, body, linearSignature(body, appSecret))
	require.Equal(t, http.StatusInternalServerError, retry.Code)
	require.NoError(t, database.Conn().Where("node_id = ?", "trigger-1").Find(&events).Error)
	assert.Len(t, events, 1)

	var receipts []models.LinearWebhookReceipt
	require.NoError(t, database.Conn().Where("webhook_id = ?", goodWebhook).Find(&receipts).Error)
	require.Len(t, receipts, 1)
	assert.Equal(t, models.LinearWebhookOutcomeAccepted, receipts[0].Outcome)
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

type linearSubscription struct {
	nodeID        string
	teamID        string
	resourceType  string
	triggerName   string
	configuration map[string]any
}

func readyLinearAppWebhook(t *testing.T, r *support.ResourceRegistry, integration *models.Integration, teamID string) uuid.UUID {
	t.Helper()
	return readyLinearSubscription(t, r, integration, linearSubscription{
		nodeID:        "trigger-1",
		teamID:        teamID,
		resourceType:  linear.IssueResourceType,
		triggerName:   "linear.onIssue",
		configuration: map[string]any{"team": teamID, "actions": []string{"create"}},
	})
}

func readyLinearSubscription(t *testing.T, r *support.ResourceRegistry, integration *models.Integration, spec linearSubscription) uuid.UUID {
	t.Helper()

	webhookID := uuid.New()
	require.NoError(t, database.Conn().Create(&models.Webhook{
		ID:                webhookID,
		State:             models.WebhookStateReady,
		Secret:            []byte("not-the-app-secret"),
		AppInstallationID: &integration.ID,
		Configuration: datatypes.NewJSONType[any](map[string]any{
			"teamId":       spec.teamID,
			"resourceType": spec.resourceType,
		}),
		Metadata: datatypes.NewJSONType[any](map[string]any{"appLevel": true}),
	}).Error)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID:        spec.nodeID,
				Name:          spec.nodeID,
				Type:          models.NodeTypeTrigger,
				Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: spec.triggerName}}),
				Configuration: datatypes.NewJSONType(spec.configuration),
			},
		},
		nil,
	)
	require.NoError(t, database.Conn().
		Model(&models.CanvasNode{}).
		Where("workflow_id = ?", canvas.ID).
		Where("node_id = ?", spec.nodeID).
		Updates(map[string]any{
			"webhook_id":          webhookID,
			"app_installation_id": integration.ID,
		}).
		Error)
	return webhookID
}

func linearAppWebhookServer(t *testing.T, r *support.ResourceRegistry) *Server {
	t.Helper()
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
	return server
}

func linearAttachmentWebhookBody(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("../integrations/linear/example_data_on_issue_attachment.json")
	require.NoError(t, err)

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &envelope))
	require.NotEmpty(t, envelope.Data)
	return envelope.Data
}

func postLinearAppWebhook(server *Server, body []byte, signature string) *httptest.ResponseRecorder {
	return postLinearAppEvent(server, linear.IssueResourceType, body, signature)
}

func postLinearAppEvent(server *Server, eventType string, body []byte, signature string) *httptest.ResponseRecorder {
	return execRequest(server, requestParams{
		method: "POST",
		path:   "/linear/webhook",
		body:   body,
		headers: map[string]string{
			linear.EventHeader:     eventType,
			linear.SignatureHeader: signature,
		},
	})
}

func linearSignature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
