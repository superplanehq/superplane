package canvases

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func Test__ListNodeExecutionLogs(t *testing.T) {
	r := support.Setup(t)

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		_, err := ListNodeExecutionLogs(r.Organization.ID, "invalid", uuid.NewString(), 0, 0)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid execution id -> error", func(t *testing.T) {
		_, err := ListNodeExecutionLogs(r.Organization.ID, uuid.NewString(), "invalid", 0, 0)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("non-runner execution -> error", func(t *testing.T) {
		canvas, execution := createExecutionLogTestExecution(t, r, "noop")

		_, err := ListNodeExecutionLogs(r.Organization.ID, canvas.ID.String(), execution.ID.String(), 0, 0)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "runner")
	})

	t.Run("returns runner logs in order", func(t *testing.T) {
		canvas, execution := createExecutionLogTestExecution(t, r, "runnerBash")
		secondLine := "second"
		firstLine := "first"
		require.NoError(t, models.CreateNodeExecutionLogs([]models.CanvasNodeExecutionLog{
			{
				WorkflowID:  canvas.ID,
				RunID:       execution.RunID,
				NodeID:      execution.NodeID,
				ExecutionID: execution.ID,
				Sequence:    2,
				Type:        models.CanvasNodeExecutionLogTypeLine,
				Text:        &secondLine,
			},
			{
				WorkflowID:  canvas.ID,
				RunID:       execution.RunID,
				NodeID:      execution.NodeID,
				ExecutionID: execution.ID,
				Sequence:    1,
				Type:        models.CanvasNodeExecutionLogTypeLine,
				Text:        &firstLine,
			},
		}))

		response, err := ListNodeExecutionLogs(r.Organization.ID, canvas.ID.String(), execution.ID.String(), 1, 0)
		require.NoError(t, err)
		require.Len(t, response.Logs, 1)
		assert.Equal(t, int64(1), response.Logs[0].Sequence)
		assert.Equal(t, firstLine, response.Logs[0].Text)
		assert.True(t, response.HasNextPage)
		assert.Equal(t, int64(1), response.LastSequence)

		response, err = ListNodeExecutionLogs(r.Organization.ID, canvas.ID.String(), execution.ID.String(), 10, response.LastSequence)
		require.NoError(t, err)
		require.Len(t, response.Logs, 1)
		assert.Equal(t, int64(2), response.Logs[0].Sequence)
		assert.Equal(t, secondLine, response.Logs[0].Text)
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
