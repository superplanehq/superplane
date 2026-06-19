package canvases

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func Test__ListNodeExecutionLogs(t *testing.T) {
	r := support.Setup(t)

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := ListNodeExecutionLogs(r.Organization.ID, "invalid", uuid.NewString(), "", "", 0, 0)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid execution id -> error", func(t *testing.T) {
		_, err := ListNodeExecutionLogs(r.Organization.ID, uuid.NewString(), "invalid", "", "", 0, 0)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("non-runner execution -> error", func(t *testing.T) {
		canvas, execution := createExecutionLogTestExecution(t, r, "noop")

		_, err := ListNodeExecutionLogs(r.Organization.ID, canvas.ID.String(), execution.ID.String(), "", "", 0, 0)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "runner")
	})

	t.Run("returns runner logs from broker", func(t *testing.T) {
		canvas, execution := createExecutionLogTestExecution(t, r, "runnerBash")

		// Store broker task ID in execution metadata.
		brokerTaskID := "broker-task-abc"
		err := database.Conn().Model(execution).Update(
			"metadata",
			datatypes.NewJSONType(map[string]any{"runner_broker_task_id": brokerTaskID}),
		).Error
		require.NoError(t, err)

		// Mock broker returning two NDJSON log lines.
		broker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			assert.Equal(t, fmt.Sprintf("/v1/tasks/%s/logs", brokerTaskID), req.URL.Path)
			assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
			w.Header().Set("Content-Type", "application/x-ndjson")
			fmt.Fprintln(w, `{"type":"line","text":"first"}`)
			fmt.Fprintln(w, `{"type":"line","text":"second"}`)
		}))
		defer broker.Close()

		response, err := ListNodeExecutionLogs(r.Organization.ID, canvas.ID.String(), execution.ID.String(), broker.URL, "test-token", 1, 0)
		require.NoError(t, err)
		require.Len(t, response.Logs, 1)
		assert.Equal(t, int64(1), response.Logs[0].Sequence)
		assert.Equal(t, "first", response.Logs[0].Text)
		assert.True(t, response.HasNextPage)
		assert.Equal(t, int64(1), response.LastSequence)

		response, err = ListNodeExecutionLogs(r.Organization.ID, canvas.ID.String(), execution.ID.String(), broker.URL, "test-token", 10, response.LastSequence)
		require.NoError(t, err)
		require.Len(t, response.Logs, 1)
		assert.Equal(t, int64(2), response.Logs[0].Sequence)
		assert.Equal(t, "second", response.Logs[0].Text)
		assert.False(t, response.HasNextPage)
	})
}

func createExecutionLogTestExecution(t *testing.T, r *support.ResourceRegistry, componentName string) (*models.Canvas, *models.CanvasNodeExecution) {
	t.Helper()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: "component",
			Name:   "Component",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: componentName},
			}),
		},
	}, []models.Edge{})

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "component", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "component", rootEvent.ID, rootEvent.ID, nil)
	return canvas, execution
}
