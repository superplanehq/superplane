package public

import (
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
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

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
		nil,
	)
	require.NoError(t, err)
	return server
}
