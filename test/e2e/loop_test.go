package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestLoopComponent(t *testing.T) {
	t.Run("routes feedback into done", func(t *testing.T) {
		steps := &loopSteps{t: t}
		steps.start()
		steps.givenLoopCanvasWithBody("noop", map[string]any{}, "default")
		steps.whenManualTriggerRuns()
		steps.thenRunFinishes(models.CanvasRunResultPassed)
		steps.thenLoopExecutionFinishes(models.CanvasNodeExecutionResultPassed)
		steps.thenNodeExecutes("Done")
	})

	t.Run("fails when body cannot return feedback", func(t *testing.T) {
		steps := &loopSteps{t: t}
		steps.start()
		steps.givenLoopCanvasWithBody("filter", map[string]any{"expression": "false"}, "default")
		steps.whenManualTriggerRuns()
		steps.thenNodeExecutes("Body")
		steps.thenRunFinishes(models.CanvasRunResultFailed)
		steps.thenLoopExecutionFinishes(models.CanvasNodeExecutionResultFailed)
	})
}

type loopSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *models.Canvas
	run     *models.CanvasRun
}

func (s *loopSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *loopSteps) givenLoopCanvasWithBody(bodyComponent string, bodyConfiguration map[string]any, bodyFeedbackChannel string) {
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)

	canvas, _ := support.CreateCanvas(
		s.t,
		s.session.OrgID,
		user.ID,
		[]models.CanvasNode{
			triggerNode("start-trigger", "Start"),
			componentNode("loop", "Loop", "loop", map[string]any{
				"untilExpression": "false",
				"maxIterations":   1,
			}),
			componentNode("body", "Body", bodyComponent, bodyConfiguration),
			componentNode("done", "Done", "noop", map[string]any{}),
		},
		[]models.Edge{
			{SourceID: "start-trigger", TargetID: "loop", Channel: "default"},
			{SourceID: "loop", TargetID: "body", Channel: "next"},
			{SourceID: "body", TargetID: "loop", Channel: bodyFeedbackChannel},
			{SourceID: "loop", TargetID: "done", Channel: "done"},
		},
	)

	s.canvas = canvas
}

func triggerNode(id, name string) models.CanvasNode {
	return models.CanvasNode{
		NodeID: id,
		Name:   name,
		Type:   models.NodeTypeTrigger,
		Ref: datatypes.NewJSONType(models.NodeRef{
			Trigger: &models.TriggerRef{Name: "start"},
		}),
		Configuration: datatypes.NewJSONType(map[string]any{}),
	}
}

func componentNode(id, name, component string, configuration map[string]any) models.CanvasNode {
	return models.CanvasNode{
		NodeID: id,
		Name:   name,
		Type:   models.NodeTypeComponent,
		Ref: datatypes.NewJSONType(models.NodeRef{
			Component: &models.ComponentRef{Name: component},
		}),
		Configuration: datatypes.NewJSONType(configuration),
	}
}

func (s *loopSteps) whenManualTriggerRuns() {
	node, err := models.FindCanvasNode(database.Conn(), s.canvas.ID, "start-trigger")
	require.NoError(s.t, err)

	eventContext := contexts.NewEventContext(database.Conn(), node, func(events []models.CanvasEvent) {
		for i := range events {
			require.NoError(s.t, messages.PublishCanvasEventCreatedMessage(&events[i]))
		}
	})

	require.NoError(s.t, eventContext.Emit("manual.run", map[string]any{"source": "e2e"}))
}

func (s *loopSteps) thenRunFinishes(result string) {
	var run models.CanvasRun
	require.Eventually(s.t, func() bool {
		err := database.Conn().
			Where("workflow_id = ?", s.canvas.ID).
			Order("created_at DESC").
			First(&run).
			Error
		if err != nil {
			return false
		}

		return run.State == models.CanvasRunStateFinished && run.Result == result
	}, 30*time.Second, 300*time.Millisecond)

	s.run = &run
}

func (s *loopSteps) thenLoopExecutionFinishes(result string) {
	execution := s.waitForNodeExecution("loop", models.CanvasNodeExecutionStateFinished)
	require.Equal(s.t, result, execution.Result)
	if result == models.CanvasNodeExecutionResultFailed {
		require.Contains(s.t, execution.ResultMessage, "cannot reach the loop conclusion")
	}
}

func (s *loopSteps) thenNodeExecutes(nodeID string) {
	s.waitForNodeExecution(nodeIDFromName(nodeID), models.CanvasNodeExecutionStateFinished)
}

func (s *loopSteps) waitForNodeExecution(nodeID, state string) models.CanvasNodeExecution {
	var execution models.CanvasNodeExecution
	require.Eventually(s.t, func() bool {
		query := database.Conn().
			Where("workflow_id = ?", s.canvas.ID).
			Where("node_id = ?", nodeID).
			Order("created_at DESC")
		if s.run != nil {
			query = query.Where("run_id = ?", s.run.ID)
		}

		err := query.First(&execution).Error
		if err != nil {
			return false
		}

		return execution.State == state
	}, 30*time.Second, 300*time.Millisecond)

	return execution
}

func nodeIDFromName(name string) string {
	switch name {
	case "Body":
		return "body"
	case "Done":
		return "done"
	default:
		return name
	}
}
