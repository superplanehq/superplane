package public

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/encryption"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__HandleWebhook_FactoryWebhookAuthenticatesOnceForEveryListener(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	server, err := NewServer(
		r.Encryptor, r.Registry, jwt.NewSigner("test"), support.NewOIDCProvider(), "",
		"http://localhost", "http://localhost", "test", "/app/templates", r.AuthService, false,
	)
	require.NoError(t, err)

	db := database.Conn()
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	webhookID := uuid.New()
	secret, encrypted, err := crypto.NewRandomKey(t.Context(), encryption.ForDB(db, r.Encryptor), webhookID.String())
	require.NoError(t, err)
	webhook := models.NewFactoryWebhook(webhookID, r.Organization.ID, factory.ID, "Jira",
		models.WebhookAuthenticationConfig{Authentication: models.WebhookAuthenticationBearer}, encrypted)
	require.NoError(t, webhook.Create(db))

	post := func(token string, body string) int {
		return execRequest(server, requestParams{
			method: "POST", path: "/webhooks/" + webhookID.String(), body: []byte(body), authToken: token,
		}).Code
	}
	assert.Equal(t, http.StatusAccepted, post(secret, `{"issue":"ENG-1"}`), "a webhook with no automation accepts the request")
	assert.Equal(t, http.StatusUnauthorized, post("wrong", `{"issue":"ENG-1"}`))
	assert.Equal(t, http.StatusBadRequest, post(secret, `not json`))

	canvasIDs := make([]uuid.UUID, 0, 2)
	for range 2 {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
			NodeID: "webhook", Name: "Webhook", Type: models.NodeTypeTrigger,
			Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: factorycomponent.OnWebhookTriggerName}}),
			Configuration: datatypes.NewJSONType(map[string]any{"webhook": webhookID.String()}),
		}}, nil)
		require.NoError(t, db.Model(&models.CanvasNode{}).
			Where("workflow_id = ? AND node_id = ?", canvas.ID, "webhook").
			Update("webhook_id", webhookID).Error)
		canvasIDs = append(canvasIDs, canvas.ID)
	}

	assert.Equal(t, http.StatusUnauthorized, post("wrong", `{"issue":"ENG-1"}`))
	for _, canvasID := range canvasIDs {
		support.VerifyCanvasNodeEventsCount(t, canvasID, "webhook", 0)
	}

	require.Equal(t, http.StatusOK, post(secret, `{"issue":"ENG-1"}`))
	for _, canvasID := range canvasIDs {
		support.VerifyCanvasNodeEventsCount(t, canvasID, "webhook", 1)
		var event models.CanvasEvent
		require.NoError(t, db.Where("workflow_id = ? AND node_id = ?", canvasID, "webhook").First(&event).Error)
		payload := event.Data.Data().(map[string]any)["data"].(map[string]any)
		assert.Equal(t, map[string]any{"issue": "ENG-1"}, payload["body"])
		headers := payload["headers"].(map[string]any)
		assert.NotContains(t, headers["Authorization"], "Bearer "+secret, "the token must not be stored in the event")
	}
}
