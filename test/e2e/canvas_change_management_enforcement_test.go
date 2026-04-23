package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
	"github.com/superplanehq/superplane/test/support"
)

func TestCanvasChangeManagementEnforcement(t *testing.T) {
	t.Run("organization change management enabled enforces effective canvas change management enabled", func(t *testing.T) {
		steps := &canvasChangeManagementEnforcementSteps{t: t}
		steps.start()
		steps.givenACanvasExists("E2E Canvas Change Mgmt Org On")

		steps.setCanvasChangeManagementInDB(false)
		steps.setOrganizationChangeManagementInDB(true)

		steps.visitCanvasSettings()
		steps.assertCanvasChangeManagementToggleChecked(true)
		steps.assertCanvasChangeManagementToggleDisabled()
		steps.session.AssertText("Change management is enabled by your organization settings for all canvases.")

		// Enforcement must not require mutating every canvas row.
		steps.assertCanvasChangeManagementInDB(false)
	})

	t.Run("organization change management disabled allows enabling per-canvas and blocks direct live edits once enabled", func(t *testing.T) {
		steps := &canvasChangeManagementEnforcementSteps{t: t}
		steps.start()
		steps.givenACanvasExists("E2E Canvas Change Mgmt Org Off")

		steps.setOrganizationChangeManagementInDB(false)
		steps.setCanvasChangeManagementInDB(false)

		steps.visitCanvasSettings()
		steps.assertCanvasChangeManagementToggleEnabled()
		steps.assertCanvasChangeManagementToggleChecked(false)
		steps.session.AssertText("This toggle controls change management for this canvas.")

		steps.setCanvasChangeManagementToggle(true)
		steps.saveCanvasSettings()
		steps.assertCanvasChangeManagementInDB(true)

		steps.visitCanvasSettings()
		steps.assertCanvasChangeManagementToggleEnabled()
		steps.setCanvasChangeManagementToggle(false)
		steps.saveCanvasSettingsExpectError("change management is enabled for this canvas")
		steps.assertCanvasChangeManagementInDB(true)
	})
}

type canvasChangeManagementEnforcementSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *canvasChangeManagementEnforcementSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *canvasChangeManagementEnforcementSteps) givenACanvasExists(name string) {
	s.canvas = shared.NewCanvasSteps(name, s.t, s.session)
	s.canvas.Create()
}

func (s *canvasChangeManagementEnforcementSteps) setOrganizationChangeManagementInDB(enabled bool) {
	err := database.Conn().
		Model(&models.Organization{}).
		Where("id = ?", s.session.OrgID).
		Update("change_management_enabled", enabled).
		Error
	require.NoError(s.t, err)
}

func (s *canvasChangeManagementEnforcementSteps) setCanvasChangeManagementInDB(enabled bool) {
	support.SetCanvasChangeManagementEnabled(s.t, s.canvas.WorkflowID, enabled)
}

func (s *canvasChangeManagementEnforcementSteps) visitCanvasSettings() {
	s.canvas.Visit()
	s.session.AssertVisible(q.Locator(`header button[aria-label="Canvas menu"]`))
	s.session.Click(q.Locator(`header button[aria-label="Canvas menu"]`))
	s.session.AssertVisible(q.Locator(`[role="menuitem"]:has-text("Settings")`))
	s.session.Click(q.Locator(`[role="menuitem"]:has-text("Settings")`))
	s.session.AssertText("Canvas Name")
	s.session.AssertVisible(canvasChangeManagementSwitchQuery())
}

func (s *canvasChangeManagementEnforcementSteps) saveCanvasSettings() {
	s.session.Click(q.TestID("canvas-settings-save-changes"))
	canvasPath := "/" + s.session.OrgID.String() + "/canvases/" + s.canvas.WorkflowID.String()
	s.session.WaitForBrowserPath(canvasPath)
}

func (s *canvasChangeManagementEnforcementSteps) saveCanvasSettingsExpectError(message string) {
	s.session.Click(q.TestID("canvas-settings-save-changes"))
	s.session.AssertURLContains("/settings")
	s.session.AssertText(message)
}

func (s *canvasChangeManagementEnforcementSteps) assertCanvasChangeManagementToggleDisabled() {
	s.session.AssertDisabled(canvasChangeManagementSwitchQuery())
}

func (s *canvasChangeManagementEnforcementSteps) assertCanvasChangeManagementToggleEnabled() {
	disabled, err := canvasChangeManagementSwitchQuery().Run(s.session).IsDisabled()
	require.NoError(s.t, err)
	require.False(s.t, disabled)
}

func (s *canvasChangeManagementEnforcementSteps) assertCanvasChangeManagementToggleChecked(expected bool) {
	attr, err := canvasChangeManagementSwitchQuery().Run(s.session).GetAttribute("aria-checked")
	require.NoError(s.t, err)

	expectedString := "false"
	if expected {
		expectedString = "true"
	}

	require.Equal(s.t, expectedString, attr)
}

func (s *canvasChangeManagementEnforcementSteps) setCanvasChangeManagementToggle(enabled bool) {
	s.assertCanvasChangeManagementToggleEnabled()

	attr, err := canvasChangeManagementSwitchQuery().Run(s.session).GetAttribute("aria-checked")
	require.NoError(s.t, err)

	currentlyEnabled := attr == "true"
	if currentlyEnabled == enabled {
		return
	}

	s.session.Click(canvasChangeManagementSwitchQuery())
	s.assertCanvasChangeManagementToggleChecked(enabled)
}

func (s *canvasChangeManagementEnforcementSteps) assertCanvasChangeManagementInDB(expected bool) {
	deadline := time.Now().Add(3 * time.Second)

	for {
		canvas, err := models.FindCanvas(s.session.OrgID, s.canvas.WorkflowID)
		require.NoError(s.t, err)
		if canvas.ChangeManagementEnabled == expected {
			return
		}

		if time.Now().After(deadline) {
			s.t.Fatalf("expected change_management_enabled=%t, got %t", expected, canvas.ChangeManagementEnabled)
		}

		time.Sleep(200 * time.Millisecond)
	}
}

func canvasChangeManagementSwitchQuery() q.Query {
	return q.Locator(`button[role="switch"][aria-label="Toggle canvas change management"]`)
}
