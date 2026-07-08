package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	pw "github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
	"github.com/superplanehq/superplane/test/support"
)

const (
	agentStagedNoopName      = "AgentStagedNoop"
	agentStagingEditTimeout  = 30 * time.Second
	agentStagingPollInterval = 200 * time.Millisecond
)

func TestAgentStagingEditTransition(t *testing.T) {
	t.Run("auto-enters edit mode from live view when agent stages via websocket", func(t *testing.T) {
		steps := newAgentStagingEditSteps(t)
		steps.withSendMessageHandler(func(call support.AgentProviderSendMessageCall) ([]agents.ProviderEvent, error) {
			steps.canvas.StageNoopNodeViaAgentPatch(agentStagedNoopName, models.Position{X: 500, Y: 200})
			// Real turns also emit staging-actions, but auto-enter must work from the
			// staging_updated websocket alone when the user is already on live view.
			return agentAssistantTurn("Added agent noop node."), nil
		})

		steps.start()
		steps.warmLiveViewStagingCaches()
		steps.openAgent()
		steps.switchToBuildMode()
		steps.sendMessage("Add a noop node to the canvas")
		steps.assertAssistantMessage("Added agent noop node")
		steps.assertAutoEnteredEditModeWithoutManualEdit()
		steps.assertStagedNodeVisibleInEditor(agentStagedNoopName)
		steps.canvas.AssertEditModeTabChrome()
		steps.canvas.AssertStagingActionsVisibleAndEnabled()
		steps.canvas.AssertLiveCanvasLacksNode(agentStagedNoopName)
		steps.canvas.AssertHasStaging(uuid.Nil)
	})

	t.Run("already in edit shows agent staged changes without reload", func(t *testing.T) {
		steps := newAgentStagingEditSteps(t)
		steps.start()
		steps.warmLiveViewStagingCaches()
		steps.canvas.EnterEditMode()
		steps.canvas.StageNoopNodeViaAgentPatch(agentStagedNoopName, models.Position{X: 500, Y: 200})
		steps.assertStagedNodeVisibleInEditor(agentStagedNoopName)
		steps.canvas.AssertEditModeTabChrome()
		steps.canvas.AssertStagingActionsVisibleAndEnabled()
		steps.canvas.AssertLiveCanvasLacksNode(agentStagedNoopName)
		steps.canvas.AssertHasStaging(uuid.Nil)
	})
}

type agentStagingEditSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func newAgentStagingEditSteps(t *testing.T) *agentStagingEditSteps {
	ctx.ResetAgentProvider()
	return &agentStagingEditSteps{t: t}
}

func (s *agentStagingEditSteps) withSendMessageHandler(
	handler func(support.AgentProviderSendMessageCall) ([]agents.ProviderEvent, error),
) {
	ctx.AgentProvider.SetSendMessageHandler(handler)
}

func (s *agentStagingEditSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
	require.NoError(s.t, models.EnableExperimentalFeature(s.session.OrgID, features.FeatureClaudeManagedAgents))
	require.NoError(s.t, s.session.Page().AddInitScript(pw.Script{Content: pw.String(`
		() => {
			window.localStorage.setItem("canvasAgentMode", "operator");
			window.localStorage.setItem("canvasAgentSidebarOpen", "false");
			window.sessionStorage.clear();
		}
	`)}))

	s.canvas = shared.NewCanvasSteps("E2E Agent Staging Edit "+uuid.NewString(), s.t, s.session)
	s.canvas.Create()
}

// warmLiveViewStagingCaches reproduces the manual bug preconditions on a canvas that
// has already been open on live view: React Query holds hasStaging=false from the
// initial /staging fetch and a warm stagedCanvasSpec snapshot from a prior edit session.
func (s *agentStagingEditSteps) warmLiveViewStagingCaches() {
	s.assertOnLiveView()
	s.canvas.EnterEditMode()
	s.canvas.ExitEditMode()
	s.assertOnLiveView()
}

func (s *agentStagingEditSteps) openAgent() {
	s.waitForToolSidebarOpen()
	s.session.AssertVisible(q.TestID("canvas-tool-sidebar"))
	s.waitForAgentInput()
	s.currentAgentSession()
}

func (s *agentStagingEditSteps) waitForToolSidebarOpen() {
	deadline := time.Now().Add(agentWaitTimeout)
	sidebar := q.TestID("canvas-tool-sidebar").Run(s.session)
	openButton := q.TestID("canvas-tool-sidebar-toggle").Run(s.session)

	for time.Now().Before(deadline) {
		visible, err := sidebar.IsVisible()
		require.NoError(s.t, err)
		if visible {
			return
		}

		visible, err = openButton.IsVisible()
		require.NoError(s.t, err)
		if visible {
			err = openButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(1000)})
			if err == nil {
				continue
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	s.session.AssertVisible(q.TestID("canvas-tool-sidebar"))
}

func (s *agentStagingEditSteps) waitForAgentInput() {
	input := q.TestID("agent-input").Run(s.session)
	require.Eventually(s.t, func() bool {
		visible, err := input.IsVisible()
		return err == nil && visible
	}, agentWaitTimeout, agentPollInterval)
}

func (s *agentStagingEditSteps) switchToBuildMode() {
	s.session.Click(q.TestID("agent-mode-builder"))
	s.assertEventuallyVisible(q.Locator(`[data-testid="agent-mode-builder"][aria-pressed="true"]`))
}

func (s *agentStagingEditSteps) sendMessage(message string) {
	s.session.FillIn(q.TestID("agent-input"), message)
	s.session.Click(q.TestID("agent-send-message-button"))
}

func (s *agentStagingEditSteps) currentAgentSession() *models.AgentSession {
	user, err := models.FindActiveUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)

	var agentSession *models.AgentSession
	require.Eventually(s.t, func() bool {
		session, err := models.FindAgentSessionByCanvasInTransaction(
			database.Conn(),
			s.session.OrgID,
			user.ID,
			s.canvas.WorkflowID,
		)
		if err != nil {
			return false
		}

		agentSession = session
		return true
	}, 10*time.Second, agentPollInterval)

	return agentSession
}

func (s *agentStagingEditSteps) assertOnLiveView() {
	editButton := q.TestID("canvas-edit-button").Run(s.session)
	require.Eventually(s.t, func() bool {
		visible, err := editButton.IsVisible()
		if err != nil || !visible {
			return false
		}
		disabled, err := editButton.IsDisabled()
		return err == nil && !disabled
	}, agentStagingEditTimeout, agentStagingPollInterval)

	exitEditButton := q.TestID("canvas-exit-edit-button").Run(s.session)
	visible, err := exitEditButton.IsVisible()
	require.NoError(s.t, err)
	require.False(s.t, visible, "expected live view")
}

func (s *agentStagingEditSteps) assertAutoEnteredEditModeWithoutManualEdit() {
	exitEditButton := q.TestID("canvas-exit-edit-button").Run(s.session)
	require.Eventually(s.t, func() bool {
		visible, err := exitEditButton.IsVisible()
		if err != nil || !visible {
			return false
		}
		disabled, err := exitEditButton.IsDisabled()
		return err == nil && !disabled
	}, agentStagingEditTimeout, agentStagingPollInterval, "agent staging should auto-enter edit mode from live view")

	editButton := q.TestID("canvas-edit-button").Run(s.session)
	visible, err := editButton.IsVisible()
	require.NoError(s.t, err)
	require.False(s.t, visible, "edit button should be hidden after auto-entering edit mode")
}

func (s *agentStagingEditSteps) assertStagedNodeVisibleInEditor(nodeName string) {
	s.session.AssertVisible(q.TestID("node", nodeName, "header"))
}

func (s *agentStagingEditSteps) assertAssistantMessage(message string) {
	s.assertEventuallyVisible(q.Locator(fmt.Sprintf(`[data-testid="agent-assistant-message"]:has-text("%s")`, message)))
}

func (s *agentStagingEditSteps) assertEventuallyVisible(query q.Query) {
	locator := query.Run(s.session)
	require.Eventually(s.t, func() bool {
		visible, err := locator.IsVisible()
		return err == nil && visible
	}, agentStagingEditTimeout, agentStagingPollInterval)
}
