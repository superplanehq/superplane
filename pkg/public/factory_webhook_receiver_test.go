package public

import (
	"fmt"
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

func Test__HandleWebhook_TaskMatchRunsOncePerMatchedTask(t *testing.T) {
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
	webhook := models.NewFactoryWebhook(webhookID, r.Organization.ID, factory.ID, "Deploy",
		models.WebhookAuthenticationConfig{Authentication: models.WebhookAuthenticationBearer}, encrypted)
	require.NoError(t, webhook.Create(db))

	draft, err := factory.CreateWorkOrder(db, "Draft task", "", &r.User, nil, nil)
	require.NoError(t, err)
	open, err := factory.CreateWorkOrder(db, "Open task", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = open.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)
	secondDraft, err := factory.CreateWorkOrder(db, "Second draft", "", &r.User, nil, nil)
	require.NoError(t, err)
	sha := "abc123def456"
	for index, order := range []*models.FactoryWorkOrder{draft, secondDraft} {
		pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL: fmt.Sprintf("https://github.com/acme/app/pull/%d", 40+index),
		})
		require.NoError(t, err, "pull request %d", index)
		require.NoError(t, pullRequest.SetMergeability(db, models.FactoryPullRequestMergeabilitySnapshot{HeadSHA: sha, Mergeable: true}))
	}

	listen := func(match map[string]any) uuid.UUID {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
			NodeID: "webhook", Name: "Webhook", Type: models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: factorycomponent.OnWebhookTriggerName}}),
			Configuration: datatypes.NewJSONType(map[string]any{
				"webhook": webhookID.String(), "taskMatch": match,
			}),
		}}, nil)
		require.NoError(t, db.Model(&models.Canvas{}).Where("id = ?", canvas.ID).Updates(map[string]any{
			"factory_id": factory.ID, "column_key": models.CanvasColumnKeyBacklog,
		}).Error)
		require.NoError(t, db.Model(&models.CanvasNode{}).
			Where("workflow_id = ? AND node_id = ?", canvas.ID, "webhook").
			Update("webhook_id", webhookID).Error)
		return canvas.ID
	}
	commitCanvas := listen(map[string]any{"kind": "commit", "fieldPath": "commit.sha"})
	taskCanvas := listen(map[string]any{"kind": "task", "fieldPath": "task.number"})
	pullCanvas := listen(map[string]any{"kind": "pull_request", "fieldPath": "pull.url"})
	post := func(body string) int {
		return execRequest(server, requestParams{
			method: "POST", path: "/webhooks/" + webhookID.String(), body: []byte(body), authToken: secret,
		}).Code
	}
	workOrderIDs := func(canvasID uuid.UUID) []string {
		var events []models.CanvasEvent
		require.NoError(t, db.Where("workflow_id = ? AND node_id = ?", canvasID, "webhook").Order("created_at, id").Find(&events).Error)
		ids := make([]string, 0, len(events))
		for _, event := range events {
			payload := event.Data.Data().(map[string]any)["data"].(map[string]any)
			order, _ := payload["workOrder"].(map[string]any)
			id, _ := order["id"].(string)
			ids = append(ids, id)
		}
		return ids
	}

	require.Equal(t, http.StatusOK, post(`{"commit":{"sha":"missing"},"pull":{"url":"notaurl"}}`))
	assert.Empty(t, workOrderIDs(commitCanvas))
	assert.Empty(t, workOrderIDs(pullCanvas))

	body := `{"commit":{"sha":"` + sha + `"}}`
	require.Equal(t, http.StatusOK, post(body))
	assert.ElementsMatch(t, []string{draft.ID.String(), secondDraft.ID.String()}, workOrderIDs(commitCanvas))
	require.Equal(t, http.StatusOK, post(body))
	assert.ElementsMatch(t, []string{draft.ID.String(), secondDraft.ID.String()}, workOrderIDs(commitCanvas))

	require.Equal(t, http.StatusOK, post(`{"task":{"number":`+fmt.Sprint(draft.Number)+`}}`))
	assert.Equal(t, []string{draft.ID.String()}, workOrderIDs(taskCanvas))
	require.Equal(t, http.StatusOK, post(`{"task":{"number":`+fmt.Sprint(open.Number)+`}}`))
	assert.Equal(t, []string{draft.ID.String()}, workOrderIDs(taskCanvas), "an Implement task does not match a Backlog automation")
}
