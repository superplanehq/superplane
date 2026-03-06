package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__CanvasNodeExecutionKV(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	steps := CanvasNodeExecutionKVTestSteps{t: t}

	steps.CreateCanvas()
	steps.CreateCanvasNode()
	steps.CreateEvent()

	t.Run("CreateNodeExecutionKVInTransaction", func(t *testing.T) {
		tx := database.Conn().Begin()
		defer tx.Rollback()

		exec := steps.CreateExecution()

		err := CreateNodeExecutionKVInTransaction(tx, exec.WorkflowID, exec.NodeID, exec.ID, "test-key", "test-value")
		require.NoError(t, err)
	})

	t.Run("FirstNodeExecutionByKVInTransaction returns the first created execution with that key", func(t *testing.T) {
		tx := database.Conn().Begin()
		defer tx.Rollback()

		exec1 := steps.CreateExecution()
		exec2 := steps.CreateExecution()

		err := CreateNodeExecutionKVInTransaction(tx, exec1.WorkflowID, exec1.NodeID, exec1.ID, "test-key", "test-value")
		require.NoError(t, err)

		err = CreateNodeExecutionKVInTransaction(tx, exec2.WorkflowID, exec2.NodeID, exec2.ID, "test-key", "test-value")
		require.NoError(t, err)

		foundExec, err := FirstNodeExecutionByKVInTransaction(tx, exec1.WorkflowID, exec1.NodeID, "test-key", "test-value")
		require.NoError(t, err)
		require.Equal(t, exec1.ID, foundExec.ID)
	})

	t.Run("FirstNodeExecutionByKVInTransaction returns error if not found", func(t *testing.T) {
		tx := database.Conn().Begin()
		defer tx.Rollback()

		_, err := FirstNodeExecutionByKVInTransaction(tx, uuid.New(), "non-existent-node", "non-existent-key", "non-existent-value")
		require.Error(t, err)
		require.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("FirstNodeExecutionByKVInTransaction includes finished executions", func(t *testing.T) {
		tx := database.Conn().Begin()
		defer tx.Rollback()

		exec := steps.CreateExecution()

		err := CreateNodeExecutionKVInTransaction(tx, exec.WorkflowID, exec.NodeID, exec.ID, "test-key", "test-value")
		require.NoError(t, err)

		// Mark execution as finished
		exec.State = CanvasNodeExecutionStateFinished
		require.NoError(t, tx.Save(exec).Error)

		_, err = FirstNodeExecutionByKVInTransaction(tx, exec.WorkflowID, exec.NodeID, "test-key", "test-value")
		require.NoError(t, err)
	})
}

type CanvasNodeExecutionKVTestSteps struct {
	t *testing.T

	wf        *Canvas
	node      *CanvasNode
	rootEvent *CanvasEvent
}

func (s *CanvasNodeExecutionKVTestSteps) CreateCanvas() {
	now := time.Now()
	liveVersionID := uuid.New()
	wf := &Canvas{
		OrganizationID: uuid.New(),
		LiveVersionID:  &liveVersionID,
		Name:           "Test Canvas",
		Description:    "This is a test workflow",
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(s.t, database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(wf).Error; err != nil {
			return err
		}
		return tx.Create(&CanvasVersion{
			ID:          liveVersionID,
			WorkflowID:  wf.ID,
			IsPublished: true,
			PublishedAt: &now,
			Nodes:       datatypes.NewJSONSlice([]Node{}),
			Edges:       datatypes.NewJSONSlice([]Edge{}),
			CreatedAt:   &now,
			UpdatedAt:   &now,
		}).Error
	}))
	s.wf = wf
}

func (s *CanvasNodeExecutionKVTestSteps) CreateCanvasNode() {

	s.node = &CanvasNode{
		WorkflowID: s.wf.ID,
		NodeID:     "node-1",
	}
	require.NoError(s.t, database.Conn().Create(s.node).Error)
}

func (s *CanvasNodeExecutionKVTestSteps) CreateEvent() {

	s.rootEvent = &CanvasEvent{
		WorkflowID: s.wf.ID,
		NodeID:     s.node.NodeID,
		Channel:    "default",
		Data:       datatypes.JSONType[any]{},
		State:      CanvasEventStatePending,
	}
	require.NoError(s.t, database.Conn().Create(s.rootEvent).Error)
}

func (s *CanvasNodeExecutionKVTestSteps) CreateExecution() *CanvasNodeExecution {
	exec := &CanvasNodeExecution{
		WorkflowID:  s.wf.ID,
		NodeID:      s.node.NodeID,
		RootEventID: s.rootEvent.ID,
		EventID:     s.rootEvent.ID,
	}
	require.NoError(s.t, database.Conn().Create(exec).Error)
	return exec
}
