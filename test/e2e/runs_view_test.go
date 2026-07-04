package e2e

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	pw "github.com/playwright-community/playwright-go"
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
		steps.whenIVisitRunInspection()
		steps.thenTheFinishedRunIsVisible()
		steps.whenIOpenRunNodeDetails()
		steps.thenRunNodeDetailsModalIsVisible()
		steps.whenICloseRunNodeDetails()
		steps.whenIEnterEditModeFromRuns()
		steps.thenEditModeIsVisible()
	})

	t.Run("autoloads more runs from the sidebar history", func(t *testing.T) {
		steps := &runsViewSteps{t: t}
		steps.start()
		steps.givenACanvasWithManualTriggerAndNoop()
		steps.givenFinishedRuns(31)
		steps.whenIVisitRunsView()
		steps.thenRunsLoadMoreButtonIsHidden()
		steps.thenRunsSidebarRowCountIsAtLeast(25)
		steps.whenIScrollRunsSidebarToTheEnd()
		steps.thenRunsSidebarRowCountIsAtLeast(31)
	})

	t.Run("autoloads more versions and keeps them visible after selection", func(t *testing.T) {
		steps := &runsViewSteps{t: t}
		steps.start()
		steps.givenACanvasWithManualTriggerAndNoop()
		steps.givenOlderPublishedVersions(55)
		steps.whenIVisitRunsView()
		steps.whenIOpenVersionsSidebar()
		steps.thenVersionsLoadMoreButtonIsHidden()
		steps.thenVersionsSidebarRowCountIsAtLeast(50)
		steps.whenIScrollVersionsSidebarToTheEnd()
		steps.thenVersionsSidebarRowCountIsAtLeast(56)
		steps.thenVersionsSidebarIsScrolledFromTop()
		steps.whenISelectTheLastLoadedVersion()
		steps.thenVersionsSidebarRowCountIsAtLeast(56)
		steps.thenVersionsSidebarIsScrolledFromTop()
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
	s.canvas.CommitAndPublish()
}

func (s *runsViewSteps) whenTheManualTriggerRuns() {
	s.canvas.EmitManualTrigger("Start")
	s.canvas.WaitForExecution("Output", models.CanvasNodeExecutionStateFinished, 30*time.Second)
	s.run = s.waitForFinishedRun()
}

func (s *runsViewSteps) whenIVisitRunsView() {
	s.canvas.Visit()
	s.canvas.WaitForRunsSidebar()
}

func (s *runsViewSteps) whenIVisitRunInspection() {
	require.NotNil(s.t, s.run, "expected run to be created before visiting run inspection")
	s.whenIVisitRunsView()
	s.canvas.SelectRunInSidebar(s.run.ID.String())
	s.waitForRunInspectionReady()
}

func (s *runsViewSteps) waitForRunInspectionReady() {
	deadline := time.Now().Add(30 * time.Second)
	runID := s.run.ID.String()
	for time.Now().Before(deadline) {
		require.Contains(s.t, s.session.Page().URL(), "run="+runID)
		startHeader := q.TestID("node-start-header").Run(s.session)
		outputHeader := q.TestID("node-output-header").Run(s.session)
		startVisible, startErr := startHeader.IsVisible()
		outputVisible, outputErr := outputHeader.IsVisible()
		if startErr == nil && outputErr == nil && startVisible && outputVisible {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	s.session.AssertVisible(q.TestID("node-start-header"))
	s.session.AssertVisible(q.TestID("node-output-header"))
}

func (s *runsViewSteps) givenFinishedRuns(count int) {
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), s.canvas.WorkflowID)
	require.NoError(s.t, err)
	triggerID := liveVersionNodeID(s.t, liveVersion, "Start")
	now := time.Now()

	for i := 0; i < count; i++ {
		createdAt := now.Add(-time.Duration(i) * time.Minute)
		run := models.CanvasRun{
			ID:         uuid.New(),
			WorkflowID: s.canvas.WorkflowID,
			VersionID:  liveVersion.ID,
			State:      models.CanvasRunStateFinished,
			Result:     models.CanvasRunResultPassed,
			CreatedAt:  &createdAt,
			UpdatedAt:  &createdAt,
			FinishedAt: &createdAt,
		}
		require.NoError(s.t, database.Conn().Create(&run).Error)

		customName := fmt.Sprintf("Seeded run %02d", i+1)
		event := models.CanvasEvent{
			ID:         uuid.New(),
			WorkflowID: s.canvas.WorkflowID,
			NodeID:     triggerID,
			Channel:    "default",
			CustomName: &customName,
			Data:       models.NewJSONValue(map[string]any{}),
			RunID:      run.ID,
			State:      models.CanvasEventStateRouted,
			CreatedAt:  &createdAt,
		}
		require.NoError(s.t, database.Conn().Create(&event).Error)
	}
}

func (s *runsViewSteps) givenOlderPublishedVersions(count int) {
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), s.canvas.WorkflowID)
	require.NoError(s.t, err)
	now := time.Now().Add(-time.Hour)

	for i := 0; i < count; i++ {
		publishedAt := now.Add(-time.Duration(i) * time.Minute)
		version := models.CanvasVersion{
			ID:          uuid.New(),
			WorkflowID:  s.canvas.WorkflowID,
			OwnerID:     liveVersion.OwnerID,
			State:       models.CanvasVersionStatePublished,
			PublishedAt: &publishedAt,
			Nodes:       liveVersion.Nodes,
			Edges:       liveVersion.Edges,
			CreatedAt:   &publishedAt,
			UpdatedAt:   &publishedAt,
		}
		require.NoError(s.t, database.Conn().Create(&version).Error)
	}
}

func (s *runsViewSteps) thenTheFinishedRunIsVisible() {
	require.NotNil(s.t, s.run, "expected run to be created")
	s.session.AssertVisible(q.TestID("canvas-runs-sidebar"))
	s.session.AssertVisible(q.Locator(`[data-testid="canvas-view-mode-live"][aria-current="page"]`))
	s.session.AssertVisible(q.TestID("node-start-header"))
	s.session.AssertVisible(q.TestID("node-output-header"))
	s.session.AssertURLContains("run=" + s.run.ID.String())
	require.NotContains(s.t, s.session.Page().URL(), "view=runs")
	s.session.AssertText("Start")
	s.session.AssertText("Output")
	s.session.AssertText("success")
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
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		modal := q.TestID("run-node-detail-modal").Run(s.session)
		visible, err := modal.IsVisible()
		if err == nil && !visible {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	s.session.AssertHidden(q.TestID("run-node-detail-modal"))
}

func (s *runsViewSteps) whenIEnterEditModeFromRuns() {
	s.session.Click(q.TestID("runs-sidebar-live-canvas"))
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if !strings.Contains(s.session.Page().URL(), "run=") {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	require.NotContains(s.t, s.session.Page().URL(), "run=")
	s.canvas.EnterEditMode()
}

func (s *runsViewSteps) whenIOpenVersionsSidebar() {
	s.canvas.OpenVersionsSidebar()
}

func (s *runsViewSteps) thenRunsLoadMoreButtonIsHidden() {
	s.session.AssertHidden(q.Locator(`[data-testid="canvas-runs-sidebar"] button:has-text("Load more")`))
}

func (s *runsViewSteps) thenVersionsLoadMoreButtonIsHidden() {
	s.session.AssertHidden(q.Locator(`[data-testid="canvas-versions-sidebar"] button:has-text("Load older versions")`))
}

func (s *runsViewSteps) thenRunsSidebarRowCountIsAtLeast(expected int) {
	s.waitForSidebarRowCountAtLeast(s.session.Page().GetByTestId("runs-sidebar-row"), expected)
}

func (s *runsViewSteps) thenVersionsSidebarRowCountIsAtLeast(expected int) {
	s.waitForSidebarRowCountAtLeast(s.session.Page().GetByTestId("canvas-live-version-row"), expected)
}

func (s *runsViewSteps) thenVersionsSidebarIsScrolledFromTop() {
	deadline := time.Now().Add(10 * time.Second)
	scrollTop := 0.0

	for time.Now().Before(deadline) {
		scrollTop = s.sidebarScrollTop("versions-sidebar-scroll")
		if scrollTop > 0 {
			return
		}

		time.Sleep(200 * time.Millisecond)
	}

	require.Greater(s.t, scrollTop, 0.0)
}

func (s *runsViewSteps) whenIScrollRunsSidebarToTheEnd() {
	s.scrollSidebarToTheEnd("runs-sidebar-scroll")
}

func (s *runsViewSteps) whenIScrollVersionsSidebarToTheEnd() {
	s.scrollSidebarToTheEnd("versions-sidebar-scroll")
}

func (s *runsViewSteps) whenISelectTheLastLoadedVersion() {
	rows := s.session.Page().GetByTestId("canvas-live-version-row")
	count := s.waitForSidebarRowCountAtLeast(rows, 56)
	require.NoError(s.t, rows.Nth(count-1).Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
}

func (s *runsViewSteps) scrollSidebarToTheEnd(testID string) {
	scroller := s.session.Page().GetByTestId(testID)
	require.NoError(s.t, scroller.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(15000),
	}))
	require.NoError(s.t, scroller.Hover(pw.LocatorHoverOptions{Timeout: pw.Float(15000)}))
	require.NoError(s.t, s.session.Page().Mouse().Wheel(0, 5000))
	s.session.Sleep(500)
}

func (s *runsViewSteps) sidebarScrollTop(testID string) float64 {
	value, err := s.session.Page().GetByTestId(testID).Evaluate(`element => element.scrollTop`, nil)
	require.NoError(s.t, err)

	switch scrollTop := value.(type) {
	case float64:
		return scrollTop
	case int:
		return float64(scrollTop)
	default:
		s.t.Fatalf("unexpected scrollTop value %T", value)
		return 0
	}
}

func (s *runsViewSteps) waitForSidebarRowCountAtLeast(locator pw.Locator, expected int) int {
	deadline := time.Now().Add(20 * time.Second)
	lastCount := 0
	var lastErr error

	for time.Now().Before(deadline) {
		lastCount, lastErr = locator.Count()
		if lastErr == nil && lastCount >= expected {
			return lastCount
		}

		time.Sleep(250 * time.Millisecond)
	}

	require.NoError(s.t, lastErr)
	require.GreaterOrEqual(s.t, lastCount, expected)
	return lastCount
}

func (s *runsViewSteps) thenEditModeIsVisible() {
	s.canvas.WaitForEnabledExitEditButton()
	url := s.session.Page().URL()
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

func liveVersionNodeID(t *testing.T, version *models.CanvasVersion, nodeName string) string {
	t.Helper()

	for _, node := range version.Nodes {
		if node.Name == nodeName {
			return node.ID
		}
	}

	t.Fatalf("node %q not found in live canvas version", nodeName)
	return ""
}
