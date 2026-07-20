package e2e

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	runAppChildOnRunNodeID   = "on-run-trigger"
	runAppChildDoneNodeID    = "child-done"
	runAppChildFailNodeID    = "child-fail"
	runAppChildWaitNodeID    = "child-wait"
	runAppParentStartNodeID  = "start-trigger"
	runAppParentRunAppNodeID = "run-app"
	runAppParentOutputNodeID = "parent-output"
)

func TestRunApp(t *testing.T) {
	t.Run("happy path runs child app and parent finishes", func(t *testing.T) {
		steps := &runAppSteps{t: t}
		steps.start()
		steps.givenChildAppWithOnRunAndNoop()
		steps.givenParentAppCallingChildOnPassed(map[string]any{
			"message": "hello from parent",
		})
		steps.whenTheParentManualTriggerRuns()
		steps.thenTheChildRunFinishedWithResult(models.CanvasRunResultPassed)
		steps.thenTheParentOutputNodeFinished()
		steps.thenTheParentRunFinishedWithResult(models.CanvasRunResultPassed)
	})

	t.Run("child app failure routes parent through failed output", func(t *testing.T) {
		steps := &runAppSteps{t: t}
		steps.start()
		steps.givenChildAppWithOnRunAndFailingFilter()
		steps.givenParentAppCallingChildOnFailed(map[string]any{
			"message": "trigger child failure",
		})
		steps.whenTheParentManualTriggerRuns()
		steps.thenTheChildRunFinishedWithResult(models.CanvasRunResultFailed)
		steps.thenTheParentOutputNodeFinished()
		steps.thenTheParentRunFinishedWithResult(models.CanvasRunResultPassed)
	})

	t.Run("bad parameters fail child initialization and parent failed output", func(t *testing.T) {
		steps := &runAppSteps{t: t}
		steps.start()
		steps.givenChildAppWithRequiredOnRunParameter()
		steps.givenParentAppCallingChildOnFailed(map[string]any{})
		steps.whenTheParentManualTriggerRuns()
		steps.thenTheChildRunFinishedWithResult(models.CanvasRunResultFailed)
		steps.thenTheChildRunResultMessageContains("field 'message' is required")
		steps.thenTheParentOutputNodeFinished()
		steps.thenTheParentRunFinishedWithResult(models.CanvasRunResultPassed)
	})

	t.Run("timeout cancels child run and routes parent through failed output", func(t *testing.T) {
		steps := &runAppSteps{t: t}
		steps.start()
		steps.givenChildAppWithOnRunAndLongWait()
		steps.givenParentAppCallingChildOnFailedWithTimeout(map[string]any{}, 2)
		steps.whenTheParentManualTriggerRuns()
		steps.thenTheChildRunFinishedWithResult(models.CanvasRunResultCancelled)
		steps.thenTheParentOutputNodeFinished()
		steps.thenTheParentRunFinishedWithResult(models.CanvasRunResultPassed)
	})
}

type runAppSteps struct {
	t       *testing.T
	session *session.TestSession

	childCanvas  *models.Canvas
	parentCanvas *shared.CanvasSteps
}

func (s *runAppSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *runAppSteps) givenChildAppWithOnRunAndNoop() {
	s.childCanvas = s.createChildCanvas(childCanvasOptions{
		name: "Run App Child Success",
		onRunParameters: []any{
			map[string]any{
				"type":     "string",
				"name":     "message",
				"label":    "Message",
				"required": false,
			},
		},
		targetNode: childTargetNodeSpec{
			id:   runAppChildDoneNodeID,
			name: "Done",
			ref: models.NodeRef{
				Component: &models.ComponentRef{Name: "noop"},
			},
			configuration: map[string]any{},
		},
	})
}

func (s *runAppSteps) givenChildAppWithOnRunAndFailingFilter() {
	s.childCanvas = s.createChildCanvas(childCanvasOptions{
		name: "Run App Child Failure",
		onRunParameters: []any{
			map[string]any{
				"type":     "string",
				"name":     "message",
				"label":    "Message",
				"required": false,
			},
		},
		targetNode: childTargetNodeSpec{
			id:   runAppChildFailNodeID,
			name: "Fail Filter",
			ref: models.NodeRef{
				Component: &models.ComponentRef{Name: "filter"},
			},
			configuration: map[string]any{
				"expression": "{{ invalid syntax",
			},
		},
	})
}

func (s *runAppSteps) givenChildAppWithRequiredOnRunParameter() {
	s.childCanvas = s.createChildCanvas(childCanvasOptions{
		name: "Run App Child Params",
		onRunParameters: []any{
			map[string]any{
				"type":     "string",
				"name":     "message",
				"label":    "Message",
				"required": true,
			},
		},
		targetNode: childTargetNodeSpec{
			id:   runAppChildDoneNodeID,
			name: "Done",
			ref: models.NodeRef{
				Component: &models.ComponentRef{Name: "noop"},
			},
			configuration: map[string]any{},
		},
	})
}

func (s *runAppSteps) givenChildAppWithOnRunAndLongWait() {
	s.childCanvas = s.createChildCanvas(childCanvasOptions{
		name: "Run App Child Slow",
		onRunParameters: []any{
			map[string]any{
				"type":     "string",
				"name":     "message",
				"label":    "Message",
				"required": false,
			},
		},
		targetNode: childTargetNodeSpec{
			id:   runAppChildWaitNodeID,
			name: "Wait",
			ref: models.NodeRef{
				Component: &models.ComponentRef{Name: "wait"},
			},
			configuration: map[string]any{
				"mode":    "interval",
				"waitFor": 120,
				"unit":    "seconds",
			},
		},
	})
}

func (s *runAppSteps) givenParentAppCallingChildOnPassed(parameters map[string]any) {
	s.givenParentAppCallingChild("passed", parameters, nil)
}

func (s *runAppSteps) givenParentAppCallingChildOnFailed(parameters map[string]any) {
	s.givenParentAppCallingChild("failed", parameters, nil)
}

func (s *runAppSteps) givenParentAppCallingChildOnFailedWithTimeout(parameters map[string]any, timeoutSeconds int) {
	timeout := timeoutSeconds
	s.givenParentAppCallingChild("failed", parameters, &timeout)
}

func (s *runAppSteps) givenParentAppCallingChild(runAppOutputChannel string, parameters map[string]any, timeoutSeconds *int) {
	require.NotNil(s.t, s.childCanvas, "child canvas must exist before creating parent canvas")

	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)

	childName := s.childCanvas.Name
	parentCanvas, _ := support.CreateCanvas(s.t, s.session.OrgID, user.ID, []models.CanvasNode{
		{
			NodeID: runAppParentStartNodeID,
			Name:   "Start",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "start"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{}),
			Position:      datatypes.NewJSONType(models.Position{X: 600, Y: 200}),
		},
		{
			NodeID: runAppParentRunAppNodeID,
			Name:   "Run Child",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "runApp"},
			}),
			Configuration: datatypes.NewJSONType(func() map[string]any {
				config := map[string]any{
					"app":        s.childCanvas.ID.String(),
					"node":       runAppChildOnRunNodeID,
					"parameters": parameters,
				}
				if timeoutSeconds != nil {
					config["timeout"] = *timeoutSeconds
				}
				return config
			}()),
			Metadata: datatypes.NewJSONType(map[string]any{
				"app": map[string]any{
					"id":   s.childCanvas.ID.String(),
					"name": childName,
				},
				"node": map[string]any{
					"id":   runAppChildOnRunNodeID,
					"name": "On Run",
				},
			}),
			Position: datatypes.NewJSONType(models.Position{X: 1000, Y: 200}),
		},
		{
			NodeID: runAppParentOutputNodeID,
			Name:   "Output",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "noop"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{}),
			Position:      datatypes.NewJSONType(models.Position{X: 1400, Y: 200}),
		},
	}, []models.Edge{
		{SourceID: runAppParentStartNodeID, TargetID: runAppParentRunAppNodeID, Channel: "default"},
		{SourceID: runAppParentRunAppNodeID, TargetID: runAppParentOutputNodeID, Channel: runAppOutputChannel},
	})

	require.NoError(s.t, database.Conn().
		Model(&models.Canvas{}).
		Where("id = ?", parentCanvas.ID).
		Update("name", "Run App Parent").Error)

	s.parentCanvas = shared.NewCanvasSteps("Run App Parent", s.t, s.session)
	s.parentCanvas.WorkflowID = parentCanvas.ID
}

func (s *runAppSteps) whenTheParentManualTriggerRuns() {
	s.parentCanvas.EmitManualTrigger("Start")
}

func (s *runAppSteps) thenTheChildRunFinishedWithResult(expected string) {
	childRun := s.waitForChildRunFinished(90 * time.Second)
	require.Equal(s.t, expected, childRun.Result)
}

func (s *runAppSteps) thenTheChildRunResultMessageContains(expected string) {
	childRun := s.waitForChildRunFinished(90 * time.Second)
	require.Contains(s.t, childRun.ResultMessage, expected)
}

func (s *runAppSteps) thenTheParentOutputNodeFinished() {
	s.parentCanvas.WaitForExecution("Output", models.CanvasNodeExecutionStateFinished, 90*time.Second)
}

func (s *runAppSteps) thenTheParentRunFinishedWithResult(expected string) {
	parentRun := s.waitForParentRunFinished(90 * time.Second)
	require.Equal(s.t, expected, parentRun.Result)
}

func (s *runAppSteps) waitForParentRunFinished(timeout time.Duration) *models.CanvasRun {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		run, err := s.latestParentRun()
		if err == nil && run.State == models.CanvasRunStateFinished {
			return run
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			require.NoError(s.t, err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	run, err := s.latestParentRun()
	require.NoError(s.t, err)
	require.Equal(s.t, models.CanvasRunStateFinished, run.State)
	return run
}

func (s *runAppSteps) waitForChildRunFinished(timeout time.Duration) *models.CanvasRun {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		run, err := s.latestChildRun()
		if err == nil && run.State == models.CanvasRunStateFinished {
			return run
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			require.NoError(s.t, err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	run, err := s.latestChildRun()
	require.NoError(s.t, err)
	require.Equal(s.t, models.CanvasRunStateFinished, run.State)
	return run
}

func (s *runAppSteps) latestParentRun() (*models.CanvasRun, error) {
	var run models.CanvasRun
	err := database.Conn().
		Where("workflow_id = ?", s.parentCanvas.WorkflowID).
		Order("created_at DESC").
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (s *runAppSteps) latestChildRun() (*models.CanvasRun, error) {
	var run models.CanvasRun
	err := database.Conn().
		Where("workflow_id = ?", s.childCanvas.ID).
		Where("parent_workflow_id = ?", s.parentCanvas.WorkflowID).
		Order("created_at DESC").
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

type childTargetNodeSpec struct {
	id            string
	name          string
	ref           models.NodeRef
	configuration map[string]any
}

type childCanvasOptions struct {
	name            string
	onRunParameters []any
	targetNode      childTargetNodeSpec
}

func (s *runAppSteps) createChildCanvas(options childCanvasOptions) *models.Canvas {
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)

	canvas, _ := support.CreateCanvas(s.t, s.session.OrgID, user.ID, []models.CanvasNode{
		{
			NodeID: runAppChildOnRunNodeID,
			Name:   "On Run",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "onRun"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{
				"parameters": options.onRunParameters,
			}),
			Position: datatypes.NewJSONType(models.Position{X: 600, Y: 200}),
		},
		{
			NodeID:        options.targetNode.id,
			Name:          options.targetNode.name,
			Type:          models.NodeTypeComponent,
			Ref:           datatypes.NewJSONType(options.targetNode.ref),
			Configuration: datatypes.NewJSONType(options.targetNode.configuration),
			Position:      datatypes.NewJSONType(models.Position{X: 1000, Y: 200}),
		},
	}, []models.Edge{
		{SourceID: runAppChildOnRunNodeID, TargetID: options.targetNode.id, Channel: "default"},
	})

	require.NoError(s.t, database.Conn().
		Model(&models.Canvas{}).
		Where("id = ?", canvas.ID).
		Update("name", options.name).Error)

	canvas.Name = options.name
	return canvas
}
