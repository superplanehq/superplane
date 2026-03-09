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
)

func TestCanvasVersioningEnforcement(t *testing.T) {
	t.Run("organization versioning enabled enforces effective canvas versioning enabled", func(t *testing.T) {
		steps := &canvasVersioningEnforcementSteps{t: t}
		steps.start()
		steps.givenACanvasExists("E2E Canvas Versioning Org On")

		steps.setCanvasVersioningInDB(false)
		steps.setOrganizationVersioningInDB(true)

		steps.visitCanvasSettings()
		steps.assertCanvasVersioningToggleChecked(true)
		steps.assertCanvasVersioningToggleDisabled()
		steps.session.AssertText("Versioning is enabled by your organization settings for all canvases.")

		// Enforcement must not require mutating every canvas row.
		steps.assertCanvasVersioningInDB(false)
	})

	t.Run("organization versioning disabled allows per-canvas on and off", func(t *testing.T) {
		steps := &canvasVersioningEnforcementSteps{t: t}
		steps.start()
		steps.givenACanvasExists("E2E Canvas Versioning Org Off")

		steps.setOrganizationVersioningInDB(false)
		steps.setCanvasVersioningInDB(false)

		steps.visitCanvasSettings()
		steps.assertCanvasVersioningToggleEnabled()
		steps.assertCanvasVersioningToggleChecked(false)
		steps.session.AssertText("This toggle controls versioning for this canvas.")

		steps.setCanvasVersioningToggle(true)
		steps.saveCanvasSettings()
		steps.assertCanvasVersioningInDB(true)

		steps.setCanvasVersioningToggle(false)
		steps.saveCanvasSettings()
		steps.assertCanvasVersioningInDB(false)
	})
}

type canvasVersioningEnforcementSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *canvasVersioningEnforcementSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *canvasVersioningEnforcementSteps) givenACanvasExists(name string) {
	s.canvas = shared.NewCanvasSteps(name, s.t, s.session)
	s.canvas.Create()
}

func (s *canvasVersioningEnforcementSteps) setOrganizationVersioningInDB(enabled bool) {
	err := database.Conn().
		Model(&models.Organization{}).
		Where("id = ?", s.session.OrgID).
		Update("canvas_versioning_enabled", enabled).
		Error
	require.NoError(s.t, err)
}

func (s *canvasVersioningEnforcementSteps) setCanvasVersioningInDB(enabled bool) {
	err := database.Conn().
		Model(&models.Canvas{}).
		Where("id = ?", s.canvas.WorkflowID).
		Update("canvas_versioning_enabled", enabled).
		Error
	require.NoError(s.t, err)
}

func (s *canvasVersioningEnforcementSteps) visitCanvasSettings() {
	s.canvas.Visit()
	s.session.AssertVisible(q.Locator(`header button:has-text("Settings")`))
	s.session.Click(q.Locator(`header button:has-text("Settings")`))
	s.session.AssertText("Canvas Name")
	s.session.AssertVisible(canvasVersioningSwitchQuery())
}

func (s *canvasVersioningEnforcementSteps) saveCanvasSettings() {
	s.session.Click(q.Locator(`button:has-text("Save Changes")`))
	s.session.AssertText("Canvas updated successfully")
}

func (s *canvasVersioningEnforcementSteps) assertCanvasVersioningToggleDisabled() {
	s.session.AssertDisabled(canvasVersioningSwitchQuery())
}

func (s *canvasVersioningEnforcementSteps) assertCanvasVersioningToggleEnabled() {
	disabled, err := canvasVersioningSwitchQuery().Run(s.session).IsDisabled()
	require.NoError(s.t, err)
	require.False(s.t, disabled)
}

func (s *canvasVersioningEnforcementSteps) assertCanvasVersioningToggleChecked(expected bool) {
	attr, err := canvasVersioningSwitchQuery().Run(s.session).GetAttribute("aria-checked")
	require.NoError(s.t, err)

	expectedString := "false"
	if expected {
		expectedString = "true"
	}

	require.Equal(s.t, expectedString, attr)
}

func (s *canvasVersioningEnforcementSteps) setCanvasVersioningToggle(enabled bool) {
	s.assertCanvasVersioningToggleEnabled()

	attr, err := canvasVersioningSwitchQuery().Run(s.session).GetAttribute("aria-checked")
	require.NoError(s.t, err)

	currentlyEnabled := attr == "true"
	if currentlyEnabled == enabled {
		return
	}

	s.session.Click(canvasVersioningSwitchQuery())
	s.assertCanvasVersioningToggleChecked(enabled)
}

func (s *canvasVersioningEnforcementSteps) assertCanvasVersioningInDB(expected bool) {
	deadline := time.Now().Add(3 * time.Second)

	for {
		canvas, err := models.FindCanvas(s.session.OrgID, s.canvas.WorkflowID)
		require.NoError(s.t, err)
		if canvas.CanvasVersioningEnabled == expected {
			return
		}

		if time.Now().After(deadline) {
			s.t.Fatalf("expected canvas_versioning_enabled=%t, got %t", expected, canvas.CanvasVersioningEnabled)
		}

		time.Sleep(200 * time.Millisecond)
	}
}

func canvasVersioningSwitchQuery() q.Query {
	return q.Locator(`button[role="switch"][aria-label="Toggle canvas versioning"]`)
}
