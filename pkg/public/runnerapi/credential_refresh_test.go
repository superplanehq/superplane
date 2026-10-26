package runnerapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestRunnerProviderKeyPersistsCleanupIntentBeforeCreation(t *testing.T) {
	fixture := createRunnerProviderFixture(t)
	db := database.DB(t.Context())
	var intent models.RunnerCredentialCleanup
	httpClient := runnerCredentialHTTPFunc(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, request.Method)
		require.Equal(t, "Bearer management-key", request.Header.Get("Authorization"))
		var body struct {
			Name      string    `json:"name"`
			ExpiresAt time.Time `json:"expires_at"`
		}
		require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
		// This query uses another connection before the provider creates the key.
		// A process crash after this point must leave a durable cleanup record.
		require.NoError(t, db.Where("task_id = ? AND attempt_id = ?", fixture.task.ID, fixture.preparation.AttemptID).First(&intent).Error)
		require.Equal(t, body.Name, intent.Name)
		require.Equal(t, fixture.provider.ID, intent.ProviderResourceID)
		require.WithinDuration(t, body.ExpiresAt, intent.ExpiresAt, time.Second)
		require.True(t, intent.ExpiresAt.After(time.Now()))
		require.Empty(t, intent.KeyHash)
		require.Nil(t, intent.AcceptedAt)
		return runnerCredentialResponse(`{"key":"runner-key","data":{"hash":"key-hash"}}`), nil
	})
	_, err := fixture.server.mintRunnerProviderKey(t.Context(), fixture.task, fixture.preparation, fixture.provider, time.Hour, httpClient)
	require.NoError(t, err)
	require.NoError(t, db.First(&intent, "id = ?", intent.ID).Error)
	require.Equal(t, "key-hash", intent.KeyHash)
	require.Nil(t, intent.AcceptedAt)
}

func TestUncertainRunnerProviderCreationReconcilesExactNameBeforeNextMint(t *testing.T) {
	fixture := createRunnerProviderFixture(t)
	db := database.DB(t.Context())
	var uncertain models.RunnerCredentialCleanup
	var calls []string
	posts := 0
	httpClient := runnerCredentialHTTPFunc(func(request *http.Request) (*http.Response, error) {
		calls = append(calls, request.Method+" "+request.URL.RequestURI())
		switch request.Method {
		case http.MethodPost:
			posts++
			if posts == 1 {
				require.NoError(t, db.Where("task_id = ?", fixture.task.ID).First(&uncertain).Error)
				return nil, errors.New("connection reset after provider creation")
			}
			require.NoError(t, db.First(&uncertain, "id = ?", uncertain.ID).Error)
			require.NotNil(t, uncertain.CompletedAt, "the previous key must be revoked before another is created")
			return runnerCredentialResponse(`{"key":"replacement-key","data":{"hash":"replacement-hash"}}`), nil
		case http.MethodGet:
			require.Equal(t, "true", request.URL.Query().Get("include_disabled"))
			if request.URL.Query().Get("offset") == "0" {
				body, err := json.Marshal(map[string]any{"data": []map[string]string{
					{"name": uncertain.Name, "hash": "uncertain-hash"},
					{"name": uncertain.Name + "-another-attempt", "hash": "unrelated-hash"},
				}})
				require.NoError(t, err)
				return runnerCredentialResponse(string(body)), nil
			}
			require.Equal(t, "2", request.URL.Query().Get("offset"))
			return runnerCredentialResponse(`{"data":[]}`), nil
		case http.MethodDelete:
			require.Equal(t, "/api/v1/keys/uncertain-hash", request.URL.Path)
			return runnerCredentialResponse(`{}`), nil
		default:
			t.Fatalf("unexpected provider request: %s", request.Method)
			return nil, nil
		}
	})
	_, err := fixture.server.mintRunnerProviderKey(t.Context(), fixture.task, fixture.preparation, fixture.provider, time.Hour, httpClient)
	require.Error(t, err)
	require.Empty(t, uncertain.KeyHash)
	require.NoError(t, fixture.preparation.Retry(db, time.Now(), time.Now()))
	next, err := models.ClaimRunnerTaskPreparation(db, fixture.task.ID, fixture.runnerID, fixture.connectionID, time.Now())
	require.NoError(t, err)
	require.NotEqual(t, fixture.preparation.AttemptID, next.AttemptID)
	_, err = fixture.server.mintRunnerProviderKey(t.Context(), fixture.task, next, fixture.provider, time.Hour, httpClient)
	require.NoError(t, err)
	require.Equal(t, []string{
		"POST /api/v1/keys",
		"GET /api/v1/keys?include_disabled=true&offset=0",
		"GET /api/v1/keys?include_disabled=true&offset=2",
		"DELETE /api/v1/keys/uncertain-hash",
		"POST /api/v1/keys",
	}, calls)
}

func TestCancellationAfterProviderMintRetainsCleanupAndRejectsAcceptance(t *testing.T) {
	fixture := createRunnerProviderFixture(t)
	db := database.DB(t.Context())
	var deleted bool
	httpClient := runnerCredentialHTTPFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method == http.MethodPost {
			return runnerCredentialResponse(`{"key":"unused-key","data":{"hash":"unused-hash"}}`), nil
		}
		require.Equal(t, http.MethodDelete, request.Method)
		require.Equal(t, "/api/v1/keys/unused-hash", request.URL.Path)
		deleted = true
		return runnerCredentialResponse(`{}`), nil
	})
	_, err := fixture.server.mintRunnerProviderKey(t.Context(), fixture.task, fixture.preparation, fixture.provider, time.Hour, httpClient)
	require.NoError(t, err)
	require.NoError(t, fixture.task.RequestCancel(db, time.Now()))
	err = fixture.preparation.Accept(db, fixture.runnerID, fixture.connectionID, []byte(`{"environment":[]}`), time.Now().Add(time.Hour), time.Now())
	require.ErrorIs(t, err, models.ErrRunnerPreparationUnavailable)
	records, err := models.ListRunnerCredentialCleanup(db, time.Now())
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, fixture.task.ID, records[0].TaskID)
	require.Nil(t, records[0].AcceptedAt)
	require.NoError(t, fixture.server.cleanupRunnerCredential(t.Context(), &records[0], httpClient))
	require.True(t, deleted)
	var persisted models.RunnerCredentialCleanup
	require.NoError(t, db.First(&persisted, "id = ?", records[0].ID).Error)
	require.NotNil(t, persisted.CompletedAt)
}

func TestRunnerProviderCleanupKeepsOriginalIssuerUntilFiniteExpiry(t *testing.T) {
	fixture := createRunnerProviderFixture(t)
	db := database.DB(t.Context())
	httpClient := runnerCredentialHTTPFunc(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, request.Method)
		return runnerCredentialResponse(`{"key":"runner-key","data":{"hash":"original-issuer-hash"}}`), nil
	})
	_, err := fixture.server.mintRunnerProviderKey(t.Context(), fixture.task, fixture.preparation, fixture.provider, time.Hour, httpClient)
	require.NoError(t, err)
	var record models.RunnerCredentialCleanup
	require.NoError(t, db.First(&record, "task_id = ?", fixture.task.ID).Error)
	require.NotEmpty(t, record.IssuerCredentialIdentity)
	require.NoError(t, db.Model(fixture.provider).Update("management_key", []byte("another-account-management-key")).Error)
	called := false
	replacementIssuer := runnerCredentialHTTPFunc(func(*http.Request) (*http.Response, error) {
		called = true
		// The replacement account could return 404 for the original hash. It
		// cannot prove that the credential was revoked at the original issuer.
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})
	require.ErrorIs(t, fixture.server.cleanupRunnerCredential(t.Context(), &record, replacementIssuer), errInvalidCredentialDescriptor)
	require.False(t, called)
	require.NoError(t, db.First(&record, "id = ?", record.ID).Error)
	require.Nil(t, record.CompletedAt)
	record.ExpiresAt = time.Now().Add(-time.Second)
	require.NoError(t, fixture.server.cleanupRunnerCredential(t.Context(), &record, replacementIssuer))
	require.NoError(t, db.First(&record, "id = ?", record.ID).Error)
	require.NotNil(t, record.CompletedAt)
	require.False(t, called)
}

type runnerProviderFixture struct {
	server       *Server
	task         *models.RunnerTask
	preparation  *models.RunnerTaskPreparation
	provider     *models.HostedLLMProvider
	runnerID     uuid.UUID
	connectionID uuid.UUID
}

func createRunnerProviderFixture(t *testing.T) runnerProviderFixture {
	t.Helper()
	r := support.Setup(t)
	db := database.DB(t.Context())
	runner, _, fleet := createRegistrationFixture(t)
	now := time.Now()
	task := &models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID,
		Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued,
		PayloadCiphertext: []byte(`{}`), QueuedAt: now, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, task.Reserve(db, runner.ID))
	connectionID := uuid.New()
	require.NoError(t, db.Model(runner).Updates(map[string]any{"state": models.RunnerStateIdle, "current_connection_id": connectionID}).Error)
	preparation, err := models.ClaimRunnerTaskPreparation(db, task.ID, runner.ID, connectionID, now)
	require.NoError(t, err)
	provider := &models.HostedLLMProvider{ID: uuid.New(), Provider: models.UsageProviderOpenRouter,
		Enabled: true, APIKey: []byte("provider-key"), ManagementKey: []byte("management-key"),
		AllowedModels: datatypes.NewJSONSlice([]string{"openai/model"}), CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Create(provider).Error)
	return runnerProviderFixture{server: &Server{encryptor: crypto.NewNoOpEncryptor()}, task: task,
		preparation: preparation, provider: provider, runnerID: runner.ID, connectionID: connectionID}
}

type runnerCredentialHTTPFunc func(*http.Request) (*http.Response, error)

func (f runnerCredentialHTTPFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func runnerCredentialResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}
