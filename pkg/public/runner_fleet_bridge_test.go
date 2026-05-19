package public

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/runners"
	"github.com/superplanehq/superplane/test/support"
)

func bridgePOST(t *testing.T, server *Server, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	return rec
}

func TestRunnerFleetBridgeSync(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, _ := mustRunnerLiveLogServer(t, r)

	store := runners.NewPostgresStore()

	t.Run("unauthorized without bearer", func(t *testing.T) {
		rec := bridgePOST(t, server, "/runner-fleets/sync", "", map[string]any{})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("unauthorized with bad token", func(t *testing.T) {
		rec := bridgePOST(t, server, "/runner-fleets/sync", "not-a-real-token", map[string]any{})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("empty queue", func(t *testing.T) {
		fleet, err := store.CreateFleet("bridge-empty-"+uuid.New().String(), runners.FleetModeBridge, "", uuid.New().String(), nil)
		require.NoError(t, err)

		rec := bridgePOST(t, server, "/runner-fleets/sync", fleet.AuthToken, map[string]any{})
		require.Equal(t, http.StatusOK, rec.Code)

		var resp runners.FleetSyncResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.False(t, resp.Continue)
		assert.Nil(t, resp.Job)
	})

	t.Run("returns queued job", func(t *testing.T) {
		fleet, err := store.CreateFleet("bridge-jobs-"+uuid.New().String(), runners.FleetModeBridge, "", uuid.New().String(), nil)
		require.NoError(t, err)

		_, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-1", nil)
		_, err = store.EnqueueJob(fleet.ID, execID, runners.JobSpec{
			Commands: []string{"echo hello"},
		})
		require.NoError(t, err)

		rec := bridgePOST(t, server, "/runner-fleets/sync", fleet.AuthToken, map[string]any{})
		require.Equal(t, http.StatusOK, rec.Code)

		var resp runners.FleetSyncResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.True(t, resp.Continue)
		require.NotNil(t, resp.Job)
		assert.Equal(t, []string{"echo hello"}, resp.Job.Spec.Commands)
	})
}

func TestRunnerFleetBridgeComplete(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, _ := mustRunnerLiveLogServer(t, r)
	store := runners.NewPostgresStore()

	t.Run("unauthorized", func(t *testing.T) {
		rec := bridgePOST(t, server, "/runner-fleets/tasks/"+uuid.New().String()+"/complete", "", runners.FleetCompleteRequest{})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("task not found", func(t *testing.T) {
		fleet, err := store.CreateFleet("bridge-complete-missing-"+uuid.New().String(), runners.FleetModeBridge, "", uuid.New().String(), nil)
		require.NoError(t, err)

		rec := bridgePOST(t, server, "/runner-fleets/tasks/"+uuid.New().String()+"/complete", fleet.AuthToken, runners.FleetCompleteRequest{
			ExitCode: 0,
		})
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("wrong fleet", func(t *testing.T) {
		fleetA, err := store.CreateFleet("bridge-a-"+uuid.New().String(), runners.FleetModeBridge, "", uuid.New().String(), nil)
		require.NoError(t, err)
		fleetB, err := store.CreateFleet("bridge-b-"+uuid.New().String(), runners.FleetModeBridge, "", uuid.New().String(), nil)
		require.NoError(t, err)

		_, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-wf", nil)
		task, err := store.EnqueueJob(fleetA.ID, execID, runners.JobSpec{Commands: []string{"echo"}})
		require.NoError(t, err)

		rec := bridgePOST(t, server, "/runner-fleets/tasks/"+task.ID.String()+"/complete", fleetB.AuthToken, runners.FleetCompleteRequest{
			ExitCode: 0,
		})
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid json body", func(t *testing.T) {
		fleet, err := store.CreateFleet("bridge-bad-json-"+uuid.New().String(), runners.FleetModeBridge, "", uuid.New().String(), nil)
		require.NoError(t, err)
		_, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-json", nil)
		task, err := store.EnqueueJob(fleet.ID, execID, runners.JobSpec{Commands: []string{"echo"}})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/runner-fleets/tasks/"+task.ID.String()+"/complete", strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+fleet.AuthToken)
		rec := httptest.NewRecorder()
		server.Router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("idempotent when already terminal", func(t *testing.T) {
		fleet, err := store.CreateFleet("bridge-idempotent-"+uuid.New().String(), runners.FleetModeBridge, "", uuid.New().String(), nil)
		require.NoError(t, err)
		_, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-idem", nil)
		task, err := store.EnqueueJob(fleet.ID, execID, runners.JobSpec{Commands: []string{"echo"}})
		require.NoError(t, err)

		claimed, err := store.ClaimNextQueuedJob(fleet.ID)
		require.NoError(t, err)
		require.NotNil(t, claimed)

		_, err = store.CompleteJob(task.ID, runners.FleetCompleteRequest{ExitCode: 0})
		require.NoError(t, err)

		rec := bridgePOST(t, server, "/runner-fleets/tasks/"+task.ID.String()+"/complete", fleet.AuthToken, runners.FleetCompleteRequest{
			ExitCode: 0,
		})
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
