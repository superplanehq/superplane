package public

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
)

func TestCodexCheckpointStoresQuotaWithoutBlockingCredentials(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
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
		"limits": []any{map[string]any{"limitId": "codex", "primary": map[string]any{"usedPercent": 25, "windowDurationMins": 300, "resetsAt": now.Add(time.Hour).Unix()}}},
		"models": []any{map[string]any{"id": "gpt-5.4", "model": "gpt-5.4", "displayName": "GPT-5.4", "defaultReasoningEffort": "medium", "supportedReasoningEfforts": []any{map[string]any{"reasoningEffort": "xhigh", "description": "Deep"}}, "isDefault": true}},
	}
	require.Equal(t, http.StatusUnauthorized, checkpoint("", valid).Code)
	response := checkpoint(token, valid)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.NoError(t, db.First(connection, "id = ?", connection.ID).Error)
	require.Len(t, connection.UsageLimits, 1)
	assert.Equal(t, 25.0, *connection.UsageLimits[0].Primary.UsedPercent)
	require.NotNil(t, connection.UsageObservedAt)
	require.Len(t, connection.AvailableModels, 1)
	assert.Equal(t, "gpt-5.4", connection.AvailableModels[0].Model)
	observed := *connection.UsageObservedAt
	version := connection.CredentialVersion
	for _, usage := range []any{
		nil,
		map[string]any{"limits": []any{map[string]any{"limitId": "codex", "primary": map[string]any{"usedPercent": -1}}}},
		map[string]any{"limits": []any{map[string]any{"limitId": "codex", "primary": map[string]any{"usedPercent": "invalid"}}}},
		map[string]any{"limits": []any{map[string]any{"limitId": "codex", "primary": map[string]any{}}}},
		map[string]any{"limits": []any{map[string]any{"limitId": "codex", "primary": map[string]any{"usedPercent": 25, "resetsAt": int64(253402300800)}}}},
	} {
		response := checkpoint(token, usage)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		require.NoError(t, db.First(connection, "id = ?", connection.ID).Error)
		version++
		assert.Equal(t, version, connection.CredentialVersion)
		assert.Equal(t, observed, *connection.UsageObservedAt)
		assert.Equal(t, 25.0, *connection.UsageLimits[0].Primary.UsedPercent)
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
			"limits": []any{map[string]any{"limitId": "codex", "primary": map[string]any{"usedPercent": 40, "windowDurationMins": 300, "resetsAt": time.Now().Add(time.Hour).Unix()}}},
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

func TestCodexCatalogRefreshCannotFetchCredentialsAfterOrganizationDeletion(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	connection, _, _ := readyCodexRun(t, r)
	require.NoError(t, db.Where("connection_id = ?", connection.ID).Delete(&models.FactoryCodexLease{}).Error)
	taskID := uuid.New()
	require.NoError(t, connection.BeginCatalogRefresh(db, taskID, time.Now()))
	signer := jwt.NewSigner("test-codex-catalog-refresh")
	token, err := runneraction.MintCodexSubscriptionToken(signer, runneraction.CodexSubscriptionScope{
		OrganizationID: r.Organization.ID, ConnectionID: connection.ID,
		TaskID: taskID, Generation: connection.ReservationGeneration,
	}, time.Hour)
	require.NoError(t, err)
	server := &Server{jwt: signer, encryptor: r.Encryptor}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runner/codex-subscription/credentials", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	server.handleRunnerCodexSubscriptionCredentials(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), "openai-refresh")

	require.NoError(t, models.SoftDeleteOrganizationInTransaction(db, r.Organization.ID.String()))
	response = httptest.NewRecorder()
	server.handleRunnerCodexSubscriptionCredentials(response, request)
	assert.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
	assert.NotContains(t, response.Body.String(), "openai-refresh")
}

func TestCodexRunningTaskCredentialsAfterOrganizationChange(t *testing.T) {
	for _, tc := range []struct {
		name               string
		changeOrganization func(*support.ResourceRegistry) error
		wantStatus         int
	}{
		{
			name: "factories disabled",
			changeOrganization: func(r *support.ResourceRegistry) error {
				return models.DisableExperimentalFeature(r.Organization.ID, features.FeatureFactories)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "organization deleted",
			changeOrganization: func(r *support.ResourceRegistry) error {
				return models.SoftDeleteOrganizationInTransaction(database.DB(t.Context()), r.Organization.ID.String())
			},
			wantStatus: http.StatusForbidden,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			connection, factory, secret := readyCodexRun(t, r)
			lease, err := models.FindCodexRunLeaseBySecret(db, secret)
			require.NoError(t, err)
			signer := jwt.NewSigner("test-codex-run")
			token, err := runneraction.MintCodexSubscriptionToken(signer, runneraction.CodexSubscriptionScope{
				OrganizationID: r.Organization.ID, FactoryID: factory.ID, ConnectionID: connection.ID,
				TaskID: lease.TaskID, Generation: lease.Generation,
			}, time.Hour)
			require.NoError(t, err)
			var refreshCalls int
			nextAccess := codexTestAccessToken(time.Now().Add(time.Hour))
			server := &Server{
				jwt: signer, encryptor: r.Encryptor,
				codexTokenRefresh: func(context.Context, string) (runneraction.CodexTokenSet, error) {
					refreshCalls++
					return runneraction.CodexTokenSet{AccessToken: nextAccess, RefreshToken: "openai-refresh-2", IDToken: "id-token-2"}, nil
				},
			}
			require.NoError(t, tc.changeOrganization(r))
			credentials := func() *httptest.ResponseRecorder {
				request := httptest.NewRequest(http.MethodGet, "/api/v1/runner/codex-subscription/credentials", nil)
				request.Header.Set("Authorization", "Bearer "+token)
				response := httptest.NewRecorder()
				server.handleRunnerCodexSubscriptionCredentials(response, request)
				return response
			}
			refresh := func() *httptest.ResponseRecorder {
				body, err := json.Marshal(map[string]string{"refresh_token": secret})
				require.NoError(t, err)
				request := httptest.NewRequest(http.MethodPost, "/api/v1/runner/codex-subscription/refresh", bytes.NewReader(body))
				response := httptest.NewRecorder()
				server.handleRunnerCodexSubscriptionRefresh(response, request)
				return response
			}
			response := credentials()
			assert.Equal(t, tc.wantStatus, response.Code, response.Body.String())
			if tc.wantStatus != http.StatusOK {
				assert.NotContains(t, response.Body.String(), secret)
			}
			response = refresh()
			assert.Equal(t, tc.wantStatus, response.Code, response.Body.String())
			if tc.wantStatus != http.StatusOK {
				assert.NotContains(t, response.Body.String(), nextAccess)
				assert.Zero(t, refreshCalls, "a deleted organization must not refresh provider credentials")
				return
			}
			assert.Contains(t, response.Body.String(), nextAccess)
			assert.NotContains(t, response.Body.String(), "openai-refresh-2")

			require.NoError(t, db.Model(lease).Update("revoked_at", time.Now()).Error)
			assert.Equal(t, http.StatusConflict, credentials().Code, "revocation must still stop credential access")
			assert.Equal(t, http.StatusUnauthorized, refresh().Code, "revocation must still stop token refresh")
		})
	}
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
}

func TestCodexRefreshFailureRequiresReconnect(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	connection, _, secret := readyCodexRun(t, r)
	server := &Server{
		encryptor: r.Encryptor,
		codexTokenRefresh: func(context.Context, string) (runneraction.CodexTokenSet, error) {
			return runneraction.CodexTokenSet{}, &runneraction.CodexTokenRefreshError{Permanent: true, Code: "refresh_token_reused"}
		},
	}
	body, err := json.Marshal(map[string]string{"refresh_token": secret})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runner/codex-subscription/refresh", bytes.NewReader(body))
	response := httptest.NewRecorder()
	server.handleRunnerCodexSubscriptionRefresh(response, request)
	require.Equal(t, http.StatusUnauthorized, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), "refresh_token_reused")
	require.NoError(t, db.First(connection, "id = ?", connection.ID).Error)
	assert.Equal(t, models.CodexConnectionStatusReconnectRequired, connection.Status)
	_, err = models.ReserveFactoryCodexConnection(db, connection.OrganizationID, support.CodexWorkspaceID(t, db, connection.OrganizationID), support.CreateCodexOwnedRun(t, db, connection), uuid.New(), time.Now(), r.Encryptor)
	require.ErrorIs(t, err, models.ErrSubscriptionOwnerInvalid)
}

func readyCodexRun(t *testing.T, r *support.ResourceRegistry) (*models.FactoryCodexConnection, *models.Factory, string) {
	t.Helper()
	db := database.DB(t.Context())
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
