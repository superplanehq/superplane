package public

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	_ "github.com/superplanehq/superplane/pkg/integrations/productive"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

func Test__HandleWebhook_ProductiveFailureLogsJSON(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	server := newWebhookTestServer(t, r)
	webhookID := uuid.New()
	require.NoError(t, database.Conn().Create(&models.Webhook{
		ID:     webhookID,
		State:  models.WebhookStateReady,
		Secret: []byte("do-not-log-secret"),
	}).Error)

	integration := support.CreateIntegrationWithCapabilities(t, r.Organization.ID, nil)
	nodeID := "on-task"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID:        nodeID,
				Name:          nodeID,
				Type:          models.NodeTypeTrigger,
				Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: productiveOnTaskTrigger}}),
				Configuration: datatypes.NewJSONType(map[string]any{}),
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

	logs := captureProductiveWebhookLogs(t)
	body := []byte(`{"raw":"do-not-log-body"}`)
	for _, event := range []string{"task.updated", "task.created"} {
		response := execRequest(server, requestParams{
			method: "POST",
			path:   "/webhooks/" + webhookID.String() + "/" + event,
			body:   body,
		})
		require.Equal(t, http.StatusInternalServerError, response.Code)
	}

	response := execRequest(server, requestParams{
		method: "POST",
		path:   "/webhooks/" + webhookID.String(),
		body:   body,
	})
	require.Equal(t, http.StatusInternalServerError, response.Code)

	payloads := decodeProductiveWebhookLogs(t, logs)
	require.Len(t, payloads, 3)
	assert.Equal(t, "task.updated", payloads[0]["webhookType"])
	assert.Equal(t, "task.created", payloads[1]["webhookType"])
	assert.Equal(t, "unknown", payloads[2]["webhookType"])
	for _, payload := range payloads {
		assert.Equal(t, logging.ComponentWebhookProductive, payload["component"])
		assert.Equal(t, "error", payload["level"])
		assert.Equal(t, "error handling webhook", payload["msg"])
		assert.Equal(t, r.Organization.ID.String(), payload["organization_id"])
		assert.Equal(t, canvas.ID.String(), payload["workflow_id"])
		assert.Equal(t, nodeID, payload["node_id"])
		assert.Equal(t, webhookID.String(), payload["webhook_id"])
		assert.Equal(t, models.WebhookStateReady, payload["webhook_state"])
		assert.Equal(t, integration.ID.String(), payload["app_installation_id"])
		assert.EqualValues(t, http.StatusInternalServerError, payload["status"])
		assert.Contains(t, payload["error"], "project is required")
	}
	assert.NotContains(t, logs.String(), "do-not-log-secret")
	assert.NotContains(t, logs.String(), "do-not-log-body")
}

func Test__HandleWebhook_OtherFailureKeepsTextLog(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	const triggerName = "dummy-webhook-failure"
	r.Registry.Triggers[triggerName] = impl.NewDummyTrigger(impl.DummyTriggerOptions{
		Name: triggerName,
		HandleWebhookFunc: func(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
			return http.StatusBadGateway, nil, fmt.Errorf("dummy webhook failed")
		},
	})

	server := newWebhookTestServer(t, r)
	webhookID := uuid.New()
	require.NoError(t, database.Conn().Create(&models.Webhook{
		ID:     webhookID,
		State:  models.WebhookStateReady,
		Secret: []byte("secret"),
	}).Error)

	nodeID := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: nodeID,
				Name:   nodeID,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: triggerName}}),
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

	logs := captureProductiveWebhookLogs(t)
	hook := logtest.NewGlobal()
	t.Cleanup(func() {
		log.StandardLogger().ReplaceHooks(make(log.LevelHooks))
	})

	response := execRequest(server, requestParams{
		method: "POST",
		path:   "/webhooks/" + webhookID.String(),
		body:   []byte(`{"ok":true}`),
	})
	require.Equal(t, http.StatusBadGateway, response.Code)
	assert.Empty(t, strings.TrimSpace(logs.String()))

	var failure *log.Entry
	for _, entry := range hook.AllEntries() {
		if entry.Level == log.ErrorLevel && strings.Contains(entry.Message, "error handling webhook") {
			failure = entry
		}
	}
	require.NotNil(t, failure)
	assert.Contains(t, failure.Message, "dummy webhook failed")
	assert.Equal(t, webhookID.String(), failure.Data["webhook_id"])
	_, hasComponent := failure.Data["component"]
	assert.False(t, hasComponent)
}

func newWebhookTestServer(t *testing.T, r *support.ResourceRegistry) *Server {
	t.Helper()

	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		jwt.NewSigner("test"),
		support.NewOIDCProvider(),
		"",
		"http://localhost",
		"http://localhost",
		"test",
		"/app/templates",
		r.AuthService,
		false,
	)
	require.NoError(t, err)
	return server
}

func captureProductiveWebhookLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	logger := logging.ProductiveWebhookLogger()
	previous := logger.Out
	buffer := &bytes.Buffer{}
	logger.SetOutput(buffer)
	t.Cleanup(func() {
		logger.SetOutput(previous)
	})
	return buffer
}

func decodeProductiveWebhookLogs(t *testing.T, buffer *bytes.Buffer) []map[string]any {
	t.Helper()

	payloads := []map[string]any{}
	for _, line := range strings.Split(buffer.String(), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		payload := map[string]any{}
		require.NoError(t, json.Unmarshal([]byte(line), &payload))
		payloads = append(payloads, payload)
	}
	return payloads
}
