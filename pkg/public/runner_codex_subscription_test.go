package public

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	scope.Generation++
	staleToken, err := runneraction.MintCodexSubscriptionToken(signer, scope, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, checkpoint(staleToken, valid).Code)
}
