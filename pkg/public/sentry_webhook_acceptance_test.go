package public

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	sentryintegration "github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/sentryinbox"
	"github.com/superplanehq/superplane/pkg/workers"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

const durableIssueBody = `{"action":"created","installation":{"uuid":"install-1"},"data":{"issue":{"id":"1"}}}`

// slowSentryIntegration replaces the synchronous Sentry handler. Durable
// acceptance must answer without calling it.
type slowSentryIntegration struct {
	core.Integration
}

func (slowSentryIntegration) HandleRequest(ctx core.HTTPRequestContext) {
	time.Sleep(3 * time.Second)
	ctx.Response.WriteHeader(http.StatusOK)
}

// failingInboxEncryptor cannot encrypt new payloads. It still decrypts, so
// a connection secret stays readable.
type failingInboxEncryptor struct {
	crypto.Encryptor
}

func (failingInboxEncryptor) Encrypt(context.Context, []byte, []byte) ([]byte, error) {
	return nil, errors.New("key service is unavailable")
}

func setupDurableSentryServer(t *testing.T) (*support.ResourceRegistry, *Server) {
	t.Helper()
	t.Setenv(config.EnvSentryAppSlug, "superplane")
	t.Setenv(config.EnvSentryAppClientID, "cid")
	t.Setenv(config.EnvSentryAppClientSecret, "csecret")
	t.Setenv(config.EnvSentryWebhookDurableAcceptance, "yes")

	r := support.Setup(t)
	original := r.Registry.Integrations["sentry"]
	r.Registry.Integrations["sentry"] = slowSentryIntegration{Integration: original}
	t.Cleanup(func() { r.Registry.Integrations["sentry"] = original })

	server, err := NewServer(
		r.Encryptor, r.Registry, jwt.NewSigner("test-client-secret"), support.NewOIDCProvider(),
		"", "", "", "test", "/app/templates", r.AuthService, false,
	)
	require.NoError(t, err)
	return r, server
}

func createDurableSentryConnection(t *testing.T, r *support.ResourceRegistry, metadata map[string]any) (*models.Integration, models.CanvasNode) {
	t.Helper()

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "sentry", support.RandomName("sentry"), map[string]any{})
	require.NoError(t, err)
	integration.Metadata = datatypes.NewJSONType(metadata)
	require.NoError(t, database.Conn().Save(integration).Error)

	_, nodes := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
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

func integrationWebhookRequest(integration *models.Integration, body []byte, secret string) *http.Request {
	request := signedSentryRequest("/api/v1/integrations/"+integration.ID.String()+"/events", body, secret)
	return mux.SetURLVars(request, map[string]string{"integrationID": integration.ID.String()})
}

func inboxCount(t *testing.T) int64 {
	t.Helper()
	var count int64
	require.NoError(t, database.Conn().Model(&models.SentryWebhookInbox{}).Count(&count).Error)
	return count
}

func nodeEventCount(t *testing.T, node models.CanvasNode) int64 {
	t.Helper()
	var count int64
	require.NoError(t, database.Conn().Model(&models.CanvasEvent{}).
		Where("workflow_id = ? AND node_id = ?", node.WorkflowID, node.NodeID).
		Count(&count).Error)
	return count
}

func TestDurableSentryAppWebhook(t *testing.T) {
	t.Run("acknowledges committed work without waiting for processing", func(t *testing.T) {
		r, server := setupDurableSentryServer(t)
		_, node := createDurableSentryConnection(t, r, map[string]any{"hostedApp": true, "installationUUID": "install-1"})

		started := time.Now()
		rec := httptest.NewRecorder()
		server.HandleSentryAppWebhook(rec, signedSentryRequest("/api/v1/sentry/app/webhook", []byte(durableIssueBody), "csecret"))

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Less(t, time.Since(started), time.Second)
		inboxID, err := uuid.Parse(rec.Header().Get("X-Superplane-Sentry-Delivery"))
		require.NoError(t, err)

		var entry models.SentryWebhookInbox
		require.NoError(t, database.Conn().Where("id = ?", inboxID).First(&entry).Error)
		assert.Equal(t, models.SentryWebhookSourceHostedApp, entry.Source)
		assert.Equal(t, models.SentryWebhookStateQueued, entry.State)
		assert.Zero(t, nodeEventCount(t, node))
	})

	t.Run("an invalid signature never enqueues", func(t *testing.T) {
		_, server := setupDurableSentryServer(t)

		rec := httptest.NewRecorder()
		server.HandleSentryAppWebhook(rec, signedSentryRequest("/api/v1/sentry/app/webhook", []byte(durableIssueBody), "wrong-secret"))

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Zero(t, inboxCount(t))
	})

	t.Run("a persistence failure never answers success", func(t *testing.T) {
		_, server := setupDurableSentryServer(t)
		server.encryptor = failingInboxEncryptor{Encryptor: server.encryptor}

		rec := httptest.NewRecorder()
		server.HandleSentryAppWebhook(rec, signedSentryRequest("/api/v1/sentry/app/webhook", []byte(durableIssueBody), "csecret"))

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Zero(t, inboxCount(t))
	})

	t.Run("committed work survives an API restart and a client disconnect", func(t *testing.T) {
		r, server := setupDurableSentryServer(t)
		_, node := createDurableSentryConnection(t, r, map[string]any{"hostedApp": true, "installationUUID": "install-1"})

		request := signedSentryRequest("/api/v1/sentry/app/webhook", []byte(durableIssueBody), "csecret")
		ctx, disconnect := context.WithCancel(request.Context())
		rec := httptest.NewRecorder()
		server.HandleSentryAppWebhook(rec, request.WithContext(ctx))
		disconnect()
		require.Equal(t, http.StatusOK, rec.Code)

		// A new worker process has no memory of the request.
		processor := sentryinbox.NewProcessor(r.Encryptor, r.Registry).WithEventPublisher(func(*models.CanvasEvent) error { return nil })
		claimed, err := workers.NewSentryWebhookWorkerWithProcessor(processor).RunOnce(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 1, claimed)
		assert.Equal(t, int64(1), nodeEventCount(t, node))
	})
}

func TestDurableSentryIntegrationWebhook(t *testing.T) {
	setup := func(t *testing.T) (*support.ResourceRegistry, *Server, *models.Integration, models.CanvasNode) {
		t.Helper()
		r, server := setupDurableSentryServer(t)
		integration, node := createDurableSentryConnection(t, r, map[string]any{})
		integrationCtx := contexts.NewIntegrationContext(database.Conn(), nil, integration, r.Encryptor, r.Registry, nil)
		require.NoError(t, integrationCtx.SetSecret("clientSecret", []byte("integration-secret")))
		return r, server, integration, node
	}

	t.Run("acknowledges committed work with the connection secret", func(t *testing.T) {
		_, server, integration, node := setup(t)

		started := time.Now()
		rec := httptest.NewRecorder()
		server.HandleIntegrationRequest(rec, integrationWebhookRequest(integration, []byte(durableIssueBody), "integration-secret"))

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Less(t, time.Since(started), time.Second)
		var entry models.SentryWebhookInbox
		require.NoError(t, database.Conn().First(&entry).Error)
		assert.Equal(t, models.SentryWebhookSourceIntegration, entry.Source)
		require.NotNil(t, entry.IntegrationID)
		assert.Equal(t, integration.ID, *entry.IntegrationID)
		targets, err := entry.ListTargets(database.Conn())
		require.NoError(t, err)
		assert.Len(t, targets, 1)
		assert.Zero(t, nodeEventCount(t, node))
	})

	t.Run("the public app secret does not authenticate a connection webhook", func(t *testing.T) {
		_, server, integration, _ := setup(t)

		rec := httptest.NewRecorder()
		server.HandleIntegrationRequest(rec, integrationWebhookRequest(integration, []byte(durableIssueBody), "csecret"))

		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Zero(t, inboxCount(t))
	})

	t.Run("a persistence failure never answers success", func(t *testing.T) {
		_, server, integration, _ := setup(t)
		server.encryptor = failingInboxEncryptor{Encryptor: server.encryptor}

		rec := httptest.NewRecorder()
		server.HandleIntegrationRequest(rec, integrationWebhookRequest(integration, []byte(durableIssueBody), "integration-secret"))

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Zero(t, inboxCount(t))
	})
}

// slowSentryAPI makes every Sentry enrichment request slow and then fail,
// so the worker uses the fallback issue payload.
type slowSentryAPI struct {
	delay   time.Duration
	started chan struct{}
	once    sync.Once
}

func (s *slowSentryAPI) Do(request *http.Request) (*http.Response, error) {
	s.once.Do(func() { close(s.started) })
	select {
	case <-time.After(s.delay):
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: http.NoBody, Header: http.Header{}}, nil
	case <-request.Context().Done():
		return nil, request.Context().Err()
	}
}

func TestDurableSentryAcknowledgmentLatencyWithSlowEnrichment(t *testing.T) {
	const requests = 200
	const concurrency = 20

	r, server := setupDurableSentryServer(t)
	sentryintegration.EnableHostedInstallBind(r.Encryptor)
	integration, _ := createDurableSentryConnection(t, r, map[string]any{
		"hostedApp":        true,
		"installationUUID": "install-1",
		"organization":     map[string]any{"slug": "acme"},
	})
	integrationCtx := contexts.NewIntegrationContext(database.Conn(), nil, integration, r.Encryptor, r.Registry, nil)
	require.NoError(t, integrationCtx.SetSecret(sentryintegration.SecretAccessToken, []byte("installation-token")))

	workerCtx, stopWorker := context.WithCancel(t.Context())
	workerDone := make(chan struct{})
	defer func() {
		stopWorker()
		<-workerDone
	}()
	sentryAPI := &slowSentryAPI{delay: 2 * time.Second, started: make(chan struct{})}
	processor := sentryinbox.NewProcessor(r.Encryptor, r.Registry).
		WithHTTP(sentryAPI).
		WithEventPublisher(func(*models.CanvasEvent) error { return nil })
	go func() {
		defer close(workerDone)
		workers.NewSentryWebhookWorkerWithProcessor(processor).Start(workerCtx)
	}()

	// Measure while the worker waits for slow enrichment.
	rec := httptest.NewRecorder()
	server.HandleSentryAppWebhook(rec, signedSentryRequest("/api/v1/sentry/app/webhook", []byte(durableIssueBody), "csecret"))
	require.Equal(t, http.StatusOK, rec.Code)
	select {
	case <-sentryAPI.started:
	case <-time.After(10 * time.Second):
		t.Fatal("the worker did not start enrichment")
	}

	latencies := make([]time.Duration, 0, requests)
	var mu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, concurrency)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		slots <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-slots }()

			started := time.Now()
			rec := httptest.NewRecorder()
			server.HandleSentryAppWebhook(rec, signedSentryRequest("/api/v1/sentry/app/webhook", []byte(durableIssueBody), "csecret"))
			elapsed := time.Since(started)

			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, http.StatusOK, rec.Code)
			latencies = append(latencies, elapsed)
		}()
	}
	wg.Wait()

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p99 := latencies[int(float64(len(latencies))*0.99)-1]
	t.Logf("acknowledgment latency p50=%s p99=%s max=%s", latencies[len(latencies)/2], p99, slices.Max(latencies))
	assert.Less(t, p99, time.Second)
	assert.Equal(t, int64(requests+1), inboxCount(t))
}
