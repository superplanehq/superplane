package e2e

import (
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

// TestAgentChat exercises the real Python agent service (via pydantic-ai
// TestModel, so no external LLM calls) through the canvas agent sidebar.
// The agent container must be running with AI_MODEL=test and DB_NAME=agents_test;
// see docker-compose.e2e.yml and Makefile `test.start`.
func TestAgentChat(t *testing.T) {
	t.Run("sending a prompt streams an assistant reply", func(t *testing.T) {
		steps := &agentChatSteps{t: t}
		steps.start()
		steps.givenAnOrgWithAgentModeEnabled()
		steps.givenACanvasInEditMode()
		steps.whenIOpenTheAgentSidebar()
		steps.whenISendPrompt("hello")
		steps.thenISeeTheUserMessage("hello")
		steps.thenISeeAnAssistantMessage()
	})

	t.Run("a new chat appears in the sessions list and is persisted in the agents DB", func(t *testing.T) {
		steps := &agentChatSteps{t: t}
		steps.start()
		steps.givenAnOrgWithAgentModeEnabled()
		steps.givenACanvasInEditMode()
		steps.whenIOpenTheAgentSidebar()
		steps.whenISendPrompt("first prompt")
		steps.thenISeeAnAssistantMessage()
		steps.whenIClickBackToStartNewChat()
		steps.thenISeeAtLeastOneSessionInTheList()
		steps.thenTheChatIsPersistedInAgentsDB()
	})

	t.Run("conversation is restored on page reload", func(t *testing.T) {
		steps := &agentChatSteps{t: t}
		steps.start()
		steps.givenAnOrgWithAgentModeEnabled()
		steps.givenACanvasInEditMode()
		steps.whenIOpenTheAgentSidebar()
		steps.whenISendPrompt("remember me")
		steps.thenISeeAnAssistantMessage()

		steps.whenIReloadThePage()
		steps.whenIOpenTheAgentSidebar()
		steps.whenISelectTheFirstSession()
		steps.thenISeeTheUserMessage("remember me")
		steps.thenISeeAnAssistantMessage()
	})

	t.Run("starting a new chat clears the conversation", func(t *testing.T) {
		steps := &agentChatSteps{t: t}
		steps.start()
		steps.givenAnOrgWithAgentModeEnabled()
		steps.givenACanvasInEditMode()
		steps.whenIOpenTheAgentSidebar()
		steps.whenISendPrompt("first")
		steps.thenISeeAnAssistantMessage()

		steps.whenIClickBackToStartNewChat()
		steps.thenTheInputIsEmpty()
		steps.thenThereAreNoVisibleMessages()
	})
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
		Timeout: pw.Float(10000),
	}))
	require.NoError(s.t, trigger.Click())
	require.NoError(s.t, sidebar.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(10000),
	}))
}

func (s *agentChatSteps) whenISendPrompt(prompt string) {
	s.session.FillIn(q.TestID("agent-chat-input"), prompt)
	// Wait for the send button to become enabled (it's gated on trimmed input).
	send := q.TestID("agent-chat-send").Run(s.session)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		disabled, err := send.IsDisabled()
		require.NoError(s.t, err)
		if !disabled {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.session.Click(q.TestID("agent-chat-send"))
}

func (s *agentChatSteps) thenISeeTheUserMessage(expected string) {
	// Locate the most-recent user message by role and wait for its text.
	loc := s.session.Page().Locator(`[data-testid="agent-chat-message"][data-role="user"]`).
		Filter(pw.LocatorFilterOptions{HasText: expected}).
		First()
	require.NoError(s.t, loc.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(10000),
	}))
}

func (s *agentChatSteps) thenISeeAnAssistantMessage() {
	// TestModel output text is non-deterministic, so we can't assert a specific
	// body — but we must exclude the UI's GENERIC_FAILURE_MESSAGE from
	// agentChatUi.ts, which also renders with data-role="assistant" when the
	// stream fails. Match any assistant message whose text is non-empty and is
	// not the failure string.
	const genericFailure = "I couldn't generate changes right now. Please try again."
	loc := s.session.Page().
		Locator(`[data-testid="agent-chat-message"][data-role="assistant"]`).
		Filter(pw.LocatorFilterOptions{HasNotText: genericFailure}).
		First()
	require.NoError(s.t, loc.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(20000),
	}))
}

func (s *agentChatSteps) whenIClickBackToStartNewChat() {
	s.session.Click(q.TestID("agent-sidebar-back-button"))
	// The back button calls handleStartNewChatSession which clears state; give
	// React a tick to re-render the sessions list.
	s.session.Sleep(300)
}

func (s *agentChatSteps) thenISeeAtLeastOneSessionInTheList() {
	loc := s.session.Page().Locator(`[data-testid="agent-chat-session-item"]`).First()
	require.NoError(s.t, loc.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(10000),
	}))
}

func (s *agentChatSteps) thenTheChatIsPersistedInAgentsDB() {
	// Persistence happens from the Python agent near the end of the stream;
	// poll briefly to avoid racing the final save.
	deadline := time.Now().Add(5 * time.Second)
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
		Timeout: pw.Float(10000),
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
	// After starting a new chat, the message list should be empty.
	count, err := s.session.Page().Locator(`[data-testid="agent-chat-message"]`).Count()
	require.NoError(s.t, err)
	assert.Equal(s.t, 0, count, "expected no visible chat messages after starting a new chat")
}
