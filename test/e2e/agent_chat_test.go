package e2e

import (
	"strings"
	"testing"
	"time"

	pw "github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

// Named timeouts keep the intent of each wait obvious and easy to tune.
// The "Ms" variants are used with Playwright APIs that take milliseconds as float64;
// the time.Duration variants are used by Go-level polls (require.Eventually, etc.).
const (
	agentSidebarTimeoutMs        = 10000.0
	agentStreamVisibleTimeoutMs  = 20000.0
	agentStreamCompleteTimeout   = 15 * time.Second
	agentSendEnableTimeout       = 5 * time.Second
	agentPersistencePollTimeout  = 5 * time.Second
	agentTestModeAssertionBudget = 5 * time.Second
)

// TestAgentChat exercises the real Python agent service (via pydantic-ai
// TestModel, so no external LLM calls) through the canvas agent sidebar.
// The agent container must be running with AI_MODEL=test and DB_NAME=agents_test;
// see docker-compose.e2e.yml and Makefile `test.start`.
func TestAgentChat(t *testing.T) {
	t.Run("sending a prompt streams an assistant reply", func(t *testing.T) {
		steps := newAgentChatScenario(t)
		steps.whenISendPrompt("hello")
		steps.thenISeeTheUserMessage("hello")
		steps.thenISeeAnAssistantMessage()
		steps.thenTheStreamIsComplete()
		// Safety net: fail early if someone swapped AI_MODEL away from "test",
		// since the rest of the suite silently relies on TestModel behavior.
		steps.thenTheAgentIsRunningInTestMode()
	})

	t.Run("a new chat appears in the sessions list and is persisted in the agents DB", func(t *testing.T) {
		steps := newAgentChatScenario(t)
		steps.whenISendPrompt("first prompt")
		steps.thenISeeAnAssistantMessage()
		steps.thenTheStreamIsComplete()
		steps.whenIClickBackToStartNewChat()
		steps.thenISeeAtLeastOneSessionInTheList()
		steps.thenTheChatIsPersistedInAgentsDB()
	})

	t.Run("conversation is restored on page reload", func(t *testing.T) {
		steps := newAgentChatScenario(t)
		steps.whenISendPrompt("remember me")
		steps.thenISeeAnAssistantMessage()
		steps.thenTheStreamIsComplete()

		steps.whenIReloadThePage()
		steps.whenIOpenTheAgentSidebar()
		steps.whenISelectTheFirstSession()
		steps.thenISeeTheUserMessage("remember me")
		steps.thenISeeAnAssistantMessage()
	})

	t.Run("starting a new chat clears the conversation", func(t *testing.T) {
		steps := newAgentChatScenario(t)
		steps.whenISendPrompt("first")
		steps.thenISeeAnAssistantMessage()
		steps.thenTheStreamIsComplete()

		steps.whenIClickBackToStartNewChat()
		steps.thenTheInputIsEmpty()
		steps.thenThereAreNoVisibleMessages()
	})
}

// newAgentChatScenario wires up the setup every agent chat subtest needs:
// fresh agents DB, a logged-in session with the agent mode flag flipped on,
// a canvas in edit mode, and the sidebar already open. Subtests should only
// contain their own whens/thens on top of this.
func newAgentChatScenario(t *testing.T) *agentChatSteps {
	steps := &agentChatSteps{t: t}
	steps.start()
	steps.givenAnOrgWithAgentModeEnabled()
	steps.givenACanvasInEditMode()
	steps.whenIOpenTheAgentSidebar()
	return steps
}

type agentChatSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *agentChatSteps) start() {
	// Wipe agents_test before each subtest so session lists and counts are
	// deterministic across runs.
	require.NoError(s.t, truncateAgentChatTables())

	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *agentChatSteps) givenAnOrgWithAgentModeEnabled() {
	err := models.UpsertOrganizationAgentSettings(&models.OrganizationAgentSettings{
		OrganizationID:   s.session.OrgID,
		AgentModeEnabled: true,
		OpenAIKeyStatus:  models.OrganizationAgentOpenAIKeyStatusNotConfigured,
	})
	require.NoError(s.t, err)
}

func (s *agentChatSteps) givenACanvasInEditMode() {
	s.canvas = shared.NewCanvasSteps("Agent Chat E2E", s.t, s.session)
	s.canvas.Create()
	// EnterEditMode flips the page out of read-only so the agent sidebar
	// trigger is rendered (see useAgentState.showAgentSidebarToggle).
	s.canvas.EnterEditMode()
}

func (s *agentChatSteps) whenIOpenTheAgentSidebar() {
	page := s.session.Page()
	sidebar := page.GetByTestId("agent-sidebar")
	if isVisible, _ := sidebar.IsVisible(); isVisible {
		return
	}

	trigger := page.GetByTestId("open-agent-sidebar")
	require.NoError(s.t, trigger.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(agentSidebarTimeoutMs),
	}))
	require.NoError(s.t, trigger.Click())
	require.NoError(s.t, sidebar.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(agentSidebarTimeoutMs),
	}))
}

func (s *agentChatSteps) whenISendPrompt(prompt string) {
	s.session.FillIn(q.TestID("agent-chat-input"), prompt)
	// The send button is gated on trimmed input; assert it enables before we
	// click so a regression in that gating surfaces a clear failure instead of
	// a generic "click timed out".
	send := q.TestID("agent-chat-send").Run(s.session)
	require.Eventually(s.t, func() bool {
		disabled, err := send.IsDisabled()
		return err == nil && !disabled
	}, agentSendEnableTimeout, 100*time.Millisecond, "send button never enabled after filling prompt")
	s.session.Click(q.TestID("agent-chat-send"))
}

func (s *agentChatSteps) thenISeeTheUserMessage(expected string) {
	// Locate the most-recent user message by role and wait for its text.
	loc := s.session.Page().Locator(`[data-testid="agent-chat-message"][data-role="user"]`).
		Filter(pw.LocatorFilterOptions{HasText: expected}).
		First()
	require.NoError(s.t, loc.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(agentSidebarTimeoutMs),
	}))
}

func (s *agentChatSteps) thenISeeAnAssistantMessage() {
	// TestModel output text is non-deterministic, so we can't assert a specific
	// body — but we must exclude the UI's GENERIC_FAILURE_MESSAGE from
	// agentChatUi.ts, which also renders with data-role="assistant" when the
	// stream fails. Match any assistant message whose text is non-empty and is
	// not the failure string.
	//
	// Note: this only confirms the bubble has been rendered (streaming started);
	// use thenTheStreamIsComplete before acting on post-stream state such as
	// the back button, page reload, or persistence checks.
	const genericFailure = "I couldn't generate changes right now. Please try again."
	loc := s.session.Page().
		Locator(`[data-testid="agent-chat-message"][data-role="assistant"]`).
		Filter(pw.LocatorFilterOptions{HasNotText: genericFailure}).
		First()
	require.NoError(s.t, loc.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(agentStreamVisibleTimeoutMs),
	}))
}

// thenTheStreamIsComplete waits until the agent stream has fully finished.
// currentChatId is only assigned in the finally block of sendChatPrompt after
// the stream closes, and the back button is rendered iff currentChatId !== null
// (see showBack in AgentSidebar). For a scenario that starts with no current
// chat, the back button appearing is therefore a deterministic post-stream
// signal. We can't use the send button's disabled state here because it is
// also bound to !aiInput.trim(), which stays truthy while the input is empty.
func (s *agentChatSteps) thenTheStreamIsComplete() {
	back := s.session.Page().Locator(`[data-testid="agent-sidebar-back-button"]`)
	require.NoError(s.t, back.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(float64(agentStreamCompleteTimeout.Milliseconds())),
	}), "agent stream did not finish (back button never appeared)")
}

// thenTheAgentIsRunningInTestMode asserts the assistant bubble rendered by
// the suite contains the client-side "test mode" hint, which is only emitted
// when the server reports runModel === "test" (see applyStreamOutcome in
// agentChatUi.ts). This fails fast if someone misconfigures AI_MODEL so the
// rest of the suite does not silently run against a real LLM.
func (s *agentChatSteps) thenTheAgentIsRunningInTestMode() {
	all := s.session.Page().Locator(`[data-testid="agent-chat-message"][data-role="assistant"]`)
	var lastTexts []string
	require.Eventually(s.t, func() bool {
		texts, err := all.AllTextContents()
		if err != nil {
			return false
		}
		lastTexts = texts
		for _, text := range texts {
			if strings.Contains(strings.ToLower(text), "test mode") {
				return true
			}
		}
		return false
	}, agentTestModeAssertionBudget, 100*time.Millisecond,
		"expected assistant output to contain the TEST model hint; is AI_MODEL=test set for the agent container? last observed texts=%#v", lastTexts)
}

func (s *agentChatSteps) whenIClickBackToStartNewChat() {
	s.session.Click(q.TestID("agent-sidebar-back-button"))
	// handleStartNewChatSession sets currentChatId to null, which removes the
	// back button from the DOM. Waiting for it to disappear is a deterministic
	// signal that the state transition has flushed to the render tree — no
	// timing heuristics needed.
	s.session.AssertHidden(q.TestID("agent-sidebar-back-button"))
}

func (s *agentChatSteps) thenISeeAtLeastOneSessionInTheList() {
	loc := s.session.Page().Locator(`[data-testid="agent-chat-session-item"]`).First()
	require.NoError(s.t, loc.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(agentSidebarTimeoutMs),
	}))
}

func (s *agentChatSteps) thenTheChatIsPersistedInAgentsDB() {
	// Persistence happens from the Python agent near the end of the stream;
	// poll briefly to avoid racing the final save.
	deadline := time.Now().Add(agentPersistencePollTimeout)
	var lastCount int64
	var lastErr error
	for time.Now().Before(deadline) {
		count, err := countAgentChatsForCanvas(s.session.OrgID.String(), s.canvas.WorkflowID.String())
		lastCount, lastErr = count, err
		if err == nil && count >= 1 {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	require.NoError(s.t, lastErr)
	assert.GreaterOrEqual(s.t, lastCount, int64(1), "expected at least one agent_chats row in agents_test DB")
}

func (s *agentChatSteps) whenIReloadThePage() {
	_, err := s.session.Page().Reload(pw.PageReloadOptions{
		WaitUntil: pw.WaitUntilStateDomcontentloaded,
	})
	require.NoError(s.t, err)
	s.canvas.EnterEditMode()
}

func (s *agentChatSteps) whenISelectTheFirstSession() {
	loc := s.session.Page().Locator(`[data-testid="agent-chat-session-button"]`).First()
	require.NoError(s.t, loc.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(agentSidebarTimeoutMs),
	}))
	require.NoError(s.t, loc.Click())
}

func (s *agentChatSteps) thenTheInputIsEmpty() {
	input := q.TestID("agent-chat-input").Run(s.session)
	value, err := input.InputValue()
	require.NoError(s.t, err)
	assert.Equal(s.t, "", value)
}

func (s *agentChatSteps) thenThereAreNoVisibleMessages() {
	// First await the DOM flush: AssertHidden on a TestID that matches no
	// element resolves immediately in Playwright, so this guards against the
	// bubbles simply not being unmounted yet.
	s.session.AssertHidden(q.TestID("agent-chat-message"))
	// Then do a strict count assertion as a final invariant check in case any
	// non-bubble node ever starts carrying the same test ID.
	count, err := s.session.Page().Locator(`[data-testid="agent-chat-message"]`).Count()
	require.NoError(s.t, err)
	assert.Equal(s.t, 0, count, "expected no visible chat messages after starting a new chat")
}
