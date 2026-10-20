package public

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support"
)

type analyticsRecorder struct {
	mu     sync.Mutex
	events []productanalytics.Event
}

func (r *analyticsRecorder) Capture(event productanalytics.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *analyticsRecorder) Close() error { return nil }

func (r *analyticsRecorder) Enabled() bool { return true }

func (r *analyticsRecorder) snapshot() []productanalytics.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := make([]productanalytics.Event, len(r.events))
	copy(copied, r.events)
	return copied
}

func useAnalytics(t *testing.T) *analyticsRecorder {
	t.Helper()
	recorder := &analyticsRecorder{}
	previous := productanalytics.SetCurrent(recorder)
	t.Cleanup(func() {
		productanalytics.SetCurrent(previous)
	})
	return recorder
}

func TestOrganizationCreationAnalytics(t *testing.T) {
	recorder := useAnalytics(t)
	require.NoError(t, database.TruncateTables())

	account, err := models.CreateAccount("metrics@example.com", "Metrics User")
	require.NoError(t, err)
	signer := jwt.NewSigner("test")

	authService, err := authorization.NewAuthService()
	require.NoError(t, err)
	encryptor := &crypto.NoOpEncryptor{}
	componentRegistry, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
	require.NoError(t, err)
	server, err := NewServer(encryptor, componentRegistry, signer, support.NewOIDCProvider(), "", "localhost", "", "test", "/app/templates", authService, false)
	require.NoError(t, err)

	postOrganization := func(body []byte) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, "/organizations", bytes.NewReader(body))
		request = request.WithContext(accountContext(account))
		response := httptest.NewRecorder()
		server.createOrganization(response, request)
		return response
	}

	t.Run("rejected creation emits nothing", func(t *testing.T) {
		response := postOrganization([]byte(`{"name":""}`))
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Empty(t, recorder.snapshot())
	})

	t.Run("later organization creation emits one event", func(t *testing.T) {
		body, err := json.Marshal(OrganizationCreationRequest{Name: "Metrics Organization"})
		require.NoError(t, err)
		response := postOrganization(body)
		require.Equal(t, http.StatusOK, response.Code)

		events := recorder.snapshot()
		require.Len(t, events, 1)
		assert.Equal(t, productanalytics.EventOrganizationCreated, events[0].Name)
		assert.Equal(t, "workspace_switcher", events[0].Properties["creation_path"])
		assert.Equal(t, account.ID.String(), events[0].DistinctID)
		assert.Equal(t, productanalytics.ActorUser, events[0].ActorType)
		assert.Equal(t, productanalytics.SourceUI, events[0].Source)
		assert.NotEmpty(t, events[0].OrganizationID)
	})
}

func TestInitialWorkspaceAnalytics(t *testing.T) {
	t.Setenv(config.EnvGitHubAppID, "")
	t.Setenv(config.EnvGitHubAppSlug, "")
	t.Setenv(config.EnvGitHubAppPrivateKey, "")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "")
	recorder := useAnalytics(t)
	resource := support.Setup(t)
	require.NoError(t, models.SaveAccountLinkedAccount(
		database.DB(t.Context()),
		models.NewAccountLinkedAccount(resource.Account.ID, models.ProviderGitHub, "github-owner-id", "github-owner", "GitHub Owner", ""),
	))

	server, err := NewServer(
		resource.Encryptor,
		resource.Registry,
		jwt.NewSigner("test"),
		support.NewOIDCProvider(),
		"",
		"localhost",
		"",
		"test",
		"/app/templates",
		resource.AuthService,
		false,
	)
	require.NoError(t, err)

	attemptID := uuid.New()
	post := func() *httptest.ResponseRecorder {
		body, err := json.Marshal(initialWorkspaceRequest{Owner: "GitHub Owner", AttemptID: attemptID.String()})
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "/account/onboarding", bytes.NewReader(body))
		request = request.WithContext(accountContext(resource.Account))
		response := httptest.NewRecorder()
		server.createInitialWorkspace(response, request)
		return response
	}

	require.Equal(t, http.StatusOK, post().Code)
	require.Equal(t, http.StatusOK, post().Code)

	events := recorder.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, productanalytics.EventOrganizationCreated, events[0].Name)
	assert.Equal(t, "initial_onboarding", events[0].Properties["creation_path"])
	assert.Equal(t, resource.Account.ID.String(), events[0].DistinctID)
	assert.Equal(t, productanalytics.ActorUser, events[0].ActorType)
}
