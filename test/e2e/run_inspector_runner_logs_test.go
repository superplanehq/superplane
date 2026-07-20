package e2e

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

const (
	runnerLogsStartNodeID  = "start-trigger"
	runnerLogsRunnerNodeID = "run-bash"
	runnerLogsTaskID       = "task-e2e-runner-logs"
)

func TestRunInspectorRunnerLogs(t *testing.T) {
	t.Run("lazy loads runner logs when the logs accordion opens", func(t *testing.T) {
		steps := &runInspectorRunnerLogsSteps{t: t}
		steps.start()
		steps.givenRunnerLiveLogsAreAvailable()
		steps.givenAPublishedCanvasWithRunner()
		steps.givenAFinishedRunnerRun(runnerLogsTaskID)
		steps.whenIVisitRunInspection()
		steps.whenIOpenTheRunnerNode()
		steps.thenTheLogsAccordionIsVisible()
		steps.thenLiveLogsWereNotRequested()
		steps.whenIOpenTheLogsAccordion()
		steps.thenLiveLogsAreRequested()
		steps.thenRunnerLogLinesAreVisible()
	})

	t.Run("hides runner logs before the runner has started", func(t *testing.T) {
		steps := &runInspectorRunnerLogsSteps{t: t}
		steps.start()
		steps.givenRunnerLiveLogsAreAvailable()
		steps.givenAPublishedCanvasWithRunner()
		steps.givenAFinishedRunnerRun("")
		steps.whenIVisitRunInspection()
		steps.whenIOpenTheRunnerNode()
		steps.thenTheLogsAccordionIsHidden()
		steps.thenLiveLogsWereNotRequested()
	})
}

type runInspectorRunnerLogsSteps struct {
	t                      *testing.T
	session                *session.TestSession
	canvas                 *shared.CanvasSteps
	run                    *models.CanvasRun
	liveLogSessionRequests atomic.Int32
}

func (s *runInspectorRunnerLogsSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *runInspectorRunnerLogsSteps) givenRunnerLiveLogsAreAvailable() {
	require.NoError(s.t, s.session.Page().Route("**/runner-live-logs/session", func(route pw.Route) {
		s.liveLogSessionRequests.Add(1)
		require.NoError(s.t, route.Fulfill(pw.RouteFulfillOptions{
			Status:      pw.Int(200),
			ContentType: pw.String("application/json"),
			Body: fmt.Sprintf(
				`{"stream_url":"/e2e-runner-live-logs/%s","token":"e2e-token","expires_at":"%s"}`,
				runnerLogsTaskID,
				time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
			),
		}))
	}))

	require.NoError(s.t, s.session.Page().Route("**/e2e-runner-live-logs/**", func(route pw.Route) {
		require.NoError(s.t, route.Fulfill(pw.RouteFulfillOptions{
			Status:      pw.Int(200),
			ContentType: pw.String("application/x-ndjson"),
			Body: "" +
				`{"type":"cmd_start","index":0,"text":"npm run build","started_at":1710000000000}` + "\n" +
				`{"type":"line","text":"> build"}` + "\n" +
				`{"type":"line","text":"vite build"}` + "\n" +
				`{"type":"cmd_end","index":0,"status":"passed","duration_ms":4200}` + "\n",
		}))
	}))
}

func (s *runInspectorRunnerLogsSteps) givenAPublishedCanvasWithRunner() {
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)

	canvas, _ := support.CreateCanvas(s.t, s.session.OrgID, user.ID, []models.CanvasNode{
		{
			NodeID: runnerLogsStartNodeID,
			Name:   "Start",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "start"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{}),
			Position:      datatypes.NewJSONType(models.Position{X: 600, Y: 200}),
		},
		{
			NodeID: runnerLogsRunnerNodeID,
			Name:   "Run Bash",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "runnerBash"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{
				"script": "npm run build",
			}),
			Position: datatypes.NewJSONType(models.Position{X: 1000, Y: 200}),
		},
	}, []models.Edge{
		{SourceID: runnerLogsStartNodeID, TargetID: runnerLogsRunnerNodeID, Channel: "default"},
	})

	s.canvas = shared.NewCanvasSteps("Run Inspector Runner Logs", s.t, s.session)
	s.canvas.WorkflowID = canvas.ID
	require.NoError(s.t, database.Conn().
		Model(&models.Canvas{}).
		Where("id = ?", s.canvas.WorkflowID).
		Update("name", s.canvas.CanvasName).
		Error)
}

func (s *runInspectorRunnerLogsSteps) givenAFinishedRunnerRun(taskID string) {
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), s.canvas.WorkflowID)
	require.NoError(s.t, err)

	createdAt := time.Now().Add(-time.Minute)
	finishedAt := time.Now()
	run := &models.CanvasRun{
		ID:         uuid.New(),
		WorkflowID: s.canvas.WorkflowID,
		NodeID:     runnerLogsStartNodeID,
		VersionID:  liveVersion.ID,
		State:      models.CanvasRunStateFinished,
		Result:     models.CanvasRunResultPassed,
		CreatedAt:  &createdAt,
		UpdatedAt:  &finishedAt,
		FinishedAt: &finishedAt,
	}
	require.NoError(s.t, database.Conn().Create(run).Error)

	customName := "Runner log E2E run"
	rootEvent := models.CanvasEvent{
		ID:         uuid.New(),
		WorkflowID: s.canvas.WorkflowID,
		NodeID:     runnerLogsStartNodeID,
		Channel:    "default",
		CustomName: &customName,
		Data:       models.NewJSONValue(map[string]any{"message": "run build"}),
		RunID:      run.ID,
		State:      models.CanvasEventStateRouted,
		CreatedAt:  &createdAt,
	}
	require.NoError(s.t, database.Conn().Create(&rootEvent).Error)

	metadata := map[string]any{}
	if taskID != "" {
		metadata["runner_broker_task_id"] = taskID
	}

	execution := models.CanvasNodeExecution{
		ID:            uuid.New(),
		WorkflowID:    s.canvas.WorkflowID,
		NodeID:        runnerLogsRunnerNodeID,
		RootEventID:   rootEvent.ID,
		RunID:         run.ID,
		EventID:       rootEvent.ID,
		State:         models.CanvasNodeExecutionStateFinished,
		Result:        models.CanvasNodeExecutionResultPassed,
		ResultReason:  models.CanvasNodeExecutionResultReasonOk,
		Metadata:      datatypes.NewJSONType(metadata),
		Configuration: datatypes.NewJSONType(map[string]any{"script": "npm run build"}),
		CreatedAt:     &createdAt,
		UpdatedAt:     &finishedAt,
	}
	require.NoError(s.t, database.Conn().Create(&execution).Error)

	outputEvent := models.CanvasEvent{
		ID:          uuid.New(),
		WorkflowID:  s.canvas.WorkflowID,
		NodeID:      runnerLogsRunnerNodeID,
		Channel:     "passed",
		Data:        models.NewJSONValue(map[string]any{"status": "succeeded", "exit_code": 0}),
		ExecutionID: &execution.ID,
		RunID:       run.ID,
		State:       models.CanvasEventStateRouted,
		CreatedAt:   &finishedAt,
	}
	require.NoError(s.t, database.Conn().Create(&outputEvent).Error)

	s.run = run
}

func (s *runInspectorRunnerLogsSteps) whenIVisitRunInspection() {
	require.NotNil(s.t, s.run, "expected run to be seeded before visiting run inspection")
	s.canvas.Visit()
	s.canvas.WaitForRunsSidebar()
	s.canvas.SelectRunInSidebar(s.run.ID.String())
	s.waitForRunnerInspectionReady()
}

func (s *runInspectorRunnerLogsSteps) waitForRunnerInspectionReady() {
	s.session.AssertURLContains("run=" + s.run.ID.String())
	s.session.AssertVisible(q.TestID("node-start-header"))
	s.session.AssertVisible(q.TestID("node-run-bash-header"))
}

func (s *runInspectorRunnerLogsSteps) whenIOpenTheRunnerNode() {
	s.session.Click(q.TestID("node-run-bash-header"))
	s.session.AssertVisible(q.TestID("run-inspector-panel"))
}

func (s *runInspectorRunnerLogsSteps) thenTheLogsAccordionIsVisible() {
	s.session.AssertVisible(q.TestID("run-inspector-logs-accordion"))
	s.session.AssertVisible(q.TestID("run-inspector-runtime-accordion"))
	s.session.AssertVisible(q.TestID("run-inspector-output-accordion"))
}

func (s *runInspectorRunnerLogsSteps) thenTheLogsAccordionIsHidden() {
	s.session.AssertHidden(q.TestID("run-inspector-logs-accordion"))
}

func (s *runInspectorRunnerLogsSteps) thenLiveLogsWereNotRequested() {
	require.Equal(s.t, int32(0), s.liveLogSessionRequests.Load())
}

func (s *runInspectorRunnerLogsSteps) whenIOpenTheLogsAccordion() {
	s.session.Click(q.TestID("run-inspector-logs-accordion-trigger"))
}

func (s *runInspectorRunnerLogsSteps) thenLiveLogsAreRequested() {
	require.Eventually(s.t, func() bool {
		return s.liveLogSessionRequests.Load() > 0
	}, 10*time.Second, 200*time.Millisecond)
}

func (s *runInspectorRunnerLogsSteps) thenRunnerLogLinesAreVisible() {
	terminal := s.session.Page().GetByTestId("run-inspector-runner-logs-terminal")
	require.NoError(s.t, terminal.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(15000),
	}))
	text, err := terminal.InnerText()
	require.NoError(s.t, err)
	require.Contains(s.t, text, "npm run build")
	require.Contains(s.t, text, "vite build")
}
