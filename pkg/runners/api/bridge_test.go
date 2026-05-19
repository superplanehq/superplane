package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/runners"
	runnermodels "github.com/superplanehq/superplane/pkg/runners/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func testHandler(t *testing.T, r *support.ResourceRegistry) *Handler {
	t.Helper()
	return New(Config{
		BaseURL:     "http://localhost",
		Registry:    r.Registry,
		AuthService: r.AuthService,
	})
}

func fleetSyncPOST(t *testing.T, h *Handler, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(http.MethodPost, "/runner-fleets/sync", &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.FleetSync(rec, req)
	return rec
}

func fleetCompletePOST(t *testing.T, h *Handler, taskID, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(http.MethodPost, "/runner-fleets/tasks/"+taskID+"/complete", &buf)
	req = mux.SetURLVars(req, map[string]string{"taskId": taskID})
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.FleetTaskComplete(rec, req)
	return rec
}

func createCanvasWithComponentExecution(
	t *testing.T,
	r *support.ResourceRegistry,
	componentName string,
	nodeID string,
	metadata map[string]any,
) (canvasID uuid.UUID, executionID uuid.UUID) {
	t.Helper()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: "trigger-1",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "start"},
			}),
		},
		{
			NodeID: nodeID,
			Type:   models.NodeTypeComponent,
			Name:   componentName,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: componentName},
			}),
		},
	}, nil)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)

	var run *models.CanvasRun
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		run, err = models.FindOrCreateCanvasRunForRootEventInTransaction(tx, rootEvent)
		if err != nil {
			return err
		}
		return rootEvent.RoutedInTransaction(tx)
	}))

	now := time.Now()
	execution := models.CanvasNodeExecution{
		ID:            uuid.New(),
		WorkflowID:    run.WorkflowID,
		NodeID:        nodeID,
		RootEventID:   rootEvent.ID,
		RunID:         run.ID,
		EventID:       rootEvent.ID,
		State:         models.CanvasNodeExecutionStatePending,
		Configuration: datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if len(metadata) > 0 {
		execution.Metadata = datatypes.NewJSONType(metadata)
	}
	require.NoError(t, database.Conn().Create(&execution).Error)
	return canvas.ID, execution.ID
}

func TestRunnerFleetBridgeSync(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	h := testHandler(t, r)
	store := runners.NewPostgresStore()

	t.Run("unauthorized without bearer", func(t *testing.T) {
		rec := fleetSyncPOST(t, h, "", map[string]any{})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("unauthorized with bad token", func(t *testing.T) {
		rec := fleetSyncPOST(t, h, "not-a-real-token", map[string]any{})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("empty queue", func(t *testing.T) {
		fleet, err := store.CreateFleet("bridge-empty-"+uuid.New().String(), uuid.New().String(), nil)
		require.NoError(t, err)

		rec := fleetSyncPOST(t, h, fleet.AuthToken, map[string]any{})
		require.Equal(t, http.StatusOK, rec.Code)

		var resp runnermodels.FleetSyncResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.False(t, resp.Continue)
		assert.Nil(t, resp.Job)
	})

	t.Run("returns queued job", func(t *testing.T) {
		fleet, err := store.CreateFleet("bridge-jobs-"+uuid.New().String(), uuid.New().String(), nil)
		require.NoError(t, err)

		_, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-1", nil)
		_, err = store.EnqueueJob(fleet.ID, execID, runnermodels.JobSpec{
			Commands: []string{"echo hello"},
		})
		require.NoError(t, err)

		rec := fleetSyncPOST(t, h, fleet.AuthToken, map[string]any{})
		require.Equal(t, http.StatusOK, rec.Code)

		var resp runnermodels.FleetSyncResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.True(t, resp.Continue)
		require.NotNil(t, resp.Job)
		assert.Equal(t, []string{"echo hello"}, resp.Job.Spec.Commands)
	})
}

func TestRunnerFleetBridgeComplete(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	h := testHandler(t, r)
	store := runners.NewPostgresStore()

	t.Run("unauthorized", func(t *testing.T) {
		rec := fleetCompletePOST(t, h, uuid.New().String(), "", runnermodels.FleetCompleteRequest{})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("task not found", func(t *testing.T) {
		fleet, err := store.CreateFleet("bridge-complete-missing-"+uuid.New().String(), uuid.New().String(), nil)
		require.NoError(t, err)

		rec := fleetCompletePOST(t, h, uuid.New().String(), fleet.AuthToken, runnermodels.FleetCompleteRequest{ExitCode: 0})
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("wrong fleet", func(t *testing.T) {
		fleetA, err := store.CreateFleet("bridge-a-"+uuid.New().String(), uuid.New().String(), nil)
		require.NoError(t, err)
		fleetB, err := store.CreateFleet("bridge-b-"+uuid.New().String(), uuid.New().String(), nil)
		require.NoError(t, err)

		_, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-wf", nil)
		task, err := store.EnqueueJob(fleetA.ID, execID, runnermodels.JobSpec{Commands: []string{"echo"}})
		require.NoError(t, err)

		rec := fleetCompletePOST(t, h, task.ID.String(), fleetB.AuthToken, runnermodels.FleetCompleteRequest{ExitCode: 0})
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid json body", func(t *testing.T) {
		fleet, err := store.CreateFleet("bridge-bad-json-"+uuid.New().String(), uuid.New().String(), nil)
		require.NoError(t, err)
		_, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-json", nil)
		task, err := store.EnqueueJob(fleet.ID, execID, runnermodels.JobSpec{Commands: []string{"echo"}})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/runner-fleets/tasks/"+task.ID.String()+"/complete", strings.NewReader("{"))
		req = mux.SetURLVars(req, map[string]string{"taskId": task.ID.String()})
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+fleet.AuthToken)
		rec := httptest.NewRecorder()
		h.FleetTaskComplete(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("idempotent when already terminal", func(t *testing.T) {
		fleet, err := store.CreateFleet("bridge-idempotent-"+uuid.New().String(), uuid.New().String(), nil)
		require.NoError(t, err)
		_, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-idem", nil)
		task, err := store.EnqueueJob(fleet.ID, execID, runnermodels.JobSpec{Commands: []string{"echo"}})
		require.NoError(t, err)

		claimed, err := store.ClaimNextQueuedJob(fleet.ID)
		require.NoError(t, err)
		require.NotNil(t, claimed)

		_, err = store.CompleteJob(task.ID, runnermodels.FleetCompleteRequest{ExitCode: 0})
		require.NoError(t, err)

		rec := fleetCompletePOST(t, h, task.ID.String(), fleet.AuthToken, runnermodels.FleetCompleteRequest{ExitCode: 0})
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
