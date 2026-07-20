package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	pw "github.com/mxschmitt/playwright-go"
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
	agentPollInterval = 200 * time.Millisecond
	agentWaitTimeout  = 30 * time.Second
	agentSendTimeout  = 20 * time.Second
)

func TestAgentE2E(t *testing.T) {
	t.Run("sends a message and renders the streamed assistant response", func(t *testing.T) {
		steps := newAgentSteps(t)
		steps.withSendMessageHandler(func(call support.AgentProviderSendMessageCall) ([]agents.ProviderEvent, error) {
			return agentAssistantTurn("E2E assistant response for: " + call.Message), nil
		})

		steps.start()
		steps.openAgent()
		steps.sendMessage("What is on this canvas?")
		steps.assertUserMessage("What is on this canvas?")
		steps.assertAssistantMessage("E2E assistant response for: What is on this canvas?")
		steps.assertLastSendUsedMode("What is on this canvas?", "[Agent Mode: ASK]")
	})

	t.Run("switches to build mode before sending a message", func(t *testing.T) {
		steps := newAgentSteps(t)
		steps.withSendMessageHandler(func(call support.AgentProviderSendMessageCall) ([]agents.ProviderEvent, error) {
			return agentAssistantTurn("Builder mode acknowledged"), nil
		})

		steps.start()
		steps.openAgent()
		steps.switchToBuildMode()
		steps.sendMessage("Add a noop node")
		steps.assertAssistantMessage("Builder mode acknowledged")
		steps.assertLastSendUsedMode("Add a noop node", "[Agent Mode: BUILD]")
	})

	t.Run("renders tool activity from provider events", func(t *testing.T) {
		steps := newAgentSteps(t)
		steps.withSendMessageHandler(func(call support.AgentProviderSendMessageCall) ([]agents.ProviderEvent, error) {
			return []agents.ProviderEvent{
				agentToolStartedEvent("tool-1", "bash", "superplane apps canvas get"),
				agentToolFinishedEvent("tool-1", "bash"),
				agentAssistantMessageEvent("Tool run complete"),
				agentTurnCompletedEvent(),
			}, nil
		})

		steps.start()
		steps.openAgent()
		steps.sendMessage("Inspect the canvas")
		steps.assertVisible(q.TestID("agent-tool-group"))
		steps.expandToolGroupIfNeeded()
		steps.assertVisible(q.TestID("agent-tool-message"))
		steps.assertAssistantMessage("Tool run complete")
	})

	t.Run("stops a running turn", func(t *testing.T) {
		steps := newAgentSteps(t)
		steps.withSendMessageHandler(func(call support.AgentProviderSendMessageCall) ([]agents.ProviderEvent, error) {
			return nil, nil
		})

		steps.start()
		steps.openAgent()
		steps.sendMessage("Keep running until I stop you")
		steps.assertVisible(q.TestID("agent-stop-button"))
		steps.stopAgent()
		steps.assertInterruptSent()
		// InterruptSession resets the row to idle and broadcasts
		// turn_completed/idle, so the composer flips back to "Ready" on
		// the WS event — regardless of any late provider stream error.
		// The "late session_failed must not overwrite idle" race is
		// covered as a unit test in
		// TestAgentStreamWorker_DoesNotOverwriteIdleWithFailedAfterInterrupt;
		// reproducing it here would race InterruptSession's DB commit
		// (assertInterruptSent fires as soon as the provider call is
		// recorded, mid-flight) and flake.
		steps.assertText("Ready")
		steps.assertHidden(q.TestID("agent-stop-button"))
	})

	t.Run("sends a follow-up message while a turn is still running", func(t *testing.T) {
		steps := newAgentSteps(t)
		steps.withSendMessageHandler(func(call support.AgentProviderSendMessageCall) ([]agents.ProviderEvent, error) {
			return nil, nil
		})

		steps.start()
		steps.openAgent()
		steps.sendMessage("Keep running while I add more context")
		steps.assertVisible(q.TestID("agent-stop-button"))
		steps.fillMessage("Here is more context")
		steps.assertSendButtonEnabled()
		steps.submitMessage()
		steps.waitForSendCall(func(call support.AgentProviderSendMessageCall) bool {
			return call.Message == "Here is more context"
		})
	})

	t.Run("starts building from a rubric response", func(t *testing.T) {
		var approvalReceived bool
		steps := newAgentSteps(t)
		steps.withSendMessageHandler(func(call support.AgentProviderSendMessageCall) ([]agents.ProviderEvent, error) {
			if call.Message == "Specs approved. Start building." {
				approvalReceived = true
				return agentAssistantTurn("Building now."), nil
			}

			return agentAssistantTurn(strings.Join([]string{
				"Here is the plan.",
				"",
				":::rubric E2E Build Plan",
				"- Add a manual trigger",
				"- Add a noop node",
				":::",
			}, "\n")), nil
		})

		steps.start()
		steps.openAgent()
		steps.sendMessage("Create a build plan")
		steps.assertText("E2E Build Plan")
		steps.startBuildingFromRubric()
		steps.assertText("Building now.")
		require.True(t, approvalReceived, "expected 'Specs approved. Start building.' message")
	})
}

type agentSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func newAgentSteps(t *testing.T) *agentSteps {
	ctx.ResetAgentProvider()
	return &agentSteps{t: t}
}

func (s *agentSteps) withSendMessageHandler(
	handler func(support.AgentProviderSendMessageCall) ([]agents.ProviderEvent, error),
) {
	ctx.AgentProvider.SetSendMessageHandler(handler)
}

func (s *agentSteps) start() {
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

	s.canvas = shared.NewCanvasSteps("E2E Agent "+uuid.NewString(), s.t, s.session)
	s.canvas.Create()
}

func (s *agentSteps) openAgent() {
	s.waitForToolSidebarOpen()

	s.session.AssertVisible(q.TestID("canvas-tool-sidebar"))
	s.waitForAgentInput()

	// Opening or refreshing a canvas must not invoke the agent: it no longer sends a
	// boot message (see web_src/src/lib/agentBootContext.ts). Sync on the provisioned
	// session instead of a boot round-trip, then assert nothing was auto-sent.
	s.currentAgentSession()
	s.assertNoBootMessageSent()
}

func (s *agentSteps) assertNoBootMessageSent() {
	require.Never(s.t, func() bool {
		for _, call := range ctx.AgentProvider.SendMessageCalls() {
			if isAgentSystemMessage(call.Message) {
				return true
			}
		}

		return false
	}, 2*time.Second, agentPollInterval, "agent must not auto-send a boot message on canvas open")
}

func (s *agentSteps) waitForAgentInput() {
	input := q.TestID("agent-input").Run(s.session)
	require.Eventually(s.t, func() bool {
		visible, err := input.IsVisible()
		return err == nil && visible
	}, agentWaitTimeout, agentPollInterval)
}

func (s *agentSteps) waitForToolSidebarOpen() {
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

func (s *agentSteps) switchToBuildMode() {
	s.session.Click(q.TestID("agent-mode-builder"))
	s.assertEventuallyVisible(q.Locator(`[data-testid="agent-mode-builder"][aria-pressed="true"]`))
}

func (s *agentSteps) switchToAskMode() {
	s.session.Click(q.TestID("agent-mode-operator"))
	s.assertEventuallyVisible(q.Locator(`[data-testid="agent-mode-operator"][aria-pressed="true"]`))
}

func (s *agentSteps) sendMessage(message string) {
	s.fillMessage(message)
	s.submitMessage()
}

func (s *agentSteps) fillMessage(message string) {
	s.session.FillIn(q.TestID("agent-input"), message)
}

func (s *agentSteps) submitMessage() {
	s.session.Click(q.TestID("agent-send-message-button"))
}

func (s *agentSteps) assertSendButtonEnabled() {
	button := q.TestID("agent-send-message-button").Run(s.session)
	require.Eventually(s.t, func() bool {
		disabled, err := button.IsDisabled()
		return err == nil && !disabled
	}, 10*time.Second, 200*time.Millisecond)
}

func (s *agentSteps) stopAgent() {
	s.session.Click(q.TestID("agent-stop-button"))
}

func (s *agentSteps) startBuildingFromRubric() {
	s.session.Click(q.Locator(`button:has-text("Start Building")`))
}

func (s *agentSteps) currentAgentSession() *models.AgentSession {
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
	}, 10*time.Second, 200*time.Millisecond)

	return agentSession
}

func (s *agentSteps) waitForSendCall(matches func(support.AgentProviderSendMessageCall) bool) support.AgentProviderSendMessageCall {
	var matched support.AgentProviderSendMessageCall
	require.Eventually(s.t, func() bool {
		for _, call := range ctx.AgentProvider.SendMessageCalls() {
			if matches(call) {
				matched = call
				return true
			}
		}

		return false
	}, agentSendTimeout, agentPollInterval)

	return matched
}

func (s *agentSteps) assertLastSendUsedMode(message string, modeMarker string) {
	call := s.waitForSendCall(func(call support.AgentProviderSendMessageCall) bool {
		return call.Message == message
	})
	require.Contains(s.t, call.Options.ContextPreamble, modeMarker)
}

func (s *agentSteps) assertInterruptSent() {
	require.Eventually(s.t, func() bool {
		return len(ctx.AgentProvider.InterruptSessionCalls()) > 0
	}, 10*time.Second, 200*time.Millisecond)
}

func (s *agentSteps) assertDefineOutcomeSent() {
	require.Eventually(s.t, func() bool {
		calls := ctx.AgentProvider.DefineOutcomeCalls()
		if len(calls) == 0 {
			return false
		}

		last := calls[len(calls)-1]
		return strings.Contains(last.Options.Description, "E2E Build Plan") &&
			strings.Contains(last.Options.Rubric, "Add a manual trigger") &&
			strings.Contains(last.Options.ContextPreamble, "[Agent Mode: BUILD]")
	}, 10*time.Second, 200*time.Millisecond)
}

func (s *agentSteps) assertUserMessage(message string) {
	s.assertEventuallyVisible(q.Locator(fmt.Sprintf(`[data-testid="agent-user-message"]:has-text("%s")`, message)))
}

func (s *agentSteps) assertAssistantMessage(message string) {
	s.assertEventuallyVisible(q.Locator(fmt.Sprintf(`[data-testid="agent-assistant-message"]:has-text("%s")`, message)))
}

func (s *agentSteps) expandToolGroupIfNeeded() {
	toolMessage := s.session.Page().GetByTestId("agent-tool-message").First()
	visible, err := toolMessage.IsVisible()
	require.NoError(s.t, err)
	if visible {
		return
	}

	s.session.Click(q.Locator(`[data-testid="agent-tool-group"] button`))
}

func (s *agentSteps) assertText(text string) {
	locator := s.session.Page().Locator("text=" + text).First()
	require.Eventually(s.t, func() bool {
		visible, err := locator.IsVisible()
		return err == nil && visible
	}, agentWaitTimeout, agentPollInterval)
}

func (s *agentSteps) assertEventuallyVisible(query q.Query) {
	locator := query.Run(s.session)
	require.Eventually(s.t, func() bool {
		visible, err := locator.IsVisible()
		return err == nil && visible
	}, agentWaitTimeout, agentPollInterval)
}

func (s *agentSteps) assertVisible(query q.Query) {
	s.assertEventuallyVisible(query)
}

func (s *agentSteps) assertHidden(query q.Query) {
	s.session.AssertHidden(query)
}

func isAgentSystemMessage(message string) bool {
	return strings.HasPrefix(message, "@@system: ")
}

func agentAssistantTurn(text string) []agents.ProviderEvent {
	return []agents.ProviderEvent{
		agentAssistantMessageEvent(text),
		agentTurnCompletedEvent(),
	}
}

func agentAssistantMessageEvent(text string) agents.ProviderEvent {
	return agents.ProviderEvent{
		ProviderEventID: "assistant-" + uuid.NewString(),
		Type:            agents.ProviderEventAssistantMessage,
		Text:            text,
	}
}

func agentToolStartedEvent(callID, name, input string) agents.ProviderEvent {
	return agents.ProviderEvent{
		ProviderEventID: "tool-started-" + uuid.NewString(),
		Type:            agents.ProviderEventToolUseStarted,
		ToolCallID:      callID,
		ToolName:        name,
		ToolInput:       input,
	}
}

func agentToolFinishedEvent(callID, name string) agents.ProviderEvent {
	return agents.ProviderEvent{
		ProviderEventID: "tool-finished-" + uuid.NewString(),
		Type:            agents.ProviderEventToolUseFinished,
		ToolCallID:      callID,
		ToolName:        name,
	}
}
