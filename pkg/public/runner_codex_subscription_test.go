package public

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestCodexCheckpointStoresQuotaWithoutBlockingCredentials(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	connection, err := models.CreateFactoryCodexConnection(db, r.Organization.ID, factory.ID, r.User, now)
	require.NoError(t, err)
	taskID := uuid.New()
	require.NoError(t, connection.BeginLogin(db, taskID, now))
	signer := jwt.NewSigner("test-codex-usage")
	scope := runneraction.CodexSubscriptionScope{OrganizationID: r.Organization.ID, FactoryID: factory.ID, ConnectionID: connection.ID, TaskID: taskID, Generation: connection.ReservationGeneration}
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
}

func TestCodexRefreshIsSingleFlightAndHidesTheRefreshToken(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	connection, _, secret := readyCodexRun(t, r)
	var calls atomic.Int32
	nextAccess := codexTestAccessToken(time.Now().Add(time.Hour))
	server := &Server{
		encryptor: r.Encryptor,
		codexTokenRefresh: func(context.Context, string) (runneraction.CodexTokenSet, error) {
			calls.Add(1)
			return runneraction.CodexTokenSet{AccessToken: nextAccess, RefreshToken: "openai-refresh-2", IDToken: "id-token-2"}, nil
		},
	}
	refresh := func() *httptest.ResponseRecorder {
		body, err := json.Marshal(map[string]string{"grant_type": "refresh_token", "refresh_token": secret})
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/runner/codex-subscription/refresh", bytes.NewReader(body))
		response := httptest.NewRecorder()
		server.handleRunnerCodexSubscriptionRefresh(response, request)
		return response
	}
	first := refresh()
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())
	second := refresh()
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
	var lease models.FactoryCodexLease
	require.NoError(t, db.Where("connection_id = ?", connection.ID).First(&lease).Error)
	signer := jwt.NewSigner("test-codex-run")
	server.jwt = signer
	token, err := runneraction.MintCodexSubscriptionToken(signer, runneraction.CodexSubscriptionScope{
		OrganizationID: connection.OrganizationID,
		FactoryID:      connection.FactoryID,
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
	_, err = models.ReserveFactoryCodexConnection(db, connection.OrganizationID, connection.FactoryID, support.CreateCodexOwnedRun(t, db, connection), uuid.New(), time.Now(), r.Encryptor)
	require.ErrorIs(t, err, models.ErrSubscriptionOwnerInvalid)
}

func readyCodexRun(t *testing.T, r *support.ResourceRegistry) (*models.FactoryCodexConnection, *models.Factory, string) {
	t.Helper()
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	connection, err := models.CreateFactoryCodexConnection(db, r.Organization.ID, factory.ID, r.User, now)
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
	reserved, err := models.ReserveFactoryCodexConnection(db, r.Organization.ID, factory.ID, support.CreateCodexOwnedRun(t, db, connection), uuid.New(), now, r.Encryptor)
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
