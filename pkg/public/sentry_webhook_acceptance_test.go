package public

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	sentryintegration "github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

const issueWebhookBody = `{"action":"created","installation":{"uuid":"install-1"},"data":{"issue":{"id":"123","title":"Broken deploy","permalink":"https://sentry.io/issues/123/","project":{"slug":"payments"}}}}`

type slowSentryIntegration struct{ core.Integration }

func (slowSentryIntegration) HandleRequest(ctx core.HTTPRequestContext) {
	time.Sleep(3 * time.Second)
	ctx.Response.WriteHeader(http.StatusOK)
}

func sentryAdmissionServer(t *testing.T) (*support.ResourceRegistry, *Server) {
	t.Helper()
	t.Setenv(config.EnvSentryAppSlug, "superplane")
	t.Setenv(config.EnvSentryAppClientID, "cid")
	t.Setenv(config.EnvSentryAppClientSecret, "csecret")

	resources := support.Setup(t)
	server, err := NewServer(
		resources.Encryptor, resources.Registry, jwt.NewSigner("test-client-secret"), support.NewOIDCProvider(),
		"", "", "", "test", "/app/templates", resources.AuthService, false,
	)
	require.NoError(t, err)
	return resources, server
}

func sentryAdmissionConnection(t *testing.T, resources *support.ResourceRegistry, metadata map[string]any) (*models.Integration, models.CanvasNode) {
	t.Helper()
	integration, err := models.CreateIntegration(uuid.New(), resources.Organization.ID, "sentry", support.RandomName("sentry"), map[string]any{})
	require.NoError(t, err)
	integration.Metadata = datatypes.NewJSONType(metadata)
	require.NoError(t, database.Conn().Save(integration).Error)

	_, nodes := support.CreateCanvas(t, resources.Organization.ID, resources.User, []models.CanvasNode{{
		NodeID:        "listener",
		Name:          "listener",
		Type:          models.NodeTypeTrigger,
		Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "sentry.onIssue"}}),
		Configuration: datatypes.NewJSONType(map[string]any{}),
	}}, nil)
	_, err = models.CreateIntegrationSubscription(&nodes[0], integration, map[string]any{"resources": []string{"issue"}})
	require.NoError(t, err)
	return integration, nodes[0]
}

func signedSentryRequest(target string, body []byte, secret string) *http.Request {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	request := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	request.Header.Set("Sentry-Hook-Signature", hex.EncodeToString(mac.Sum(nil)))
	request.Header.Set("Sentry-Hook-Resource", "issue")
	return request
}

func sentryConnectionRequest(integration *models.Integration, body []byte, secret string) *http.Request {
	request := signedSentryRequest("/api/v1/integrations/"+integration.ID.String()+"/events", body, secret)
	return mux.SetURLVars(request, map[string]string{"integrationID": integration.ID.String()})
}

func sentryNodeEvents(t *testing.T, node models.CanvasNode) []models.CanvasEvent {
	t.Helper()
	var events []models.CanvasEvent
	require.NoError(t, database.Conn().Where("workflow_id = ? AND node_id = ?", node.WorkflowID, node.NodeID).Find(&events).Error)
	return events
}

func TestSentryHostedWebhookAdmitsWorkflowEvents(t *testing.T) {
	resources, server := sentryAdmissionServer(t)
	_, node := sentryAdmissionConnection(t, resources, map[string]any{"hostedApp": true, "installationUUID": "install-1"})
	original := resources.Registry.Integrations["sentry"]
	resources.Registry.Integrations["sentry"] = slowSentryIntegration{Integration: original}
	t.Cleanup(func() { resources.Registry.Integrations["sentry"] = original })

	started := time.Now()
	recorder := httptest.NewRecorder()
	server.HandleSentryAppWebhook(recorder, signedSentryRequest("/api/v1/sentry/app/webhook", []byte(issueWebhookBody), "csecret"))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Less(t, time.Since(started), 2*time.Second)
	events := sentryNodeEvents(t, node)
	require.Len(t, events, 1)
	assert.Equal(t, models.CanvasEventStatePending, events[0].State)
	assert.Equal(t, "123", func() string {
		id, _ := sentryintegration.IssueIDFromEventData(events[0].Data.Data())
		return id
	}())
	receiptID, ok := sentryintegration.ReceiptIDFromEventData(events[0].Data.Data())
	require.True(t, ok)
	assert.Equal(t, receiptID.String(), recorder.Header().Get(sentryintegration.HeaderWebhookReceipt))
	var receipt models.SentryWebhookReceipt
	require.NoError(t, database.Conn().First(&receipt, receiptID).Error)
	assert.Equal(t, models.SentryWebhookOutcomeAccepted, receipt.Outcome)
}

func TestSentryHostedWebhookRejectsBadSignatureAndFailedPersistence(t *testing.T) {
	resources, server := sentryAdmissionServer(t)
	_, node := sentryAdmissionConnection(t, resources, map[string]any{"hostedApp": true, "installationUUID": "install-1"})

	bad := httptest.NewRecorder()
	server.HandleSentryAppWebhook(bad, signedSentryRequest("/api/v1/sentry/app/webhook", []byte(issueWebhookBody), "wrong"))
	assert.Equal(t, http.StatusBadRequest, bad.Code)
	assert.Empty(t, sentryNodeEvents(t, node))
	var receiptsBefore int64
	require.NoError(t, database.Conn().Model(&models.SentryWebhookReceipt{}).Count(&receiptsBefore).Error)

	request := signedSentryRequest("/api/v1/sentry/app/webhook", []byte(issueWebhookBody), "csecret")
	ctx, cancel := context.WithCancel(request.Context())
	cancel()
	failed := httptest.NewRecorder()
	server.HandleSentryAppWebhook(failed, request.WithContext(ctx))
	assert.Equal(t, http.StatusInternalServerError, failed.Code)
	assert.Empty(t, sentryNodeEvents(t, node))
	var receiptsAfter int64
	require.NoError(t, database.Conn().Model(&models.SentryWebhookReceipt{}).Count(&receiptsAfter).Error)
	assert.Equal(t, receiptsBefore, receiptsAfter)
}

func TestSentryDirectWebhookAdmitsOnlyVerifiedEvents(t *testing.T) {
	resources, server := sentryAdmissionServer(t)
	integration, node := sentryAdmissionConnection(t, resources, map[string]any{})
	integrationCtx := contexts.NewIntegrationContext(database.Conn(), nil, integration, resources.Encryptor, resources.Registry, nil)
	require.NoError(t, integrationCtx.SetSecret("clientSecret", []byte("direct-secret")))

	bad := httptest.NewRecorder()
	server.HandleIntegrationRequest(bad, sentryConnectionRequest(integration, []byte(issueWebhookBody), "csecret"))
	assert.Equal(t, http.StatusForbidden, bad.Code)
	assert.Empty(t, sentryNodeEvents(t, node))

	good := httptest.NewRecorder()
	server.HandleIntegrationRequest(good, sentryConnectionRequest(integration, []byte(issueWebhookBody), "direct-secret"))
	require.Equal(t, http.StatusOK, good.Code)
	events := sentryNodeEvents(t, node)
	require.Len(t, events, 1)
	_, ok := sentryintegration.ReceiptIDFromEventData(events[0].Data.Data())
	assert.True(t, ok)
}

func TestSentryDirectWebhookDoesNotWaitForIssueAPI(t *testing.T) {
	resources, server := sentryAdmissionServer(t)
	var apiCalls atomic.Int64
	issueAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiCalls.Add(1)
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(issueAPI.Close)

	integration, node := sentryAdmissionConnection(t, resources, map[string]any{})
	integration.Configuration = datatypes.NewJSONType(map[string]any{
		"baseUrl": issueAPI.URL, "userToken": "token",
	})
	require.NoError(t, database.Conn().Save(integration).Error)
	integrationCtx := contexts.NewIntegrationContext(database.Conn(), nil, integration, resources.Encryptor, resources.Registry, nil)
	require.NoError(t, integrationCtx.SetSecret("clientSecret", []byte("direct-secret")))

	started := time.Now()
	recorder := httptest.NewRecorder()
	server.HandleIntegrationRequest(recorder, sentryConnectionRequest(integration, []byte(issueWebhookBody), "direct-secret"))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Less(t, time.Since(started), 2*time.Second)
	assert.Zero(t, apiCalls.Load())
	require.Len(t, sentryNodeEvents(t, node), 1)
}

func TestSentryDirectWebhookFailsAdmissionWhenEventCannotPersist(t *testing.T) {
	resources, server := sentryAdmissionServer(t)
	integration, node := sentryAdmissionConnection(t, resources, map[string]any{})
	integrationCtx := contexts.NewIntegrationContext(database.Conn(), nil, integration, resources.Encryptor, resources.Registry, nil)
	require.NoError(t, integrationCtx.SetSecret("clientSecret", []byte("direct-secret")))

	t.Setenv("SUPERPLANE_MAX_PAYLOAD_SIZE", "128")
	var receiptsBefore int64
	require.NoError(t, database.Conn().Model(&models.SentryWebhookReceipt{}).Count(&receiptsBefore).Error)
	recorder := httptest.NewRecorder()
	server.HandleIntegrationRequest(recorder, sentryConnectionRequest(integration, []byte(issueWebhookBody), "direct-secret"))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Empty(t, sentryNodeEvents(t, node))
	var receiptsAfter int64
	require.NoError(t, database.Conn().Model(&models.SentryWebhookReceipt{}).Count(&receiptsAfter).Error)
	assert.Equal(t, receiptsBefore, receiptsAfter)
}
