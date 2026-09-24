package workers

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

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

	t.Run("retries a sentry request timeout", func(t *testing.T) {
		order, httpCtx := seedSentryWorkOrder(t, r, factoryModel, sentryIssuePayload("4081"), sentryMockResponses(
			http.StatusRequestTimeout, `{"detail":"timeout"}`,
		))

		err := newSentryResolveConsumer(r, httpCtx).process(db, completedMessage(order))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve Sentry issue 4081")
	})

	t.Run("uses the source run version integration after the live trigger changes", func(t *testing.T) {
		order, httpCtx := seedSentryWorkOrder(t, r, factoryModel, sentryIssuePayload("222"), sentryMockResponses(
			http.StatusOK, `{"id":"222","status":"unresolved"}`,
			http.StatusOK, `{"status":"resolved"}`,
			http.StatusOK, `{"id":"222","status":"resolved"}`,
		))

		run, err := models.FindUnscopedCanvasRun(db, *order.SourceRunID)
		require.NoError(t, err)
		retargetLiveTriggerInstallation(t, r, run.WorkflowID, run.NodeID)

		require.NoError(t, newSentryResolveConsumer(r, httpCtx).process(db, completedMessage(order)))
		require.Len(t, httpCtx.Requests, 3)
		assert.Contains(t, httpCtx.Requests[0].URL.Path, "/issues/222/")
	})

	t.Run("retries a transient sentry client construction error", func(t *testing.T) {
		order, httpCtx := seedHostedSentryWorkOrder(t, r, factoryModel, sentryIssuePayload("81"), sentryMockResponses(
			http.StatusInternalServerError, `{"detail":"unavailable"}`,
		))

		err := newSentryResolveConsumer(r, httpCtx).process(db, completedMessage(order))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve Sentry issue 81")
		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, httpCtx.Requests[0].URL.Path, "/authorizations/")
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

	return seedSentryWorkOrderWithIntegration(t, r, factoryModel, createReadySentryIntegration(t, r), eventPayload, responses)
}

func seedHostedSentryWorkOrder(
	t *testing.T,
	r *support.ResourceRegistry,
	factoryModel *models.Factory,
	eventPayload map[string]any,
	responses []*http.Response,
) (*models.FactoryWorkOrder, *supportcontexts.HTTPContext) {
	t.Helper()

	t.Setenv("SUPERPLANE_SENTRY_APP_SLUG", "superplane")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_ID", "cid")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_SECRET", "csecret")

	return seedSentryWorkOrderWithIntegration(
		t,
		r,
		factoryModel,
		createReadyHostedSentryIntegration(t, r),
		eventPayload,
		responses,
	)
}

func seedSentryWorkOrderWithIntegration(
	t *testing.T,
	r *support.ResourceRegistry,
	factoryModel *models.Factory,
	integration *models.Integration,
	eventPayload map[string]any,
	responses []*http.Response,
) (*models.FactoryWorkOrder, *supportcontexts.HTTPContext) {
	t.Helper()

	db := database.Conn()
	const triggerNodeID = "trigger"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: triggerNodeID, Type: models.NodeTypeTrigger}},
		nil,
	)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryModel.ID).Error)
	bindCanvasNodeInstallation(t, canvas.ID, triggerNodeID, integration.ID)

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

func createReadySentryIntegration(t *testing.T, r *support.ResourceRegistry) *models.Integration {
	t.Helper()

	integrationID := uuid.New()
	userToken, err := r.Encryptor.Encrypt(t.Context(), []byte("user-token"), []byte(integrationID.String()))
	require.NoError(t, err)

	return saveReadySentryIntegration(t, r, integrationID, map[string]any{
		"baseUrl":   "https://sentry.io",
		"userToken": base64.StdEncoding.EncodeToString(userToken),
	}, map[string]any{
		"organization": map[string]any{"slug": "example", "name": "Example", "id": "1"},
	})
}

func createReadyHostedSentryIntegration(t *testing.T, r *support.ResourceRegistry) *models.Integration {
	t.Helper()

	integration := saveReadySentryIntegration(t, r, uuid.New(), map[string]any{}, map[string]any{
		"hostedApp":        true,
		"installationUUID": "install-uuid",
		"tokenExpiresAt":   time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		"organization":     map[string]any{"slug": "example", "name": "Example", "id": "1"},
	})
	saveEncryptedIntegrationSecret(t, r, integration, sentry.SecretAccessToken, "expired-token")
	saveEncryptedIntegrationSecret(t, r, integration, sentry.SecretRefreshToken, "refresh-token")
	return integration
}

func saveReadySentryIntegration(
	t *testing.T,
	r *support.ResourceRegistry,
	integrationID uuid.UUID,
	config map[string]any,
	metadata map[string]any,
) *models.Integration {
	t.Helper()

	integration, err := models.CreateIntegration(
		integrationID,
		r.Organization.ID,
		sentryAppName,
		support.RandomName("sentry"),
		config,
	)
	require.NoError(t, err)
	integration.State = models.IntegrationStateReady
	integration.Metadata = datatypes.NewJSONType(metadata)
	require.NoError(t, database.Conn().Save(integration).Error)
	return integration
}

func saveEncryptedIntegrationSecret(
	t *testing.T,
	r *support.ResourceRegistry,
	integration *models.Integration,
	name, value string,
) {
	t.Helper()

	encrypted, err := r.Encryptor.Encrypt(t.Context(), []byte(value), []byte(integration.ID.String()))
	require.NoError(t, err)

	now := time.Now()
	require.NoError(t, database.Conn().Create(&models.IntegrationSecret{
		OrganizationID: integration.OrganizationID,
		InstallationID: integration.ID,
		Name:           name,
		Value:          encrypted,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}).Error)
}

func bindCanvasNodeInstallation(t *testing.T, canvasID uuid.UUID, nodeID string, integrationID uuid.UUID) {
	t.Helper()

	db := database.Conn()
	require.NoError(t, db.Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvasID, nodeID).
		Update("app_installation_id", integrationID).Error)

	liveVersion, err := models.FindLiveCanvasVersionInTransaction(db, canvasID)
	require.NoError(t, err)

	nodes := append([]models.Node(nil), liveVersion.Nodes...)
	for i := range nodes {
		if nodes[i].ID != nodeID {
			continue
		}
		id := integrationID.String()
		nodes[i].IntegrationID = &id
	}
	require.NoError(t, db.Model(liveVersion).Update("nodes", datatypes.NewJSONSlice(nodes)).Error)
}

func retargetLiveTriggerInstallation(t *testing.T, r *support.ResourceRegistry, canvasID uuid.UUID, nodeID string) {
	t.Helper()

	other := createReadySentryIntegration(t, r)
	other.State = models.IntegrationStatePending
	require.NoError(t, database.Conn().Save(other).Error)
	require.NoError(t, database.Conn().Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", canvasID, nodeID).
		Update("app_installation_id", other.ID).Error)
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
