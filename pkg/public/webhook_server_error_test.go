package public

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

const (
	webhookSentrySecret = "webhook-secret-must-stay-out-of-sentry"
	webhookSentryBody   = `{"payload":"webhook-body-must-stay-out-of-sentry"}`
	webhookSentryCause  = "node rejected the delivery"
)

func Test__HandleWebhook_ServerErrorCapturesCause(t *testing.T) {
	testCases := []struct {
		name       string
		nodeType   string
		status     int
		deliveryID string
	}{
		{
			name:       "trigger returns 500",
			nodeType:   models.NodeTypeTrigger,
			status:     http.StatusInternalServerError,
			deliveryID: "delivery-500",
		},
		{
			name:     "action returns 502",
			nodeType: models.NodeTypeComponent,
			status:   http.StatusBadGateway,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			transport := bindTestSentryHub(t)
			response, ids := postFailingWebhook(t, testCase.nodeType, testCase.status, testCase.deliveryID)

			require.Equal(t, testCase.status, response.Code)
			assert.Contains(t, response.Body.String(), webhookSentryCause)

			events := transport.Events()
			require.Len(t, events, 1)
			event := events[0]
			assert.Empty(t, event.Message)
			assert.Contains(t, capturedExceptionText(event), webhookSentryCause)
			assert.Equal(t, ids.webhookID, event.Tags["webhook_id"])
			assert.Equal(t, ids.nodeID, event.Tags["node_id"])
			assert.Equal(t, ids.canvasID, event.Tags["canvas_id"])
			assert.Equal(t, ids.organizationID, event.Tags["organization_id"])
			assert.Equal(t, strconv.Itoa(testCase.status), event.Tags["status"])
			if testCase.deliveryID == "" {
				_, hasDelivery := event.Tags["github_delivery_id"]
				assert.False(t, hasDelivery)
			} else {
				assert.Equal(t, testCase.deliveryID, event.Tags["github_delivery_id"])
			}

			encoded, err := json.Marshal(event)
			require.NoError(t, err)
			assert.NotContains(t, string(encoded), webhookSentrySecret)
			assertEventOmitsWebhookBody(t, event, encoded)
		})
	}
}

func assertEventOmitsWebhookBody(t *testing.T, event *sentry.Event, encoded []byte) {
	t.Helper()

	if event.Request != nil {
		assert.NotContains(t, event.Request.Data, webhookSentryBody)
	}

	encodedBody, err := json.Marshal(webhookSentryBody)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), webhookSentryBody)
	assert.NotContains(t, string(encoded), string(encodedBody))
}

func Test__HandleWebhook_ClientErrorIsNotSentToSentry(t *testing.T) {
	transport := bindTestSentryHub(t)
	response, _ := postFailingWebhook(t, models.NodeTypeTrigger, http.StatusBadRequest, "delivery-400")

	require.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), webhookSentryCause)
	assert.Empty(t, transport.Events())
}

type webhookFailureIDs struct {
	webhookID      string
	nodeID         string
	canvasID       string
	organizationID string
}

func postFailingWebhook(t *testing.T, nodeType string, status int, deliveryID string) (*httptest.ResponseRecorder, webhookFailureIDs) {
	t.Helper()

	resources := support.Setup(t)
	t.Cleanup(resources.Close)

	componentName := "dummy-webhook-sentry-" + nodeType
	registerFailingWebhookNode(resources, componentName, nodeType, status)

	server := newWebhookTestServer(t, resources)
	webhookID := uuid.New()
	require.NoError(t, database.Conn().Create(&models.Webhook{
		ID:     webhookID,
		State:  models.WebhookStateReady,
		Secret: []byte(webhookSentrySecret),
	}).Error)

	nodeID := "node-1"
	node := models.CanvasNode{
		NodeID: nodeID,
		Name:   nodeID,
		Type:   nodeType,
	}
	if nodeType == models.NodeTypeTrigger {
		node.Ref = datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: componentName}})
	} else {
		node.Ref = datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: componentName}})
	}

	canvas, _ := support.CreateCanvas(t, resources.Organization.ID, resources.User, []models.CanvasNode{node}, nil)
	require.NoError(t, database.Conn().
		Model(&models.CanvasNode{}).
		Where("workflow_id = ?", canvas.ID).
		Where("node_id = ?", nodeID).
		Update("webhook_id", webhookID).
		Error)

	headers := map[string]string{
		"X-Hub-Signature-256": "sha256=not-the-secret",
	}
	if deliveryID != "" {
		headers["X-GitHub-Delivery"] = deliveryID
	}

	response := execRequest(server, requestParams{
		method:  "POST",
		path:    "/webhooks/" + webhookID.String(),
		body:    []byte(webhookSentryBody),
		headers: headers,
	})

	return response, webhookFailureIDs{
		webhookID:      webhookID.String(),
		nodeID:         nodeID,
		canvasID:       canvas.ID.String(),
		organizationID: resources.Organization.ID.String(),
	}
}

func registerFailingWebhookNode(resources *support.ResourceRegistry, name string, nodeType string, status int) {
	handle := func(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
		return status, nil, errors.New(webhookSentryCause)
	}
	if nodeType == models.NodeTypeTrigger {
		resources.Registry.Triggers[name] = impl.NewDummyTrigger(impl.DummyTriggerOptions{
			Name:              name,
			HandleWebhookFunc: handle,
		})
		return
	}
	resources.Registry.Actions[name] = impl.NewDummyAction(impl.DummyActionOptions{
		Name:              name,
		HandleWebhookFunc: handle,
	})
}
