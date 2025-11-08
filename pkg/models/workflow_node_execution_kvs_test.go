package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__WorkflowNodeExecutionKV(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	steps := WorkflowNodeExecutionKVTesSteps{t: t}

	steps.CreateWorkflow()
	steps.CreateWorkflowNode()
	steps.CreateEvent()

	t.Run("CreateWorkflowNodeExecutionKVInTransaction", func(t *testing.T) {
		tx := database.Conn().Begin()
		defer tx.Rollback()

		exec := steps.CreateExecution()

		err := CreateWorkflowNodeExecutionKVInTransaction(tx, exec.WorkflowID, exec.NodeID, exec.ID, "test-key", "test-value")
		require.NoError(t, err)
	})

	t.Run("FirstNodeExecutionByKVInTransaction returns the first created execution with that key", func(t *testing.T) {
		tx := database.Conn().Begin()
		defer tx.Rollback()

		exec1 := steps.CreateExecution()
		exec2 := steps.CreateExecution()

		err := CreateWorkflowNodeExecutionKVInTransaction(tx, exec1.WorkflowID, exec1.NodeID, exec1.ID, "test-key", "test-value")
		require.NoError(t, err)

		err = CreateWorkflowNodeExecutionKVInTransaction(tx, exec2.WorkflowID, exec2.NodeID, exec2.ID, "test-key", "test-value")
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

	t.Run("FirstNodeExecutionByKVInTransaction ignores finished executions", func(t *testing.T) {
		tx := database.Conn().Begin()
		defer tx.Rollback()

		exec := steps.CreateExecution()

		err := CreateWorkflowNodeExecutionKVInTransaction(tx, exec.WorkflowID, exec.NodeID, exec.ID, "test-key", "test-value")
		require.NoError(t, err)

		// Mark execution as finished
		exec.State = WorkflowNodeExecutionStateFinished
		require.NoError(t, tx.Save(exec).Error)

		_, err = FirstNodeExecutionByKVInTransaction(tx, exec.WorkflowID, exec.NodeID, "test-key", "test-value")
		require.Error(t, err)
		require.Equal(t, gorm.ErrRecordNotFound, err)
	})
}

type WorkflowNodeExecutionKVTesSteps struct {
	t *testing.T

	wf        *Workflow
	node      *WorkflowNode
	rootEvent *WorkflowEvent
}

func (s *WorkflowNodeExecutionKVTesSteps) CreateWorkflow() {
	s.wf = &Workflow{
		OrganizationID: uuid.New(),
		Name:           "Test Workflow",
		Description:    "This is a test workflow",
	}
	require.NoError(s.t, database.Conn().Create(s.wf).Error)
}

func (s *WorkflowNodeExecutionKVTesSteps) CreateWorkflowNode() {

	s.node = &WorkflowNode{
		WorkflowID: s.wf.ID,
		NodeID:     "node-1",
	}
	require.NoError(s.t, database.Conn().Create(s.node).Error)
}

func (s *WorkflowNodeExecutionKVTesSteps) CreateEvent() {

	s.rootEvent = &WorkflowEvent{
		WorkflowID: s.wf.ID,
		NodeID:     s.node.NodeID,
		Channel:    "default",
		Data:       datatypes.JSONType[any]{},
		State:      WorkflowEventStatePending,
	}
	require.NoError(s.t, database.Conn().Create(s.rootEvent).Error)
}

func (s *WorkflowNodeExecutionKVTesSteps) CreateExecution() *WorkflowNodeExecution {
	exec := &WorkflowNodeExecution{
		WorkflowID:  s.wf.ID,
		NodeID:      s.node.NodeID,
		RootEventID: s.rootEvent.ID,
		EventID:     s.rootEvent.ID,
	}
	require.NoError(s.t, database.Conn().Create(exec).Error)
	return exec
}
