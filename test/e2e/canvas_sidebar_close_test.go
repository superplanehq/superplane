package e2e

import (
	"testing"
	"time"

	pw "github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestCanvasSidebarClose(t *testing.T) {
	t.Run("building blocks sidebar is not shown after exiting edit mode on versioned canvas", func(t *testing.T) {
		steps := &sidebarCloseSteps{t: t}
		steps.start()
		steps.givenCanvasWithChangeManagementEnabled("E2E Sidebar Close")
		steps.enterEditMode()
		steps.openBuildingBlocksSidebar()
		steps.assertSidebarVisible()
		steps.exitEditMode()
		steps.assertSidebarHidden()
	})
}

type sidebarCloseSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *sidebarCloseSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *sidebarCloseSteps) givenCanvasWithChangeManagementEnabled(name string) {
	err := database.Conn().
		Model(&models.Organization{}).
		Where("id = ?", s.session.OrgID).
		Update("change_management_enabled", true).
		Error
	require.NoError(s.t, err)

	s.canvas = shared.NewCanvasSteps(name, s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()

	s.session.AssertVisible(q.TestID("canvas-edit-button"))
}

func (s *sidebarCloseSteps) enterEditMode() {
	editButton := q.TestID("canvas-edit-button").Run(s.session)
	deadline := time.Now().Add(15 * time.Second)

	for {
		disabled, err := editButton.IsDisabled()
		require.NoError(s.t, err)
		if !disabled {
			break
		}

		if time.Now().After(deadline) {
			s.t.Fatalf("editor control did not become enabled")
		}

		time.Sleep(200 * time.Millisecond)
	}

	require.NoError(s.t, editButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.AssertVisible(q.Locator(`header button:has-text("Propose Change")`))
}

func (s *sidebarCloseSteps) openBuildingBlocksSidebar() {
	s.canvas.OpenBuildingBlocksSidebar()
}

func (s *sidebarCloseSteps) assertSidebarVisible() {
	s.session.AssertVisible(q.TestID("building-blocks-sidebar"))
}

func (s *sidebarCloseSteps) assertSidebarHidden() {
	s.session.AssertHidden(q.TestID("building-blocks-sidebar"))
}

func (s *sidebarCloseSteps) exitEditMode() {
	exitButton := q.TestID("canvas-exit-edit-button").Run(s.session)
	deadline := time.Now().Add(15 * time.Second)
	for {
		disabled, err := exitButton.IsDisabled()
		require.NoError(s.t, err)
		if !disabled {
			break
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("exit edit control did not become enabled")
		}
		time.Sleep(200 * time.Millisecond)
	}
	require.NoError(s.t, exitButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.AssertVisible(q.TestID("canvas-edit-button"))
	s.session.Sleep(500)
}
