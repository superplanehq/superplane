package public

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestHandleRunnerLogs(t *testing.T) {
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "log-secret")
	require.NoError(t, database.TruncateTables())
	r := support.Setup(t)
	defer r.Close()

	server, _ := mustRunnerLiveLogServer(t, r)
	canvasID, executionID := createCanvasWithComponentExecution(t, r, runner.ComponentName, "runner", nil)
	require.NoError(t, models.CreateNodeExecutionKVInTransaction(database.Conn(), canvasID, "runner", executionID, "task_id", "task-1"))

	t.Run("rejects invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/runner-logs", bytes.NewBufferString(`{"task_id":"task-1","records":[]}`))
		req.Header.Set("Authorization", "Bearer wrong")
		rec := httptest.NewRecorder()

		server.Router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("persists pushed runner log records idempotently", func(t *testing.T) {
		body := `{
			"task_id":"task-1",
			"records":[
				{"seq":1,"task_id":"task-1","type":"cmd_start","index":1,"text":"echo hello"},
				{"seq":2,"task_id":"task-1","type":"line","text":"hello"},
				{"seq":3,"task_id":"task-1","type":"cmd_end","index":1,"status":"passed","duration_ms":10}
			]
		}`

		for range 2 {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/runner-logs", bytes.NewBufferString(body))
			req.Header.Set("Authorization", "Bearer log-secret")
			rec := httptest.NewRecorder()
			server.Router.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusNoContent, rec.Code)
		}

		logs, err := models.ListNodeExecutionLogs(canvasID, executionID, 10, nil)
		require.NoError(t, err)
		require.Len(t, logs, 3)
		assert.Equal(t, models.CanvasNodeExecutionLogTypeCmdStart, logs[0].Type)
		assert.Equal(t, "echo hello", *logs[0].Text)
		assert.Equal(t, models.CanvasNodeExecutionLogTypeLine, logs[1].Type)
		assert.Equal(t, "hello", *logs[1].Text)
		assert.Equal(t, models.CanvasNodeExecutionLogTypeCmdEnd, logs[2].Type)
		assert.Equal(t, "passed", *logs[2].Status)
	})
}
