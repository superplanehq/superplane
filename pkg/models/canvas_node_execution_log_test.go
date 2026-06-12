package models_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__CanvasNodeExecutionLog(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: "runner",
			Name:   "Runner",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "runner"},
			}),
		},
	}, []models.Edge{})

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "runner", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "runner", rootEvent.ID, rootEvent.ID, nil)
	lineText := "hello"
	errorMessage := "boom"

	logs := []models.CanvasNodeExecutionLog{
		{
			WorkflowID:  canvas.ID,
			RunID:       execution.RunID,
			NodeID:      execution.NodeID,
			ExecutionID: execution.ID,
			Sequence:    2,
			Type:        models.CanvasNodeExecutionLogTypeError,
			Message:     &errorMessage,
		},
		{
			WorkflowID:  canvas.ID,
			RunID:       execution.RunID,
			NodeID:      execution.NodeID,
			ExecutionID: execution.ID,
			Sequence:    1,
			Type:        models.CanvasNodeExecutionLogTypeLine,
			Text:        &lineText,
		},
	}

	t.Run("lists logs in sequence order", func(t *testing.T) {
		require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
			return models.CreateNodeExecutionLogsInTransaction(tx, logs)
		}))

		found, err := models.ListNodeExecutionLogs(canvas.ID, execution.ID, 10, nil)
		require.NoError(t, err)
		require.Len(t, found, 2)
		require.Equal(t, int64(1), found[0].Sequence)
		require.Equal(t, lineText, *found[0].Text)
		require.Equal(t, int64(2), found[1].Sequence)
		require.Equal(t, errorMessage, *found[1].Message)
	})

	t.Run("ignores duplicate sequence inserts", func(t *testing.T) {
		require.NoError(t, models.CreateNodeExecutionLogs(logs))

		found, err := models.ListNodeExecutionLogs(canvas.ID, execution.ID, 10, nil)
		require.NoError(t, err)
		require.Len(t, found, 2)
	})

	t.Run("supports pagination by sequence", func(t *testing.T) {
		after := int64(1)
		found, err := models.ListNodeExecutionLogs(canvas.ID, execution.ID, 10, &after)
		require.NoError(t, err)
		require.Len(t, found, 1)
		require.Equal(t, int64(2), found[0].Sequence)
	})
}
