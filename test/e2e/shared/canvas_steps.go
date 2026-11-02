package shared

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"

	"github.com/superplanehq/superplane/test/e2e/session"
)

type CanvasSteps struct {
	t       *testing.T
	session *session.TestSession

	CanvasName string
	WorkflowID uuid.UUID
}

func NewCanvasSteps(name string, t *testing.T, session *session.TestSession) *CanvasSteps {
	return &CanvasSteps{t: t, session: session, CanvasName: name}
}

func (s *CanvasSteps) EmitManualTrigger(name string) {
	node := s.GetNodeFromDB(name)
	context := contexts.NewEventContext(database.Conn(), node, nil, func(events []models.CanvasEvent) {
		for i := range events {
			require.NoError(s.t, messages.PublishCanvasEventCreatedMessage(&events[i]))
		}
	})

	require.NoError(s.t, context.Emit("manual.run", map[string]any{"message": "Hello, World!"}))
}

func (s *CanvasSteps) GetNodeFromDB(name string) *models.CanvasNode {
	canvas, err := models.FindCanvas(s.session.OrgID, s.WorkflowID)
	require.NoError(s.t, err)

	nodes, err := models.FindCanvasNodes(canvas.ID)
	require.NoError(s.t, err)

	nodeID := ""
	for _, n := range nodes {
		if n.Name == name {
			nodeID = n.NodeID
			break
		}
	}

	if nodeID == "" {
		s.t.Fatalf("node %s not found in database", name)
		return nil
	}

	node, err := models.FindCanvasNode(database.Conn(), s.WorkflowID, nodeID)
	require.NoError(s.t, err)

	return node
}

func (s *CanvasSteps) GetExecutionsForNode(name string) []models.CanvasNodeExecution {
	node := s.GetNodeFromDB(name)

	var executions []models.CanvasNodeExecution

	query := database.Conn().
		Where("workflow_id = ?", s.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Order("created_at DESC")

	err := query.Find(&executions).Error
	require.NoError(s.t, err)

	return executions
}

func (s *CanvasSteps) GetExecutionsForNodeInState(name string, state string) []models.CanvasNodeExecution {
	node := s.GetNodeFromDB(name)

	var executions []models.CanvasNodeExecution

	query := database.Conn().
		Where("workflow_id = ?", s.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Where("state = ?", state).
		Order("created_at DESC")

	err := query.Find(&executions).Error
	require.NoError(s.t, err)

	return executions
}

func (s *CanvasSteps) WaitForExecution(name string, state string, timeout time.Duration) {
	found := false
	start := time.Now()

	for time.Since(start) < timeout {
		executions := s.GetExecutionsForNodeInState(name, state)
		if len(executions) > 0 {
			found = true
			break
		}

		s.t.Log("waiting for execution of node", name)
		s.session.Sleep(1000)
	}

	require.True(s.t, found, "timed out waiting for execution of node %s", name)
}
