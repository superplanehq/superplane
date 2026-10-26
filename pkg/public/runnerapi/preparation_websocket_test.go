package runnerapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	runnerpayload "github.com/superplanehq/superplane/pkg/runners/api"
	runnercontrol "github.com/superplanehq/superplane/pkg/runners/control"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	runnerlogsfs "github.com/superplanehq/superplane/pkg/runners/logs/fs"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"go.opentelemetry.io/otel"
)

type recoveringRunnerCredentials struct {
	core.Integration
	attempts atomic.Int32
}

func (p *recoveringRunnerCredentials) ResolveSecrets(core.IntegrationSecretContext) (core.IntegrationSecrets, error) {
	if p.attempts.Add(1) == 1 {
		return core.IntegrationSecrets{}, core.NewProviderTransportError("temporary connection failure", nil)
	}
	return core.IntegrationSecrets{Values: map[string][]byte{"TOKEN": []byte("refreshed-token")}}, nil
}

func TestRunnerPreparationRetriesWithoutInterruptingHeartbeat(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	provider := &recoveringRunnerCredentials{}
	resource.Registry.Integrations["preparation-test"] = provider
	integration := models.NewIntegration(uuid.New(), resource.Organization.ID, "preparation-test", "credential source", nil)
	integration.State = models.IntegrationStateReady
	require.NoError(t, db.Create(integration).Error)
	runner, _, fleet := createRegistrationFixture(t)
	taskID := uuid.New()
	saved := runnerpayload.TaskPayload{ID: taskID.String(), Script: "echo original-prompt", ExecutionMode: "host", Environment: []runnerpayload.EnvironmentVariable{{Name: "TOKEN", Value: "saved-token"}}, CredentialRefreshes: []runnerpayload.CredentialRefresh{{Version: 1, Kind: "integration", OrganizationID: resource.Organization.ID.String(), ResourceID: integration.ID.String(), Recipient: integration.AppName, Scope: map[string]string{"key": "TOKEN", "identity": workercontexts.IntegrationCredentialIdentity(integration)}, Fields: []runnerpayload.CredentialField{{Path: []string{"environment", "0", "value"}, Original: "saved-token"}}}}}
	payload, err := json.Marshal(saved)
	require.NoError(t, err)
	task := &models.RunnerTask{ID: taskID, OrganizationID: resource.Organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued, PayloadCiphertext: payload, QueuedAt: time.Now(), CreatedAt: time.Now()}
	require.NoError(t, db.Create(task).Error)
	server, socket := connectPreparationRunner(t, resource, runner, fleet)
	var preparation models.RunnerTaskPreparation
	require.Eventually(t, func() bool {
		// GORM does not clear nullable fields when it scans into a reused value.
		// Read a fresh snapshot so a persisted NULL ends the pending attempt.
		var current models.RunnerTaskPreparation
		if err := db.First(&current, "task_id = ?", task.ID).Error; err != nil {
			return false
		}
		if current.AttemptCount != 1 || current.AttemptExpiresAt != nil {
			return false
		}
		preparation = current
		return true
	}, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, int32(1), provider.attempts.Load())
	firstDeadline := preparation.Deadline
	server.wakeConnections(runnercontrol.Notification{RunnerID: runner.ID.String()})
	var ping typedMessage
	require.NoError(t, socket.ReadJSON(&ping))
	assert.Equal(t, messageTypePing, ping.Type, "temporary preparation failures must not produce fatal protocol errors")
	require.NoError(t, socket.WriteJSON(typedMessage{Type: messageTypePong}))
	require.NoError(t, db.Model(&preparation).Update("next_retry_at", time.Now()).Error)
	var delivered taskMessage
	require.NoError(t, socket.ReadJSON(&delivered))
	require.Equal(t, messageTypeTask, delivered.Type)
	var actual runnerpayload.TaskPayload
	require.NoError(t, json.Unmarshal(delivered.Task, &actual))
	assert.Equal(t, "refreshed-token", actual.Environment[0].Value)
	assert.Equal(t, saved.Script, actual.Script)
	assert.Empty(t, actual.CredentialRefreshes)
	var prepared models.RunnerTaskPreparation
	require.NoError(t, db.First(&prepared, "task_id = ?", task.ID).Error)
	assert.Equal(t, firstDeadline, prepared.Deadline)
}

func TestInvalidOldestRunnerJobFailsAndHealthyRunnerTakesNextJob(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	runner, _, fleet := createRegistrationFixture(t)
	now := time.Now()
	invalidID := uuid.New()
	saved := runnerpayload.TaskPayload{ID: invalidID.String(), Environment: []runnerpayload.EnvironmentVariable{{Name: "TOKEN", Value: "old"}}, CredentialRefreshes: []runnerpayload.CredentialRefresh{{Version: 1, Kind: "integration", OrganizationID: resource.Organization.ID.String(), ResourceID: uuid.NewString(), Recipient: "deleted", Scope: map[string]string{"key": "TOKEN"}, Fields: []runnerpayload.CredentialField{{Path: []string{"environment", "0", "value"}, Original: "old"}}}}}
	payload, err := json.Marshal(saved)
	require.NoError(t, err)
	invalid := &models.RunnerTask{ID: invalidID, OrganizationID: resource.Organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued, PayloadCiphertext: payload, QueuedAt: now, CreatedAt: now.Add(-time.Minute)}
	require.NoError(t, db.Create(invalid).Error)
	healthy := &models.RunnerTask{ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued, QueuedAt: now, CreatedAt: now}
	healthy.PayloadCiphertext = []byte(`{"id":"` + healthy.ID.String() + `","script":"echo saved work","execution_mode":"host"}`)
	require.NoError(t, db.Create(healthy).Error)
	_, socket := connectPreparationRunner(t, resource, runner, fleet)
	var delivered taskMessage
	require.NoError(t, socket.ReadJSON(&delivered))
	require.Equal(t, messageTypeTask, delivered.Type)
	var actual runnerpayload.TaskPayload
	require.NoError(t, json.Unmarshal(delivered.Task, &actual))
	assert.Equal(t, healthy.ID.String(), actual.ID)
	failed, err := models.FindRunnerTask(db, invalid.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateFailed, failed.State)
	assert.Nil(t, failed.StartedAt)
	assert.NotNil(t, failed.FinishedAt)
}

func TestUnreportedDispatchedReservationIsLostOnReconnect(t *testing.T) {
	resource := support.Setup(t)
	db := database.DB(t.Context())
	runner, _, fleet := createRegistrationFixture(t)
	now := time.Now()
	require.NoError(t, db.Model(runner).Updates(map[string]any{"state": models.RunnerStateIdle, "registered_at": now, "last_seen_at": now}).Error)
	task := &models.RunnerTask{ID: uuid.New(), OrganizationID: resource.Organization.ID, FleetID: fleet.ID, RunnerID: &runner.ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateReserved, PayloadCiphertext: []byte(`{}`), QueuedAt: now, ReservedAt: &now, DispatchStartedAt: &now, CreatedAt: now}
	require.NoError(t, db.Create(task).Error)
	connectionID := uuid.New()
	require.NoError(t, runner.OpenConnection(db, connectionID, time.Now()))
	server, err := NewServer(jwt.NewSigner("secret"), resource.Encryptor, runnerlogs.StoreFS, nil, resource.Registry)
	require.NoError(t, err)
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		socket, err := runnerUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer socket.Close()
		connection := &runnerConnection{socket: socket}
		if err := server.reconcileRunnerConnection(r.Context(), connection, runner, connectionID, "", fleet.Slug, fleet.ScopeType, true); err != nil {
			t.Error(err)
		}
	}))
	defer httpServer.Close()
	socket, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(httpServer.URL, "http"), nil)
	require.NoError(t, err)
	if response != nil {
		defer response.Body.Close()
	}
	defer socket.Close()
	var shutdown typedMessage
	require.NoError(t, socket.ReadJSON(&shutdown))
	assert.Equal(t, messageTypeShutdown, shutdown.Type)
	lost, err := models.FindRunnerTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateLost, lost.State)
	assert.Nil(t, lost.StartedAt, "an ambiguous dispatch does not invent an execution start")
	terminated, err := models.FindRunner(db, runner.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerStateTerminated, terminated.State)
}

func TestReportedRunnerTaskDoesNotRepeatCredentialOrLogPreparation(t *testing.T) {
	fixture := createRunnerProviderFixture(t)
	db := database.DB(t.Context())
	now := time.Now()
	require.NoError(t, db.Model(fixture.task).Update("dispatch_started_at", now).Error)
	fixture.server.activeLogStoreName = "unavailable-log-store"
	previous := runnerlogs.Current()
	runnerlogs.SetCurrent(nil)
	t.Cleanup(func() { runnerlogs.SetCurrent(previous) })
	runner, err := models.FindRunner(db, fixture.runnerID)
	require.NoError(t, err)
	fleet, err := models.FindRunnerFleet(db, runner.FleetID)
	require.NoError(t, err)
	require.NoError(t, fixture.server.reconcileRunnerConnection(t.Context(), &runnerConnection{}, runner, fixture.connectionID, fixture.task.ID.String(), fleet.Slug, fleet.ScopeType, true))
	started, err := models.FindRunnerTask(db, fixture.task.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RunnerTaskStateRunning, started.State)
	assert.NotNil(t, started.StartedAt)
}

func connectPreparationRunner(t *testing.T, resource *support.ResourceRegistry, runner *models.Runner, fleet *models.RunnerFleet) (*Server, *websocket.Conn) {
	t.Helper()
	db := database.DB(t.Context())
	now := time.Now()
	require.NoError(t, db.Model(runner).Updates(map[string]any{"state": models.RunnerStateIdle, "registered_at": now, "last_seen_at": now}).Error)
	accessToken := uuid.NewString()
	require.NoError(t, db.Create(&models.RunnerCredential{RunnerID: runner.ID, AccessTokenHash: crypto.HashToken(accessToken), CreatedAt: now}).Error)
	store, err := runnerlogsfs.New(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, store.Setup(runnerlogs.SetupContext{Context: t.Context(), MeterProvider: otel.GetMeterProvider()}))
	previous := runnerlogs.Current()
	runnerlogs.SetCurrent(store)
	t.Cleanup(func() { runnerlogs.SetCurrent(previous) })
	server, err := NewServer(jwt.NewSigner("secret"), resource.Encryptor, runnerlogs.StoreFS, nil, resource.Registry)
	require.NoError(t, err)
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	socket, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(httpServer.URL, "http")+"/runner/v1/connect", http.Header{"Authorization": []string{"Bearer " + accessToken}})
	require.NoError(t, err)
	if response != nil {
		response.Body.Close()
	}
	t.Cleanup(func() { socket.Close() })
	require.NoError(t, socket.SetReadDeadline(time.Now().Add(10*time.Second)))
	require.NoError(t, socket.WriteJSON(helloMessage{Type: messageTypeHello, RunnerID: runner.ID.String(), FleetID: fleet.Slug, Version: runner.RunnerVersion}))
	return server, socket
}
