package workers

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/test/support"
	supportcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__FactorySentryResolveConsumer(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	completedMessage := func(order *models.FactoryWorkOrder) messages.FactoryWorkOrderNotificationMessage {
		return messages.FactoryWorkOrderNotificationMessage{
			OrganizationID: r.Organization.ID.String(),
			FactoryID:      factoryModel.ID.String(),
			OrderID:        order.ID.String(),
			EventType:      factoryevents.EventTypeOrderStatusUpdated,
			FromState:      models.FactoryWorkOrderStateOpen,
			ToState:        models.FactoryWorkOrderStateClosed,
			Result:         models.FactoryWorkOrderResultCompleted,
		}
	}

	t.Run("resolves the originating sentry issue on completed", func(t *testing.T) {
		order, httpCtx := seedSentryWorkOrder(t, r, factoryModel, sentryIssuePayload("123"), sentryMockResponses(
			http.StatusOK, `{"id":"123","status":"unresolved"}`,
			http.StatusOK, `{"status":"resolved"}`,
			http.StatusOK, `{"id":"123","status":"resolved"}`,
		))

		require.NoError(t, newSentryResolveConsumer(r, httpCtx).process(db, completedMessage(order)))
		require.Len(t, httpCtx.Requests, 3)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.Path, "/issues/123/")
		assert.Equal(t, http.MethodPut, httpCtx.Requests[1].Method)
		assert.Contains(t, httpCtx.Requests[1].URL.RawQuery, "id=123")
		body, err := io.ReadAll(httpCtx.Requests[1].Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"status":"resolved"}`, string(body))
	})

	t.Run("skips an issue that is already resolved", func(t *testing.T) {
		order, httpCtx := seedSentryWorkOrder(t, r, factoryModel, sentryIssuePayload("99"), sentryMockResponses(
			http.StatusOK, `{"id":"99","status":"resolved"}`,
		))

		require.NoError(t, newSentryResolveConsumer(r, httpCtx).process(db, completedMessage(order)))
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
	})

	t.Run("retries a transient sentry error", func(t *testing.T) {
		order, httpCtx := seedSentryWorkOrder(t, r, factoryModel, sentryIssuePayload("55"), sentryMockResponses(
			http.StatusInternalServerError, `{"detail":"unavailable"}`,
		))

		err := newSentryResolveConsumer(r, httpCtx).process(db, completedMessage(order))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve Sentry issue 55")
	})

	t.Run("skips a permanent sentry error", func(t *testing.T) {
		order, httpCtx := seedSentryWorkOrder(t, r, factoryModel, sentryIssuePayload("4041"), sentryMockResponses(
			http.StatusNotFound, `{"detail":"not found"}`,
		))

		require.NoError(t, newSentryResolveConsumer(r, httpCtx).process(db, completedMessage(order)))
		require.Len(t, httpCtx.Requests, 1)
	})

	t.Run("skips rejected and failed closes", func(t *testing.T) {
		order, httpCtx := seedSentryWorkOrder(t, r, factoryModel, sentryIssuePayload("1"), nil)
		message := completedMessage(order)

		message.Result = models.FactoryWorkOrderResultRejected
		require.NoError(t, newSentryResolveConsumer(r, httpCtx).process(db, message))

		message.Result = models.FactoryWorkOrderResultFailed
		require.NoError(t, newSentryResolveConsumer(r, httpCtx).process(db, message))

		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("skips a github intake origin", func(t *testing.T) {
		order, httpCtx := seedSentryWorkOrder(t, r, factoryModel, map[string]any{
			"type": "github.issue",
			"data": map[string]any{
				"issue": map[string]any{
					"id":       "12",
					"html_url": "https://github.com/acme/app/issues/12",
				},
			},
		}, nil)

		require.NoError(t, newSentryResolveConsumer(r, httpCtx).process(db, completedMessage(order)))
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("skips a task without a source run", func(t *testing.T) {
		httpCtx := &supportcontexts.HTTPContext{}
		order, err := factoryModel.CreateWorkOrder(db, "Manual task", "", nil, nil, nil)
		require.NoError(t, err)
		completeWorkOrder(t, order)

		require.NoError(t, newSentryResolveConsumer(r, httpCtx).process(db, completedMessage(order)))
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("skips a missing work order", func(t *testing.T) {
		httpCtx := &supportcontexts.HTTPContext{}
		message := messages.FactoryWorkOrderNotificationMessage{
			OrganizationID: r.Organization.ID.String(),
			FactoryID:      factoryModel.ID.String(),
			OrderID:        uuid.NewString(),
			EventType:      factoryevents.EventTypeOrderStatusUpdated,
			ToState:        models.FactoryWorkOrderStateClosed,
			Result:         models.FactoryWorkOrderResultCompleted,
		}

		require.NoError(t, newSentryResolveConsumer(r, httpCtx).process(db, message))
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("skips when the source integration is gone", func(t *testing.T) {
		order, httpCtx := seedSentryWorkOrder(t, r, factoryModel, sentryIssuePayload("77"), nil)
		origin, err := resolveSentryWorkOrderOrigin(db, order)
		require.NoError(t, err)
		require.NotNil(t, origin)

		integration, err := models.FindUnscopedIntegrationInTransaction(db, origin.IntegrationID)
		require.NoError(t, err)
		require.NoError(t, db.Delete(integration).Error)

		require.NoError(t, newSentryResolveConsumer(r, httpCtx).process(db, completedMessage(order)))
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("consume rejects invalid json", func(t *testing.T) {
		err := newSentryResolveConsumer(r, &supportcontexts.HTTPContext{}).
			Consume(tackle.NewFakeDelivery([]byte("{")))
		require.Error(t, err)
	})
}

func Test__FactorySentryResolveConsumer_Consume(t *testing.T) {
	r := support.Setup(t)
	factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, httpCtx := seedSentryWorkOrder(t, r, factoryModel, sentryIssuePayload("321"), sentryMockResponses(
		http.StatusOK, `{"id":"321","status":"unresolved"}`,
		http.StatusOK, `{"status":"resolved"}`,
		http.StatusOK, `{"id":"321","status":"resolved"}`,
	))

	payload, err := json.Marshal(messages.FactoryWorkOrderNotificationMessage{
		OrganizationID: r.Organization.ID.String(),
		FactoryID:      factoryModel.ID.String(),
		OrderID:        order.ID.String(),
		EventType:      factoryevents.EventTypeOrderStatusUpdated,
		ToState:        models.FactoryWorkOrderStateClosed,
		Result:         models.FactoryWorkOrderResultCompleted,
	})
	require.NoError(t, err)
	require.NoError(t, newSentryResolveConsumer(r, httpCtx).Consume(tackle.NewFakeDelivery(payload)))
	require.Len(t, httpCtx.Requests, 3)
}

func newSentryResolveConsumer(r *support.ResourceRegistry, httpCtx *supportcontexts.HTTPContext) *FactorySentryResolveConsumer {
	consumer := NewFactorySentryResolveConsumer("amqp://localhost:5672", r.Encryptor, r.Registry)
	consumer.httpContext = httpCtx
	return consumer
}

func seedSentryWorkOrder(
	t *testing.T,
	r *support.ResourceRegistry,
	factoryModel *models.Factory,
	eventPayload map[string]any,
	responses []*http.Response,
) (*models.FactoryWorkOrder, *supportcontexts.HTTPContext) {
	t.Helper()

	db := database.Conn()
	integrationID := uuid.New()
	userToken, err := r.Encryptor.Encrypt(t.Context(), []byte("user-token"), []byte(integrationID.String()))
	require.NoError(t, err)

	integration, err := models.CreateIntegration(
		integrationID,
		r.Organization.ID,
		sentryAppName,
		support.RandomName("sentry"),
		map[string]any{
			"baseUrl":   "https://sentry.io",
			"userToken": base64.StdEncoding.EncodeToString(userToken),
		},
	)
	require.NoError(t, err)
	integration.State = models.IntegrationStateReady
	integration.Metadata = datatypes.NewJSONType(map[string]any{
		"organization": map[string]any{"slug": "example", "name": "Example", "id": "1"},
	})
	require.NoError(t, db.Save(integration).Error)

	const triggerNodeID = "trigger"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: triggerNodeID, Type: models.NodeTypeTrigger}},
		nil,
	)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryModel.ID).Error)
	require.NoError(t, db.Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvas.ID, triggerNodeID).
		Update("app_installation_id", integration.ID).Error)

	triggerEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNodeID, "default", nil, eventPayload)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, triggerEvent)
	require.NoError(t, err)

	nodeExecution := support.CreateCanvasNodeExecution(t, canvas.ID, triggerNodeID, triggerEvent.ID, triggerEvent.ID)
	nodeExecution.RunID = run.ID
	require.NoError(t, db.Save(nodeExecution).Error)

	order, err := factoryModel.CreateWorkOrder(db, "Fix sentry issue", "", nil, nil, &run.ID)
	require.NoError(t, err)
	completeWorkOrder(t, order)

	httpCtx := &supportcontexts.HTTPContext{Responses: responses}
	return order, httpCtx
}

func completeWorkOrder(t *testing.T, order *models.FactoryWorkOrder) {
	t.Helper()
	db := database.Conn()
	_, err := order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
	})
	require.NoError(t, err)
	_, err = order.Close(db, models.FactoryWorkOrderResultCompleted, nil)
	require.NoError(t, err)
}

func sentryIssuePayload(issueID string) map[string]any {
	return map[string]any{
		"type": sentry.IssuePayloadType,
		"data": map[string]any{
			"resource": "issue",
			"action":   "created",
			"data": map[string]any{
				"issue": map[string]any{
					"id":        issueID,
					"title":     "boom",
					"permalink": "https://sentry.io/issues/" + issueID + "/",
				},
			},
		},
	}
}

func sentryMockResponses(pairs ...any) []*http.Response {
	responses := make([]*http.Response, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		status, _ := pairs[i].(int)
		body, _ := pairs[i+1].(string)
		responses = append(responses, &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		})
	}
	return responses
}
