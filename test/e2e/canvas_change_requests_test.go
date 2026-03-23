package e2e

import (
	"regexp"
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

func TestCanvasChangeRequests(t *testing.T) {
	t.Run("organization versioning enabled allows proposing and opening a change request", func(t *testing.T) {
		steps := &canvasChangeRequestSteps{t: t}
		steps.start()
		steps.givenCanvasWithOrganizationVersioningEnabled("E2E CR Open")
		steps.enterEditMode()
		steps.addNoopNode("Noop A", models.Position{X: 500, Y: 220})
		steps.waitForProposeChangeReady()
		steps.proposeChange()
		steps.createChangeRequest()
		steps.openCreatedChangeRequestFromList()
		steps.assertChangeRequestStatusInDB(models.CanvasChangeRequestStatusOpen)
	})

	t.Run("approving and publishing an existing change request", func(t *testing.T) {
		steps := &canvasChangeRequestSteps{t: t}
		steps.start()
		steps.givenCanvasWithOrganizationVersioningEnabled("E2E CR Publish")
		steps.enterEditMode()
		steps.addNoopNode("Noop Publish", models.Position{X: 500, Y: 220})
		steps.waitForProposeChangeReady()
		steps.proposeChange()
		steps.createChangeRequest()
		steps.openCreatedChangeRequestFromList()
		steps.approveChangeRequest()
		steps.publishChangeRequest()
		steps.assertChangeRequestStatusInDB(models.CanvasChangeRequestStatusPublished)
	})

	t.Run("rejecting and reopening an existing change request", func(t *testing.T) {
		steps := &canvasChangeRequestSteps{t: t}
		steps.start()
		steps.givenCanvasWithOrganizationVersioningEnabled("E2E CR Reject")
		steps.enterEditMode()
		steps.addNoopNode("Noop Reject", models.Position{X: 500, Y: 220})
		steps.waitForProposeChangeReady()
		steps.proposeChange()
		steps.createChangeRequest()
		steps.openCreatedChangeRequestFromList()
		steps.rejectChangeRequest()
		steps.assertChangeRequestStatusInDB(models.CanvasChangeRequestStatusRejected)
		steps.reopenChangeRequest()
		steps.assertChangeRequestStatusInDB(models.CanvasChangeRequestStatusOpen)
	})
}

type canvasChangeRequestSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps

	changeRequestID string
}

func (s *canvasChangeRequestSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *canvasChangeRequestSteps) givenCanvasWithOrganizationVersioningEnabled(name string) {
	s.setOrganizationVersioningInDB(true)

	s.canvas = shared.NewCanvasSteps(name, s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()

	s.session.AssertVisible(q.Locator(`header button:has-text("Edit")`))
}

func (s *canvasChangeRequestSteps) setOrganizationVersioningInDB(enabled bool) {
	err := database.Conn().
		Model(&models.Organization{}).
		Where("id = ?", s.session.OrgID).
		Update("versioning_enabled", enabled).
		Error
	require.NoError(s.t, err)
}

func (s *canvasChangeRequestSteps) enterEditMode() {
	s.session.Click(q.Locator(`header button:has-text("Edit")`))
	s.session.AssertVisible(q.Locator(`header button:has-text("Propose Change")`))
}

// headerProposeChangeButton matches "Propose Change" or "Propose Change (n)" in the canvas header.
func (s *canvasChangeRequestSteps) headerProposeChangeButton() pw.Locator {
	return s.session.Page().Locator("header").GetByRole("button", pw.LocatorGetByRoleOptions{
		Name: regexp.MustCompile(`Propose Change`),
	})
}

func (s *canvasChangeRequestSteps) addNoopNode(name string, pos models.Position) {
	s.canvas.AddNoop(name, pos)
	s.session.AssertText(name)
}

func (s *canvasChangeRequestSteps) waitForProposeChangeReady() {
	deadline := time.Now().Add(8 * time.Second)
	propose := s.headerProposeChangeButton()

	for {
		disabled, err := propose.IsDisabled()
		require.NoError(s.t, err)
		if !disabled {
			return
		}

		if time.Now().After(deadline) {
			s.t.Fatalf("expected draft to be saved (Propose Change enabled) before proposing change")
		}

		time.Sleep(200 * time.Millisecond)
	}
}

func (s *canvasChangeRequestSteps) proposeChange() {
	require.NoError(s.t, s.headerProposeChangeButton().Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.AssertText("Create Change Request")
}

func (s *canvasChangeRequestSteps) createChangeRequest() {
	createButton := q.Locator(`button:has-text("Create")`).Run(s.session)
	deadline := time.Now().Add(8 * time.Second)

	for {
		disabled, err := createButton.IsDisabled()
		require.NoError(s.t, err)
		if !disabled {
			break
		}

		if time.Now().After(deadline) {
			s.t.Fatalf("create change request button did not become enabled")
		}

		time.Sleep(200 * time.Millisecond)
	}

	s.session.Click(q.Locator(`button:has-text("Create")`))
	s.session.AssertText("Change request created")

	s.assertChangeRequestStatusInDB(models.CanvasChangeRequestStatusOpen)
}

func (s *canvasChangeRequestSteps) openCreatedChangeRequestFromList() {
	// After create, the version sidebar may already be open; otherwise use the canvas control.
	openVersionControl := q.Locator(`button[aria-label="Open version control"]`).Run(s.session)
	openVisible, err := openVersionControl.IsVisible()
	require.NoError(s.t, err)
	if openVisible {
		s.session.Click(q.Locator(`button[aria-label="Open version control"]`))
	}

	s.session.AssertText("Versions")

	// Pending rows are tagged in CanvasVersionControlSidebar (data-testid) so we do not rely on
	// accessible-name collisions between pending and live "Preview v1" rows or on :has() CSS support.
	// "View details" only mounts after liveVersions[0] exists (VersionRow previousVersion); CI can need >15s.
	previewRow := s.session.Page().GetByTestId("canvas-pending-change-request-version-row")
	require.NoError(s.t, previewRow.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(30000)}))
	viewDetails := previewRow.Locator(`[aria-label="View details"]`)
	require.NoError(s.t, viewDetails.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(30000)}))
	require.NoError(s.t, viewDetails.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))

	dialogTitle := s.session.Page().Locator(`[role=dialog] [data-slot="dialog-title"]`)
	require.NoError(s.t, dialogTitle.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(15000)}))
	s.session.AssertText("Review Actions")
}

func (s *canvasChangeRequestSteps) approveChangeRequest() {
	btn := s.session.Page().Locator("[role=dialog]").GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Approve", Exact: pw.Bool(true)})
	require.NoError(s.t, btn.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(15000)}))
	require.NoError(s.t, btn.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.AssertText("Change request approved")
}

func (s *canvasChangeRequestSteps) publishChangeRequest() {
	btn := s.session.Page().Locator("[role=dialog]").GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Publish", Exact: pw.Bool(true)})
	require.NoError(s.t, btn.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(15000)}))
	require.NoError(s.t, btn.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
}

func (s *canvasChangeRequestSteps) rejectChangeRequest() {
	btn := s.session.Page().Locator("[role=dialog]").GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Reject", Exact: pw.Bool(true)})
	require.NoError(s.t, btn.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(15000)}))
	require.NoError(s.t, btn.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.AssertText("Change request rejected")
}

func (s *canvasChangeRequestSteps) reopenChangeRequest() {
	btn := s.session.Page().Locator("[role=dialog]").GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Reopen", Exact: pw.Bool(true)})
	require.NoError(s.t, btn.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(15000)}))
	require.NoError(s.t, btn.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.AssertText("Change request reopened")
}

func (s *canvasChangeRequestSteps) assertChangeRequestStatusInDB(expectedStatus string) {
	deadline := time.Now().Add(8 * time.Second)

	for {
		requests, err := models.ListCanvasChangeRequests(s.canvas.WorkflowID)
		require.NoError(s.t, err)

		if len(requests) > 0 && s.changeRequestID == "" {
			s.changeRequestID = requests[0].ID.String()
		}

		for _, request := range requests {
			if request.ID.String() != s.changeRequestID {
				continue
			}

			if request.Status == expectedStatus {
				return
			}
		}

		if time.Now().After(deadline) {
			s.t.Fatalf("expected change request status %q for request %s", expectedStatus, s.changeRequestID)
		}

		time.Sleep(200 * time.Millisecond)
	}
}
