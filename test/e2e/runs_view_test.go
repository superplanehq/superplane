package e2e

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
	"gorm.io/gorm"
)

func TestRunsView(t *testing.T) {
	t.Run("shows a finished manual-trigger run", func(t *testing.T) {
		steps := &runsViewSteps{t: t}
		steps.start()
		steps.givenACanvasWithManualTriggerAndNoop()
		steps.whenTheManualTriggerRuns()
		steps.whenIVisitRunsView()
		steps.thenTheFinishedRunIsVisible()
		steps.whenIOpenRunNodeDetails()
		steps.thenRunNodeDetailsModalIsVisible()
		steps.whenICloseRunNodeDetails()
		steps.whenIEnterEditModeFromRuns()
		steps.thenEditModeIsVisible()
	})
}

type runsViewSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
	run     *models.CanvasRun
}

func (s *runsViewSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *runsViewSteps) givenACanvasWithManualTriggerAndNoop() {
	s.canvas = shared.NewCanvasSteps("Runs View", s.t, s.session)
	s.canvas.Create()
	s.canvas.EnterEditMode()
	s.canvas.AddManualTrigger("Start", models.Position{X: 600, Y: 200})
	s.canvas.AddNoop("Output", models.Position{X: 1000, Y: 200})
	s.canvas.Connect("Start", "Output")
	s.canvas.Save()
	s.canvas.Publish()
}

func (s *runsViewSteps) whenTheManualTriggerRuns() {
	s.canvas.RunManualTrigger("Start")
	s.canvas.WaitForExecution("Output", models.CanvasNodeExecutionStateFinished, 30*time.Second)
	s.run = s.waitForFinishedRun()
}

func (s *runsViewSteps) whenIVisitRunsView() {
	s.session.Visit("/" + s.session.OrgID.String() + "/canvases/" + s.canvas.WorkflowID.String() + "?view=runs")
}

func (s *runsViewSteps) thenTheFinishedRunIsVisible() {
	require.NotNil(s.t, s.run, "expected run to be created")
	s.session.AssertVisible(q.TestID("runs-sidebar"))
	s.session.AssertVisible(q.TestID("node-start-header"))
	s.session.AssertVisible(q.TestID("node-output-header"))
	s.session.AssertURLContains("view=runs")
	s.session.AssertURLContains("run=" + s.run.ID.String())
	s.session.AssertText("Start")
	s.session.AssertText("Output")
	s.session.AssertText("SUCCESS")
	s.session.AssertHidden(q.Locator(`button[aria-label^="Add next component"]`))
}

func (s *runsViewSteps) whenIOpenRunNodeDetails() {
	s.session.Click(q.TestID("node-start-header"))
}

func (s *runsViewSteps) thenRunNodeDetailsModalIsVisible() {
	s.session.AssertVisible(q.TestID("run-node-detail-modal"))
	s.session.AssertText("Details")
}

func (s *runsViewSteps) whenICloseRunNodeDetails() {
	s.session.PressKey("Escape")
	s.session.AssertHidden(q.TestID("run-node-detail-modal"))
}

func (s *runsViewSteps) whenIEnterEditModeFromRuns() {
	s.session.Click(q.TestID("canvas-edit-button"))
}

func (s *runsViewSteps) thenEditModeIsVisible() {
	deadline := time.Now().Add(15 * time.Second)

	for time.Now().Before(deadline) {
		url := s.session.Page().URL()
		if !strings.Contains(url, "view=runs") && !strings.Contains(url, "run="+s.run.ID.String()) {
			s.session.AssertHidden(q.TestID("runs-sidebar"))
			return
		}

		time.Sleep(200 * time.Millisecond)
	}

	url := s.session.Page().URL()
	require.NotContains(s.t, url, "view=runs")
	require.NotContains(s.t, url, "run="+s.run.ID.String())
}

func (s *runsViewSteps) waitForFinishedRun() *models.CanvasRun {
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		var run models.CanvasRun
		err := database.Conn().
			Where("workflow_id = ?", s.canvas.WorkflowID).
			Order("created_at DESC").
			First(&run).
			Error

		if err == nil && run.State == models.CanvasRunStateFinished {
			return &run
		}

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			require.NoError(s.t, err)
		}

		time.Sleep(300 * time.Millisecond)
	}

	var run models.CanvasRun
	err := database.Conn().
		Where("workflow_id = ?", s.canvas.WorkflowID).
		Order("created_at DESC").
		First(&run).
		Error
	require.NoError(s.t, err)
	require.Equal(s.t, models.CanvasRunStateFinished, run.State)
	return &run
}
