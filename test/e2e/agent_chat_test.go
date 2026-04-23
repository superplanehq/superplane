package e2e

import (
	"testing"
	"time"

	pw "github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
// fresh agents DB, a logged-in session, a canvas in edit mode, and the
// sidebar already open. Subtests should only contain their own whens/thens
// on top of this.
//
// Note: we do NOT flip OrganizationAgentSettings.AgentModeEnabled here.
// Sidebar visibility is gated purely by window.SUPERPLANE_AGENT_ENABLED
// (see useAgentState), which the e2e Go server sets from the AGENT_ENABLED=yes
// env var in test_context.go. The DB setting is only consulted by the
// organization settings UI today.
func newAgentChatScenario(t *testing.T) *agentChatSteps {
	steps := &agentChatSteps{t: t}
	steps.start()
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
	// AiMessage returns null for empty assistant content (see
	// AiBuilderChatMessage.tsx), so the selector below can only match once at
	// least one non-empty, non-failure content chunk has rendered — i.e. the
	// stream has started producing output. This is still weaker than "the
	// stream has fully completed": use thenTheStreamIsComplete before acting
	// on post-stream state such as the back button, page reload, or
	// persistence checks.
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

// thenTheAgentIsRunningInTestMode asserts that the most recent agent_chat_runs
// row for the current (org, canvas) has model="test". This fails fast if
// someone misconfigures AI_MODEL so the rest of the suite does not silently
// run against a real LLM.
//
// We check the DB instead of the DOM on purpose: the client-side TEST_MODE_HINT
// written by applyStreamOutcome is immediately overwritten when
// setCurrentChatId (in sendChatPrompt's finally block) triggers
// useLoadChatConversation to reload the conversation from agents_test — the
// DB-stored assistant content is the raw TestModel sentinel, not the hint,
// so polling the DOM for "test mode" is inherently racy. The
// agent_chat_runs.model column, written by PersistedRunRecorder, does not
// move and is the authoritative signal.
func (s *agentChatSteps) thenTheAgentIsRunningInTestMode() {
	// Manual poll instead of require.Eventually: the diagnostic message needs
	// to reflect the last observed (model, err), but Eventually evaluates its
	// msgAndArgs eagerly at call time — mutations to outer variables inside
	// the condition closure are not reflected in the already-wrapped interface
	// values, so the failure message would always report zero values. Same
	// pattern as thenTheChatIsPersistedInAgentsDB below.
	var lastModel string
	var lastErr error
	deadline := time.Now().Add(agentTestModeAssertionBudget)
	for time.Now().Before(deadline) {
		model, err := lastRunModelForCanvas(s.session.OrgID.String(), s.canvas.WorkflowID.String())
		lastModel, lastErr = model, err
		if err == nil && model == "test" {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.Failf(s.t, "agent not running in test mode",
		"expected agent_chat_runs.model='test' for the current canvas; is AI_MODEL=test set for the agent container? last observed model=%q err=%v",
		lastModel, lastErr)
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
	// The agent_chats row is written synchronously in the CreateAgentChat gRPC
	// handler at the start of sendChatPrompt (before streaming), so its
	// existence is nearly tautological once thenTheStreamIsComplete returned.
	// The meaningful signal that persistence actually worked end-to-end is in
	// agent_chat_messages, which PersistedRunRecorder.save_authoritative_messages
	// writes near the end of _stream_agent_run. Poll that briefly to avoid
	// racing the final save.
	orgID := s.session.OrgID.String()
	canvasID := s.canvas.WorkflowID.String()

	chatCount, err := countAgentChatsForCanvas(orgID, canvasID)
	require.NoError(s.t, err)
	assert.GreaterOrEqual(s.t, chatCount, int64(1), "expected at least one agent_chats row in agents_test DB")

	deadline := time.Now().Add(agentPersistencePollTimeout)
	var lastMsgCount int64
	var lastErr error
	for time.Now().Before(deadline) {
		count, err := countAgentChatMessagesForCanvas(orgID, canvasID)
		lastMsgCount, lastErr = count, err
		// At least user + assistant messages for a single successful run.
		if err == nil && count >= 2 {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	require.NoError(s.t, lastErr)
	assert.GreaterOrEqual(s.t, lastMsgCount, int64(2),
		"expected at least 2 agent_chat_messages rows (user + assistant) in agents_test DB; got %d", lastMsgCount)
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
	// AssertHidden passes immediately when no element matches the selector
	// (the desired state here) and otherwise waits for any existing match to
	// become hidden — so it gives us a DOM-flush barrier if a previous bubble
	// is still being unmounted.
	s.session.AssertHidden(q.TestID("agent-chat-message"))
	// Belt-and-braces: a strict count == 0 guards against future reuse of the
	// agent-chat-message test id on some non-bubble node that would still be
	// considered "hidden" here.
	count, err := s.session.Page().Locator(`[data-testid="agent-chat-message"]`).Count()
	require.NoError(s.t, err)
	assert.Equal(s.t, 0, count, "expected no visible chat messages after starting a new chat")
}
