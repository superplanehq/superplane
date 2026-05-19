package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/pkg/runners"
	runnermodels "github.com/superplanehq/superplane/pkg/runners/models"
	"github.com/superplanehq/superplane/test/support"
)

func liveLogGET(
	t *testing.T,
	h *Handler,
	r *support.ResourceRegistry,
	canvasID, executionID string,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/v1/canvases/%s/node-executions/%s/runner-live-logs", canvasID, executionID),
		nil,
	)
	req = mux.SetURLVars(req, map[string]string{
		"canvas_id":    canvasID,
		"execution_id": executionID,
	})
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, r.UserModel)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.LiveLogStream(rec, req)
	return rec
}

func TestHandleRunnerLiveLogStream(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	h := testHandler(t, r)

	t.Run("no user in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet,
			fmt.Sprintf("/api/v1/canvases/%s/node-executions/%s/runner-live-logs", uuid.New(), uuid.New()),
			nil,
		)
		req = mux.SetURLVars(req, map[string]string{
			"canvas_id":    uuid.New().String(),
			"execution_id": uuid.New().String(),
		})
		rec := httptest.NewRecorder()
		h.LiveLogStream(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid canvas id", func(t *testing.T) {
		rec := liveLogGET(t, h, r, "not-a-uuid", uuid.New().String())
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid canvas id")
	})

	t.Run("invalid execution id", func(t *testing.T) {
		canvasID, _ := createCanvasWithComponentExecution(t, r, "runner", "runner-1", nil)
		rec := liveLogGET(t, h, r, canvasID.String(), "bad-id")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid execution id")
	})

	t.Run("canvas not found", func(t *testing.T) {
		rec := liveLogGET(t, h, r, uuid.New().String(), uuid.New().String())
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Canvas not found")
	})

	t.Run("execution not found", func(t *testing.T) {
		canvasID, _ := createCanvasWithComponentExecution(t, r, "runner", "runner-1", nil)
		rec := liveLogGET(t, h, r, canvasID.String(), uuid.New().String())
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Execution not found")
	})

	t.Run("non-runner node", func(t *testing.T) {
		canvasID, execID := createCanvasWithComponentExecution(t, r, "noop", "noop-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: "tb-1",
		})
		rec := liveLogGET(t, h, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Runner components")
	})

	t.Run("broker task id missing", func(t *testing.T) {
		canvasID, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-1", map[string]any{})
		rec := liveLogGET(t, h, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "not available for this execution")
	})

	t.Run("stored task output without cloudwatch", func(t *testing.T) {
		canvasID, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: uuid.New().String(),
		})
		store := runners.NewPostgresStore()
		fleet, err := store.CreateFleet("live-log-fleet-"+uuid.New().String(), uuid.New().String())
		require.NoError(t, err)
		task, err := store.EnqueueJob(fleet.ID, execID, runnermodels.JobSpec{Commands: []string{"echo hi"}})
		require.NoError(t, err)
		_, err = store.CompleteJob(task.ID, runnermodels.FleetCompleteRequest{
			ExitCode: 0,
			Output:   "hello from runner\nsecond line",
		})
		require.NoError(t, err)

		rec := liveLogGET(t, h, r, canvasID.String(), execID.String())
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"type":"line"`)
		assert.Contains(t, rec.Body.String(), "hello from runner")
	})

	t.Run("cloudwatch task log starts ndjson stream", func(t *testing.T) {
		canvasID, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-1", map[string]any{
			runneraction.ExecutionMetadataTaskLog: map[string]any{
				"type": "cloudwatch",
				"cloudwatch": map[string]any{
					"log_group_name":  "/test/group",
					"log_stream_name": "task-1",
					"region":          "us-east-1",
				},
			},
		})
		rec := liveLogGET(t, h, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/x-ndjson; charset=utf-8", rec.Header().Get("Content-Type"))
		assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	})
}
