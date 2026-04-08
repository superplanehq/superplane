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
	t.Run("sidebar close button works after exiting edit mode on versioned canvas", func(t *testing.T) {
		steps := &sidebarCloseSteps{t: t}
		steps.start()
		steps.givenCanvasWithVersioningEnabled("E2E Sidebar Close")
		steps.enterEditMode()
		steps.openBuildingBlocksSidebar()
		steps.assertSidebarVisible()
		steps.exitEditMode()
		steps.assertSidebarVisible()
		steps.closeSidebarViaButton()
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

func (s *sidebarCloseSteps) givenCanvasWithVersioningEnabled(name string) {
	err := database.Conn().
		Model(&models.Organization{}).
		Where("id = ?", s.session.OrgID).
		Update("versioning_enabled", true).
		Error
	require.NoError(s.t, err)

	s.canvas = shared.NewCanvasSteps(name, s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()

	s.session.AssertVisible(q.Locator(`header button:has-text("Edit")`))
}

func (s *sidebarCloseSteps) enterEditMode() {
	editButton := q.Locator(`header button:has-text("Edit")`).Run(s.session)
	deadline := time.Now().Add(15 * time.Second)

	for {
		disabled, err := editButton.IsDisabled()
		require.NoError(s.t, err)
		if !disabled {
			break
		}

		if time.Now().After(deadline) {
			s.t.Fatalf("edit button did not become enabled")
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
	exitButton := q.Locator(`button[aria-label="Exit edit mode"]`).Run(s.session)
	require.NoError(s.t, exitButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.AssertVisible(q.Locator(`header button:has-text("Edit")`))
	s.session.Sleep(500)
}

func (s *sidebarCloseSteps) closeSidebarViaButton() {
	closeButton := q.TestID("close-sidebar-button").Run(s.session)
	require.NoError(s.t, closeButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
}
