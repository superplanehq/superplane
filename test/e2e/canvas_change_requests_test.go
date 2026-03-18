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

func TestCanvasChangeRequests(t *testing.T) {
	t.Run("organization versioning enabled allows proposing and opening a change request", func(t *testing.T) {
		steps := &canvasChangeRequestSteps{t: t}
		steps.start()
		steps.givenCanvasWithOrganizationVersioningEnabled("E2E CR Open")
		steps.enterEditMode()
		steps.addNoopNode("Noop A", models.Position{X: 500, Y: 220})
		steps.waitForCanvasSaved()
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
		steps.waitForCanvasSaved()
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
		steps.waitForCanvasSaved()
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

	changeRequestID    string
	changeRequestTitle string
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

	s.changeRequestTitle = "Update " + name
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
	s.session.AssertVisible(q.Locator(`header button:has-text("Editing")`))
}

func (s *canvasChangeRequestSteps) addNoopNode(name string, pos models.Position) {
	s.canvas.AddNoop(name, pos)
	s.session.AssertText(name)
}

func (s *canvasChangeRequestSteps) waitForCanvasSaved() {
	deadline := time.Now().Add(8 * time.Second)

	for {
		savedVisible, savedErr := q.Locator(`header button:has-text("Saved")`).Run(s.session).IsVisible()
		require.NoError(s.t, savedErr)

		savingVisible, savingErr := q.Locator(`header button:has-text("Saving...")`).Run(s.session).IsVisible()
		require.NoError(s.t, savingErr)

		if savedVisible && !savingVisible {
			return
		}

		if time.Now().After(deadline) {
			s.t.Fatalf("expected canvas to be saved before proposing change")
		}

		time.Sleep(200 * time.Millisecond)
	}
}

func (s *canvasChangeRequestSteps) proposeChange() {
	s.session.Click(q.Locator(`header button:has-text("Editing")`))
	s.session.Click(q.Locator(`button:has-text("Propose Change")`))
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
	s.session.Click(q.Locator(`header button:has-text("Versioning")`))

	backButton := q.Locator(`button:has-text("Back to Change Requests")`).Run(s.session)
	backVisible, err := backButton.IsVisible()
	require.NoError(s.t, err)
	if backVisible {
		s.session.Click(q.Locator(`button:has-text("Back to Change Requests")`))
	}

	s.session.AssertText("Change Requests")

	s.session.Click(q.Locator(`button:has-text("` + s.changeRequestTitle + `")`))
	s.session.AssertVisible(q.Locator(`h3:has-text("` + s.changeRequestTitle + `")`))
	s.session.AssertText("Review Actions")
}

func (s *canvasChangeRequestSteps) approveChangeRequest() {
	s.session.AssertVisible(q.Locator(`aside button:has-text("Approve")`))
	s.session.Click(q.Locator(`aside button:has-text("Approve")`))
	s.session.AssertText("Change request approved")
}

func (s *canvasChangeRequestSteps) publishChangeRequest() {
	s.session.AssertVisible(q.Locator(`aside button:has-text("Publish")`))
	s.session.Click(q.Locator(`aside button:has-text("Publish")`))
}

func (s *canvasChangeRequestSteps) rejectChangeRequest() {
	s.session.AssertVisible(q.Locator(`aside button:has-text("Reject")`))
	s.session.Click(q.Locator(`aside button:has-text("Reject")`))
	s.session.AssertText("Change request rejected")
}

func (s *canvasChangeRequestSteps) reopenChangeRequest() {
	s.session.AssertVisible(q.Locator(`aside button:has-text("Reopen")`))
	s.session.Click(q.Locator(`aside button:has-text("Reopen")`))
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
