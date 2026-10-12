package runnerapi

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
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestRegisterRunnerConsumesGrantAndReturnsCredential(t *testing.T) {
	support.Setup(t)
	signer := jwt.NewSigner("registration-secret")
	runner, registration, fleet := createRegistrationFixture(t)
	otherRunner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStatePending,
		RunnerVersion: fleet.RunnerVersion,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	require.NoError(t, database.DB(t.Context()).Create(otherRunner).Error)
	otherRegistration := &models.RunnerRegistration{
		JTI:       uuid.New(),
		RunnerID:  otherRunner.ID,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		CreatedAt: time.Now(),
	}
	require.NoError(t, database.DB(t.Context()).Create(otherRegistration).Error)

	token, err := MintRegistrationToken(signer, runner, registration, fleet.Slug, nil)
	require.NoError(t, err)
	server, err := NewServer(signer, &crypto.NoOpEncryptor{}, runnerlogs.StoreFS)
	require.NoError(t, err)

	response := executeRegistrationRequest(t, server, token, runner.RunnerVersion)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	var body registerRunnerResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, runner.ID.String(), body.RunnerID)
	assert.Equal(t, fleet.Slug, body.FleetID)
	assert.NotEmpty(t, body.AccessToken)
	assert.True(t, body.Ephemeral)

	reloaded, err := models.FindRunner(database.DB(t.Context()), runner.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerStateIdle, reloaded.State)
	require.NotNil(t, reloaded.RegisteredAt)

	authenticated, err := models.FindRunnerByAccessTokenHash(
		database.DB(t.Context()),
		crypto.HashToken(body.AccessToken),
	)
	require.NoError(t, err)
	assert.Equal(t, runner.ID, authenticated.ID)

	reloadedOther, err := models.FindRunnerRegistration(database.DB(t.Context()), otherRunner.ID)
	require.NoError(t, err)
	assert.Nil(t, reloadedOther.ConsumedAt)

	retry := executeRegistrationRequest(t, server, token, runner.RunnerVersion)
	assert.Equal(t, http.StatusUnauthorized, retry.Code)
}

func TestRegisterRunnerRejectsVersionMismatchAndTerminatesRunner(t *testing.T) {
	support.Setup(t)
	signer := jwt.NewSigner("registration-secret")
	runner, registration, fleet := createRegistrationFixture(t)
	token, err := MintRegistrationToken(signer, runner, registration, fleet.Slug, nil)
	require.NoError(t, err)
	server, err := NewServer(signer, &crypto.NoOpEncryptor{}, runnerlogs.StoreFS)
	require.NoError(t, err)

	response := executeRegistrationRequest(t, server, token, "different-version")
	assert.Equal(t, http.StatusConflict, response.Code)

	reloaded, err := models.FindRunnerRegistration(database.DB(t.Context()), runner.ID)
	require.NoError(t, err)
	assert.Nil(t, reloaded.ConsumedAt)
	assert.NotNil(t, reloaded.RevokedAt)
	reloadedRunner, err := models.FindRunner(database.DB(t.Context()), runner.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerStateTerminated, reloadedRunner.State)
	assert.Equal(t, models.RunnerTerminationVersionMismatch, *reloadedRunner.TerminationReason)
}

func executeRegistrationRequest(
	t *testing.T,
	server *Server,
	registrationToken string,
	version string,
) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(registerRunnerRequest{Version: version})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/runner/v1/register", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+registrationToken)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	return response
}

func createRegistrationFixture(
	t *testing.T,
) (*models.Runner, *models.RunnerRegistration, *models.RunnerFleet) {
	t.Helper()
	db := database.DB(t.Context())
	now := time.Now().Truncate(time.Second)
	fleet := &models.RunnerFleet{
		ID:        uuid.New(),
		Slug:      "linux-amd64",
		ScopeType: models.RunnerFleetScopeInstallation,
		Enabled:   true,
		Spec: datatypes.NewJSONType(models.RunnerFleetSpec{
			OperatingSystem: "linux",
			Architecture:    "amd64",
		}),
		RunnerVersion: "0.1.0",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, fleet.Create(db))

	runner := &models.Runner{
		ID:            uuid.New(),
		FleetID:       fleet.ID,
		State:         models.RunnerStatePending,
		RunnerVersion: fleet.RunnerVersion,
		Ephemeral:     true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	require.NoError(t, db.Create(runner).Error)

	registration := &models.RunnerRegistration{
		JTI:       uuid.New(),
		RunnerID:  runner.ID,
		ExpiresAt: now.Add(10 * time.Minute),
		CreatedAt: now,
	}
	require.NoError(t, db.Create(registration).Error)
	return runner, registration, fleet
}
