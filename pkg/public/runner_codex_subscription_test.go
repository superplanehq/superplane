package public

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

func TestCodexCheckpointStoresQuotaWithoutBlockingCredentials(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	_, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	connection, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, now)
	require.NoError(t, err)
	taskID := uuid.New()
	require.NoError(t, connection.BeginLogin(db, taskID, now))
	signer := jwt.NewSigner("test-codex-usage")
	scope := runneraction.CodexSubscriptionScope{OrganizationID: r.Organization.ID, ConnectionID: connection.ID, TaskID: taskID, Generation: connection.ReservationGeneration}
	token, err := runneraction.MintCodexSubscriptionToken(signer, scope, time.Hour)
	require.NoError(t, err)
	server := &Server{jwt: signer, encryptor: r.Encryptor}
	checkpoint := func(token string, usage any) *httptest.ResponseRecorder {
		body, err := json.Marshal(map[string]any{
			"bundle": map[string]any{"access_token": "access-token", "refresh_token": "refresh-token", "id_token": "id-token"},
			"usage":  usage,
		})
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/runner/codex-subscription/checkpoint", bytes.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		server.handleRunnerCodexSubscriptionCheckpoint(response, request)
		return response
	}
	valid := map[string]any{
		"limits": []any{map[string]any{"limitId": "codex", "primary": map[string]any{"usedPercent": 25, "windowDurationMins": 300, "resetsAt": now.Add(time.Hour).Unix()}, "credits": map[string]any{"hasCredits": true, "unlimited": false, "balance": "4712"}}},
		"models": []any{map[string]any{"id": "gpt-5.4", "model": "gpt-5.4", "displayName": "GPT-5.4", "defaultReasoningEffort": "medium", "supportedReasoningEfforts": []any{map[string]any{"reasoningEffort": "xhigh", "description": "Deep"}}, "isDefault": true}},
	}
	require.Equal(t, http.StatusUnauthorized, checkpoint("", valid).Code)
	response := checkpoint(token, valid)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.NoError(t, db.First(connection, "id = ?", connection.ID).Error)
	require.Len(t, connection.UsageLimits, 1)
	assert.Equal(t, 25.0, *connection.UsageLimits[0].Primary.UsedPercent)
	require.NotNil(t, connection.UsageLimits[0].Credits)
	require.NotNil(t, connection.UsageLimits[0].Credits.Balance)
	assert.Equal(t, "4712", *connection.UsageLimits[0].Credits.Balance)
	assert.True(t, connection.UsageLimits[0].Credits.HasCredits)
	require.NotNil(t, connection.UsageObservedAt)
	require.Len(t, connection.AvailableModels, 1)
	assert.Equal(t, "gpt-5.4", connection.AvailableModels[0].Model)
	observed := *connection.UsageObservedAt
	version := connection.CredentialVersion
	invalidUsage := []any{
		nil,
		map[string]any{"limits": []any{map[string]any{"limitId": "codex", "primary": map[string]any{"usedPercent": -1}}}},
		map[string]any{"limits": []any{map[string]any{"limitId": "codex", "primary": map[string]any{"usedPercent": "invalid"}}}},
		map[string]any{"limits": []any{map[string]any{"limitId": "codex", "primary": map[string]any{}}}},
		map[string]any{"limits": []any{map[string]any{"limitId": "codex", "primary": map[string]any{"usedPercent": 25, "resetsAt": int64(253402300800)}}}},
	}
	for _, balance := range []any{"invalid", "NaN", "Infinity", "-1", 4712} {
		invalidUsage = append(invalidUsage, map[string]any{"limits": []any{map[string]any{
			"limitId": "codex", "credits": map[string]any{"balance": balance},
		}}})
	}
	for _, usage := range invalidUsage {
		response := checkpoint(token, usage)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		require.NoError(t, db.First(connection, "id = ?", connection.ID).Error)
		version++
		assert.Equal(t, version, connection.CredentialVersion)
		assert.Equal(t, observed, *connection.UsageObservedAt)
		assert.Equal(t, 25.0, *connection.UsageLimits[0].Primary.UsedPercent)
		assert.Equal(t, "4712", *connection.UsageLimits[0].Credits.Balance)
	}
	response = checkpoint(token, map[string]any{"models": []any{map[string]any{"model": "gpt-5.3"}}})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.NoError(t, db.First(connection, "id = ?", connection.ID).Error)
	require.Len(t, connection.AvailableModels, 1)
	assert.Equal(t, "gpt-5.3", connection.AvailableModels[0].Model)
	assert.Equal(t, observed, *connection.UsageObservedAt)
	assert.Equal(t, 25.0, *connection.UsageLimits[0].Primary.UsedPercent)
	scope.Generation++
	staleToken, err := runneraction.MintCodexSubscriptionToken(signer, scope, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, checkpoint(staleToken, valid).Code)
}

func TestCodexRunCheckpointDoesNotReplaceTheRefreshToken(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	connection, factory, secret := readyCodexRun(t, r)
	original := append([]byte(nil), connection.BundleCiphertext...)
	signer := jwt.NewSigner("test-codex-run")
	var lease models.FactoryCodexLease
	require.NoError(t, db.Where("connection_id = ? AND purpose = ?", connection.ID, models.CodexLeasePurposeRun).First(&lease).Error)
	token, err := runneraction.MintCodexSubscriptionToken(signer, runneraction.CodexSubscriptionScope{
		OrganizationID: r.Organization.ID,
		FactoryID:      factory.ID,
		ConnectionID:   connection.ID,
		TaskID:         lease.TaskID,
		Generation:     lease.Generation,
	}, time.Hour)
	require.NoError(t, err)
	server := &Server{jwt: signer, encryptor: r.Encryptor}
	body, err := json.Marshal(map[string]any{
		"bundle": map[string]any{"access_token": "access-token", "refresh_token": "stolen-refresh-token", "id_token": "id-token"},
		"usage": map[string]any{
			"limits": []any{map[string]any{"limitId": "codex", "primary": map[string]any{"usedPercent": 40, "windowDurationMins": 300, "resetsAt": time.Now().Add(time.Hour).Unix()}, "credits": map[string]any{"hasCredits": false, "balance": "0"}}},
			"models": []any{map[string]any{"model": "gpt-5.4"}},
		},
	})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runner/codex-subscription/checkpoint", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.handleRunnerCodexSubscriptionCheckpoint(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.NoError(t, db.First(connection, "id = ?", connection.ID).Error)
	assert.Equal(t, original, []byte(connection.BundleCiphertext))
	require.Len(t, connection.UsageLimits, 1)
	assert.Equal(t, 40.0, *connection.UsageLimits[0].Primary.UsedPercent)
	require.NotNil(t, connection.UsageLimits[0].Credits)
	require.NotNil(t, connection.UsageLimits[0].Credits.Balance)
	assert.Equal(t, "0", *connection.UsageLimits[0].Credits.Balance)
	assert.NotContains(t, string(connection.BundleCiphertext), secret)
	for _, workspaceID := range []uuid.UUID{uuid.Nil, uuid.New()} {
		wrongScope, err := runneraction.MintCodexSubscriptionToken(signer, runneraction.CodexSubscriptionScope{
			OrganizationID: r.Organization.ID, FactoryID: workspaceID, ConnectionID: connection.ID,
			TaskID: lease.TaskID, Generation: lease.Generation,
		}, time.Hour)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodGet, "/api/v1/runner/codex-subscription/credentials", nil)
		request.Header.Set("Authorization", "Bearer "+wrongScope)
		response := httptest.NewRecorder()
		server.handleRunnerCodexSubscriptionCredentials(response, request)
		assert.Equal(t, http.StatusConflict, response.Code, "the persisted lease must reject another workspace")
	}
}

func TestCodexRunningTaskKeepsCredentialsAfterWorkspaceModelPolicyChange(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	connection, factory, _ := readyCodexRun(t, r)
	var lease models.FactoryCodexLease
	require.NoError(t, db.Where("connection_id = ? AND purpose = ?", connection.ID, models.CodexLeasePurposeRun).First(&lease).Error)
	inherit := false
	_, err := models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, models.UsageProviderOpenAI, models.UsageFundingSourceSubscription, []string{}, &inherit)
	require.NoError(t, err)
	signer := jwt.NewSigner("test-codex-run")
	token, err := runneraction.MintCodexSubscriptionToken(signer, runneraction.CodexSubscriptionScope{
		OrganizationID: r.Organization.ID,
		FactoryID:      factory.ID,
		ConnectionID:   connection.ID,
		TaskID:         lease.TaskID,
		Generation:     lease.Generation,
	}, time.Hour)
	require.NoError(t, err)
	server := &Server{jwt: signer, encryptor: r.Encryptor}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runner/codex-subscription/credentials", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.handleRunnerCodexSubscriptionCredentials(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	require.NoError(t, db.Model(&models.FactoryCodexLease{}).Where("id = ?", lease.ID).Update("revoked_at", time.Now()).Error)
	revoked := httptest.NewRecorder()
	server.handleRunnerCodexSubscriptionCredentials(revoked, request)
	assert.Equal(t, http.StatusConflict, revoked.Code, "revocation must still stop a running task")
}

func TestCodexRefreshIsSingleFlightAndHidesTheRefreshToken(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	connection, _, secret := readyCodexRun(t, r)
	otherWorkspace, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("other-workspace"), "", "")
	require.NoError(t, err)
	var originalLease models.FactoryCodexLease
	require.NoError(t, db.Where("connection_id = ?", connection.ID).First(&originalLease).Error)
	previousTask, err := models.FindRunnerTask(db, originalLease.TaskID)
	require.NoError(t, err)
	otherTask := *previousTask
	otherTask.ID = uuid.New()
	require.NoError(t, db.Create(&otherTask).Error)
	runID := support.CreateCodexOwnedRunInWorkspace(t, db, connection, otherWorkspace.ID, otherTask.ID)
	_, err = models.ReserveFactoryCodexConnection(db, r.Organization.ID, otherWorkspace.ID, runID, otherTask.ID, time.Now(), r.Encryptor)
	require.NoError(t, err)
	secondLease, err := models.FindCodexLeaseByTask(db, otherTask.ID)
	require.NoError(t, err)
	secondSecret, err := r.Encryptor.Decrypt(t.Context(), secondLease.SecretCiphertext, []byte(connection.ID.String()))
	require.NoError(t, err)
	var calls atomic.Int32
	nextAccess := codexTestAccessToken(time.Now().Add(time.Hour))
	server := &Server{
		encryptor: r.Encryptor,
		codexTokenRefresh: func(context.Context, string) (runneraction.CodexTokenSet, error) {
			calls.Add(1)
			return runneraction.CodexTokenSet{AccessToken: nextAccess, RefreshToken: "openai-refresh-2", IDToken: "id-token-2"}, nil
		},
	}
	refresh := func(secret string) *httptest.ResponseRecorder {
		body, err := json.Marshal(map[string]string{"grant_type": "refresh_token", "refresh_token": secret})
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/runner/codex-subscription/refresh", bytes.NewReader(body))
		response := httptest.NewRecorder()
		server.handleRunnerCodexSubscriptionRefresh(response, request)
		return response
	}
	responses := make(chan *httptest.ResponseRecorder, 2)
	start := make(chan struct{})
	for _, credential := range []string{secret, string(secondSecret)} {
		go func() { <-start; responses <- refresh(credential) }()
	}
	close(start)
	for range 2 {
		response := <-responses
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	}
	second := refresh(secret)
	require.Equal(t, http.StatusOK, second.Code, second.Body.String())
	assert.Equal(t, int32(1), calls.Load())
	var payload map[string]string
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &payload))
	assert.Equal(t, secret, payload["refresh_token"])
	assert.Equal(t, nextAccess, payload["access_token"])
	assert.NotContains(t, second.Body.String(), "openai-refresh")
	require.NoError(t, db.First(connection, "id = ?", connection.ID).Error)
	assert.Contains(t, string(connection.BundleCiphertext), "openai-refresh-2")
	assert.NotContains(t, string(connection.BundleCiphertext), secret)

	credentials := httptest.NewRequest(http.MethodGet, "/api/v1/runner/codex-subscription/credentials", nil)
	lease := originalLease
	signer := jwt.NewSigner("test-codex-run")
	server.jwt = signer
	token, err := runneraction.MintCodexSubscriptionToken(signer, runneraction.CodexSubscriptionScope{
		OrganizationID: connection.OrganizationID,
		FactoryID:      *lease.FactoryID,
		ConnectionID:   connection.ID,
		TaskID:         lease.TaskID,
		Generation:     lease.Generation,
	}, time.Hour)
	require.NoError(t, err)
	credentials.Header.Set("Authorization", "Bearer "+token)
	credentialResponse := httptest.NewRecorder()
	server.handleRunnerCodexSubscriptionCredentials(credentialResponse, credentials)
	require.Equal(t, http.StatusOK, credentialResponse.Code, credentialResponse.Body.String())
	assert.Contains(t, credentialResponse.Body.String(), secret)
	assert.NotContains(t, credentialResponse.Body.String(), "openai-refresh-2")

	assert.Equal(t, http.StatusUnauthorized, refresh("invalid-lease-secret").Code)
	require.NoError(t, db.Model(&originalLease).Update("revoked_at", time.Now()).Error)
	assert.Equal(t, http.StatusUnauthorized, refresh(secret).Code)
	assert.Equal(t, int32(1), calls.Load(), "invalid and revoked leases must not reach the provider")
}

// The provider can rotate a token before the client disconnects. The stored
// bundle and subsequent credential fetch must still use the rotated token.
func TestCodexRefreshPersistsAfterRequestCancellation(t *testing.T) {
	for _, replacement := range []string{"openai-refresh-2", ""} {
		t.Run("replacement="+replacement, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			connection, factory, secret := readyCodexRun(t, r)
			version := connection.CredentialVersion
			nextAccess := codexTestAccessToken(time.Now().Add(time.Hour))
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var exchangeDeadline time.Time
			encryptor := &codexRefreshEncryptor{Encryptor: r.Encryptor, beforeEncrypt: func(ctx context.Context) error {
				deadline, bounded := ctx.Deadline()
				assert.True(t, bounded, "persistence must have an operation deadline")
				assert.True(t, deadline.After(exchangeDeadline), "exchange must leave time to save credentials")
				return ctx.Err()
			}}
			server := &Server{
				encryptor: encryptor,
				codexTokenRefresh: func(operation context.Context, refreshToken string) (runneraction.CodexTokenSet, error) {
					assert.Equal(t, "openai-refresh", refreshToken)
					var bounded bool
					exchangeDeadline, bounded = operation.Deadline()
					assert.True(t, bounded, "provider exchange must have a deadline")
					cancel()
					assert.NoError(t, operation.Err(), "request cancellation must not cancel the provider exchange")
					return runneraction.CodexTokenSet{AccessToken: nextAccess, RefreshToken: replacement, IDToken: "id-token-2"}, nil
				},
			}
			request := httptest.NewRequest(http.MethodPost, "/api/v1/runner/codex-subscription/refresh", strings.NewReader(`{"refresh_token":"`+secret+`"}`)).WithContext(ctx)
			response := httptest.NewRecorder()
			server.handleRunnerCodexSubscriptionRefresh(response, request)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			require.NoError(t, db.First(connection, "id = ?", connection.ID).Error)
			assert.Equal(t, version+1, connection.CredentialVersion)
			assert.Equal(t, models.CodexConnectionStatusReady, connection.Status)
			plain, err := r.Encryptor.Decrypt(t.Context(), connection.BundleCiphertext, []byte(connection.ID.String()))
			require.NoError(t, err)
			assert.Equal(t, nextAccess, runneraction.CodexAuthToken(plain, "access_token"))
			expectedRefresh := replacement
			if expectedRefresh == "" {
				expectedRefresh = "openai-refresh"
			}
			assert.Equal(t, expectedRefresh, runneraction.CodexAuthToken(plain, "refresh_token"))

			var lease models.FactoryCodexLease
			err = db.Where("connection_id = ? AND purpose = ?", connection.ID, models.CodexLeasePurposeRun).First(&lease).Error
			require.NoError(t, err)
			signer := jwt.NewSigner("test-codex-cancellation")
			server.jwt = signer
			token, err := runneraction.MintCodexSubscriptionToken(signer, runneraction.CodexSubscriptionScope{
				OrganizationID: connection.OrganizationID, FactoryID: factory.ID,
				ConnectionID: connection.ID, TaskID: lease.TaskID, Generation: lease.Generation,
			}, time.Hour)
			require.NoError(t, err)
			credentials := httptest.NewRequest(http.MethodGet, "/api/v1/runner/codex-subscription/credentials", nil)
			credentials.Header.Set("Authorization", "Bearer "+token)
			fetched := httptest.NewRecorder()
			server.handleRunnerCodexSubscriptionCredentials(fetched, credentials)
			require.Equal(t, http.StatusOK, fetched.Code, fetched.Body.String())
			assert.Contains(t, fetched.Body.String(), nextAccess)
			assert.Contains(t, fetched.Body.String(), secret)
			assert.NotContains(t, fetched.Body.String(), expectedRefresh)
		})
	}
}

type codexRefreshEncryptor struct {
	crypto.Encryptor
	beforeEncrypt func(context.Context) error
}

func (e *codexRefreshEncryptor) Encrypt(ctx context.Context, plain, associated []byte) ([]byte, error) {
	if err := e.beforeEncrypt(ctx); err != nil {
		return nil, err
	}
	return e.Encryptor.Encrypt(ctx, plain, associated)
}

// Provider response classification must reach durable connection state and the
// update event used by the reconnect UI, rather than a preclassified mock error.
func TestCodexRefreshFailureRequiresReconnect(t *testing.T) {
	for _, test := range []struct {
		name      string
		status    int
		body      string
		permanent bool
	}{
		{"invalid grant", http.StatusBadRequest, `{"error":"invalid_grant"}`, true},
		{"rate limit with token code", http.StatusTooManyRequests, `{"error":"refresh_token_reused"}`, false},
		{"server error with token code", http.StatusServiceUnavailable, `{"code":"invalid_grant"}`, false},
		{"quota exhausted", http.StatusBadRequest, `{"error":{"code":"insufficient_quota"}}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			connection, factory, secret := readyCodexRun(t, r)
			original := append([]byte(nil), connection.BundleCiphertext...)
			version := connection.CredentialVersion
			channel, queue := taskOwnerUpdateQueue(t)
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer provider.Close()
			client := provider.Client()
			client.Transport = codexProviderTransport{upstream: provider.URL, transport: http.DefaultTransport}
			server := &Server{
				encryptor: r.Encryptor,
				codexTokenRefresh: func(ctx context.Context, token string) (runneraction.CodexTokenSet, error) {
					return runneraction.RefreshCodexTokens(ctx, client, token)
				},
			}
			request := httptest.NewRequest(http.MethodPost, "/api/v1/runner/codex-subscription/refresh", strings.NewReader(`{"refresh_token":"`+secret+`"}`))
			response := httptest.NewRecorder()
			server.handleRunnerCodexSubscriptionRefresh(response, request)
			require.NoError(t, db.First(connection, "id = ?", connection.ID).Error)
			assert.Equal(t, original, []byte(connection.BundleCiphertext))
			assert.Equal(t, version, connection.CredentialVersion)
			if !test.permanent {
				assert.Equal(t, http.StatusServiceUnavailable, response.Code, response.Body.String())
				assert.Equal(t, models.CodexConnectionStatusReady, connection.Status)
				_, found, err := channel.Get(queue, true)
				require.NoError(t, err)
				assert.False(t, found, "retryable errors must not publish a reconnect transition")
				return
			}
			require.Equal(t, http.StatusUnauthorized, response.Code, response.Body.String())
			assert.Contains(t, response.Body.String(), "invalid_grant")
			assert.Equal(t, models.CodexConnectionStatusReconnectRequired, connection.Status)
			require.Eventually(t, func() bool {
				delivery, found, err := channel.Get(queue, true)
				require.NoError(t, err)
				if !found {
					return false
				}
				var update pb.FactoryWorkOrderUpdatedMessage
				require.NoError(t, proto.Unmarshal(delivery.Body, &update))
				return update.Reason == "codex_connection" && update.FactoryId == factory.ID.String()
			}, 3*time.Second, 10*time.Millisecond)
			_, err := models.ReserveFactoryCodexConnection(db, connection.OrganizationID, factory.ID, support.CreateCodexOwnedRun(t, db, connection), uuid.New(), time.Now(), r.Encryptor)
			require.ErrorIs(t, err, models.ErrSubscriptionOwnerInvalid)
		})
	}
}

type codexProviderTransport struct {
	upstream  string
	transport http.RoundTripper
}

func (c codexProviderTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	forward := request.Clone(request.Context())
	endpoint, err := url.Parse(c.upstream)
	if err != nil {
		return nil, err
	}
	forward.URL = endpoint
	return c.transport.RoundTrip(forward)
}

func TestCodexRefreshDoesNotReportReconnectAfterCommitFailure(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	connection, _, secret := readyCodexRun(t, r)
	channel, queue := taskOwnerUpdateQueue(t)
	// A deferred foreign key fails at COMMIT, after the reconnect update has
	// succeeded. The temporary table is removed when the transaction rolls back.
	callback := "test:codex_refresh_commit_failure"
	require.NoError(t, db.Callback().Update().After("gorm:update").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table != "factory_codex_connections" {
			return
		}
		isolated := tx.Session(&gorm.Session{NewDB: true})
		require.NoError(t, isolated.Exec(`CREATE TEMP TABLE codex_refresh_commit_failure (
			id INT PRIMARY KEY,
			parent INT REFERENCES codex_refresh_commit_failure(id) DEFERRABLE INITIALLY DEFERRED
		) ON COMMIT DROP`).Error)
		require.NoError(t, isolated.Exec("INSERT INTO codex_refresh_commit_failure VALUES (1, 2)").Error)
	}))
	t.Cleanup(func() { require.NoError(t, db.Callback().Update().Remove(callback)) })
	server := &Server{
		encryptor: r.Encryptor,
		codexTokenRefresh: func(context.Context, string) (runneraction.CodexTokenSet, error) {
			return runneraction.CodexTokenSet{}, &runneraction.CodexTokenRefreshError{Permanent: true, Code: "invalid_grant"}
		},
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runner/codex-subscription/refresh", strings.NewReader(`{"refresh_token":"`+secret+`"}`))
	response := httptest.NewRecorder()
	server.handleRunnerCodexSubscriptionRefresh(response, request)
	assert.Equal(t, http.StatusInternalServerError, response.Code, response.Body.String())
	assert.NotContains(t, response.Body.String(), "invalid_grant")
	require.NoError(t, db.First(connection, "id = ?", connection.ID).Error)
	assert.Equal(t, models.CodexConnectionStatusReady, connection.Status)
	_, found, err := channel.Get(queue, true)
	require.NoError(t, err)
	assert.False(t, found, "failed commits must not publish a reconnect transition")
}

func readyCodexRun(t *testing.T, r *support.ResourceRegistry) (*models.FactoryCodexConnection, *models.Factory, string) {
	t.Helper()
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	connection, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, now)
	require.NoError(t, err)
	taskID := uuid.New()
	require.NoError(t, connection.BeginLogin(db, taskID, now))
	bundle, err := json.Marshal(map[string]any{
		"access_token":  codexTestAccessToken(now.Add(time.Minute)),
		"refresh_token": "openai-refresh",
		"id_token":      "id-token",
	})
	require.NoError(t, err)
	require.NoError(t, connection.StoreBundle(db, taskID, connection.ReservationGeneration, 1, bundle, "ada@example.com", true, now))
	require.NoError(t, db.Where("connection_id = ?", connection.ID).Delete(&models.FactoryCodexLease{}).Error)
	require.NoError(t, db.Model(connection).Update("status", models.CodexConnectionStatusReady).Error)
	fleet := models.RunnerFleet{ID: uuid.New(), Slug: support.RandomName("fleet"), ScopeType: models.RunnerFleetScopeInstallation, Enabled: true, RunnerVersion: "test"}
	require.NoError(t, fleet.Create(db))
	runTask := &models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued, QueuedAt: now, PayloadCiphertext: []byte("payload")}
	require.NoError(t, db.Create(runTask).Error)
	reserved, err := models.ReserveFactoryCodexConnection(db, r.Organization.ID, factory.ID, support.CreateCodexOwnedRun(t, db, connection, runTask.ID), runTask.ID, now, r.Encryptor)
	require.NoError(t, err)
	var lease models.FactoryCodexLease
	require.NoError(t, db.Where("connection_id = ? AND purpose = ?", reserved.ID, models.CodexLeasePurposeRun).First(&lease).Error)
	secret, err := r.Encryptor.Decrypt(t.Context(), lease.SecretCiphertext, []byte(reserved.ID.String()))
	require.NoError(t, err)
	require.NoError(t, db.First(reserved, "id = ?", reserved.ID).Error)
	return reserved, factory, string(secret)
}

func codexTestAccessToken(exp time.Time) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d}`, exp.Unix())))
	return header + "." + payload + ".x"
}

func TestCodexConnectionChangesNotifyEveryActiveOrganizationWorkspace(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	first, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("first"), "", "")
	require.NoError(t, err)
	second, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("second"), "", "")
	require.NoError(t, err)
	deleted, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("deleted"), "", "")
	require.NoError(t, err)
	require.NoError(t, db.Delete(deleted).Error)
	channel, queue := taskOwnerUpdateQueue(t)
	require.NoError(t, messages.PublishOrganizationCodexConnectionUpdated(db, r.Organization.ID))
	workspaceIDs := []string{first.ID.String(), second.ID.String(), deleted.ID.String()}
	var received []string
	require.Eventually(t, func() bool {
		delivery, found, err := channel.Get(queue, true)
		require.NoError(t, err)
		if !found {
			return len(received) == 2
		}
		var message pb.FactoryWorkOrderUpdatedMessage
		require.NoError(t, proto.Unmarshal(delivery.Body, &message))
		if message.Reason != "codex_connection" || !slices.Contains(workspaceIDs, message.FactoryId) {
			return false
		}
		assert.Empty(t, message.OrderId)
		received = append(received, message.FactoryId)
		return false
	}, 3*time.Second, 10*time.Millisecond)
	assert.ElementsMatch(t, []string{first.ID.String(), second.ID.String()}, received)
}
