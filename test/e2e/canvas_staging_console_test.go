package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestCanvasStagingConsoleEdit(t *testing.T) {
	t.Run("console edits persist after warm staging cache", func(t *testing.T) {
		steps := &canvasStagingConsoleSteps{t: t}
		panelName := "Staging Notes"
		bodyText := fmt.Sprintf("E2E staged console body %s", uuid.NewString())

		steps.start()
		steps.givenACanvasOnLiveView()
		steps.whenIWarmLiveViewStagingCaches()
		steps.whenIEnterEditMode()
		steps.whenIOpenConsoleAndAddMarkdownPanel(panelName)
		steps.whenIEditMarkdownBody(bodyText)
		steps.thenConsoleShowsBody(bodyText)
		steps.thenConsoleEditIsStaged(bodyText)
	})
}

type canvasStagingConsoleSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *canvasStagingConsoleSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *canvasStagingConsoleSteps) givenACanvasOnLiveView() {
	s.canvas = shared.NewCanvasSteps("E2E Staging Console "+uuid.NewString(), s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()
	s.session.AssertVisible(q.TestID("canvas-edit-button"))
}

func (s *canvasStagingConsoleSteps) whenIWarmLiveViewStagingCaches() {
	s.canvas.WarmLiveViewStagingCaches()
}

func (s *canvasStagingConsoleSteps) whenIEnterEditMode() {
	s.canvas.EnterEditMode()
}

func (s *canvasStagingConsoleSteps) whenIOpenConsoleAndAddMarkdownPanel(panelName string) {
	s.canvas.SwitchToConsoleView()
	s.canvas.AddMarkdownConsolePanel(panelName)
}

func (s *canvasStagingConsoleSteps) whenIEditMarkdownBody(body string) {
	s.canvas.EditFirstMarkdownConsoleBody(body)
}

func (s *canvasStagingConsoleSteps) thenConsoleShowsBody(body string) {
	s.session.AssertText(body)
}

func (s *canvasStagingConsoleSteps) thenConsoleEditIsStaged(body string) {
	s.canvas.WaitForStagingOnCurrentDraft()
	s.canvas.AssertStagingActionsVisibleAndEnabled()
	userID := s.canvas.UserIDForEmail(s.session.Account.Email)
	require.Eventually(s.t, func() bool {
		return s.canvas.StagingContainsConsoleTextForUser(userID, body)
	}, 15*time.Second, 200*time.Millisecond, "expected staged console.yaml to contain edited body")
}
