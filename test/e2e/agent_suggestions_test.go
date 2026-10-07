package e2e

import (
	"encoding/json"
	"fmt"
	"testing"

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

type agentSuggestion struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Prompt string `json:"prompt"`
}

func TestAgentSuggestionsE2E(t *testing.T) {
	t.Run("clicking a suggestion sends the prompt and dismisses only that option", func(t *testing.T) {
		suggestions := []agentSuggestion{
			{
				ID:     "add-ci",
				Label:  "Add CI to my loop",
				Prompt: "E2E suggestion prompt: add CI validation to this canvas",
			},
			{
				ID:     "fail-notifications",
				Label:  "Add notifications if something fails",
				Prompt: "E2E suggestion prompt: add failure notifications",
			},
			{
				ID:     "slack-approvals",
				Label:  "Connect Slack for approvals",
				Prompt: "E2E suggestion prompt: connect Slack approvals",
			},
		}

		steps := newAgentSuggestionSteps(t)
		steps.withSendMessageHandler(func(call support.AgentProviderSendMessageCall) ([]agents.ProviderEvent, error) {
			return agentAssistantTurn("E2E assistant response for suggestion"), nil
		})

		steps.startWithSuggestions(suggestions)
		steps.assertSuggestionsBadgeCount(3)
		steps.assertSuggestionVisible("add-ci")
		steps.assertSuggestionVisible("fail-notifications")

		steps.clickSuggestion("add-ci")
		steps.assertUserMessage(suggestions[0].Prompt)
		steps.assertAssistantMessage("E2E assistant response for suggestion")
		steps.assertSuggestionHidden("add-ci")
		steps.assertSuggestionsBadgeCount(2)
		steps.assertSuggestionVisible("fail-notifications")
		steps.assertSuggestionVisible("slack-approvals")

		// Dismissals persist in user_canvas_preferences across reloads.
		_, err := steps.session.Page().Reload()
		require.NoError(t, err)
		steps.waitForAgentToggle()
		steps.assertSuggestionHidden("add-ci")
		steps.assertSuggestionsBadgeCount(2)
		steps.assertSuggestionVisible("fail-notifications")
	})
}

type agentSuggestionSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func newAgentSuggestionSteps(t *testing.T) *agentSuggestionSteps {
	ctx.ResetAgentProvider()
	return &agentSuggestionSteps{t: t}
}

func (s *agentSuggestionSteps) withSendMessageHandler(
	handler func(support.AgentProviderSendMessageCall) ([]agents.ProviderEvent, error),
) {
	ctx.AgentProvider.SetSendMessageHandler(handler)
}

func (s *agentSuggestionSteps) startWithSuggestions(suggestions []agentSuggestion) {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
	require.NoError(s.t, models.EnableExperimentalFeature(s.session.OrgID, features.FeatureClaudeManagedAgents))

	s.canvas = shared.NewCanvasSteps("E2E Agent Suggestions "+uuid.NewString(), s.t, s.session)
	s.createCanvasWithoutVisit()

	script := fmt.Sprintf(`(() => {
		window.localStorage.setItem("canvasAgentMode", "operator");
		window.localStorage.setItem("canvasAgentSidebarOpen", "false");
		window.localStorage.setItem(%q, "false");
	})();`, "canvasAgentSidebarOpen:"+s.canvas.WorkflowID.String())
	require.NoError(s.t, s.session.Page().AddInitScript(pw.Script{Content: pw.String(script)}))

	s.canvas.Visit()
	s.seedSuggestions(suggestions)
	// sessionStorage survives reload; AppPage reads suggestions on mount.
	_, err := s.session.Page().Reload()
	require.NoError(s.t, err)
	s.waitForAgentToggle()
}

func (s *agentSuggestionSteps) seedSuggestions(suggestions []agentSuggestion) {
	payload, err := json.Marshal(map[string][]agentSuggestion{
		s.canvas.WorkflowID.String(): suggestions,
	})
	require.NoError(s.t, err)

	_, err = s.session.Page().Evaluate(`(payload) => {
		window.sessionStorage.setItem("agent-suggestions", payload);
	}`, string(payload))
	require.NoError(s.t, err)
}

func (s *agentSuggestionSteps) createCanvasWithoutVisit() {
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)
	canvas, _ := support.CreateCanvas(s.t, s.session.OrgID, user.ID, nil, nil)
	s.canvas.WorkflowID = canvas.ID

	err = database.Conn().
		Model(&models.Canvas{}).
		Where("id = ?", s.canvas.WorkflowID).
		Update("name", s.canvas.CanvasName).Error
	require.NoError(s.t, err)
}

func (s *agentSuggestionSteps) waitForAgentToggle() {
	toggle := q.TestID("canvas-tool-sidebar-toggle").Run(s.session)
	require.Eventually(s.t, func() bool {
		visible, err := toggle.IsVisible()
		return err == nil && visible
	}, agentWaitTimeout, agentPollInterval)
}

func (s *agentSuggestionSteps) assertSuggestionsBadgeCount(count int) {
	badge := q.TestID("agent-suggestions-badge").Run(s.session)
	require.Eventually(s.t, func() bool {
		visible, err := badge.IsVisible()
		if err != nil || !visible {
			return false
		}
		text, err := badge.TextContent()
		return err == nil && text == fmt.Sprintf("%d", count)
	}, agentWaitTimeout, agentPollInterval, "expected suggestions badge count %d", count)
}

func (s *agentSuggestionSteps) assertSuggestionVisible(id string) {
	locator := q.TestID("agent-suggestion-" + id).Run(s.session)
	require.Eventually(s.t, func() bool {
		visible, err := locator.IsVisible()
		return err == nil && visible
	}, agentWaitTimeout, agentPollInterval, "expected suggestion %s to be visible", id)
}

func (s *agentSuggestionSteps) assertSuggestionHidden(id string) {
	locator := q.TestID("agent-suggestion-" + id).Run(s.session)
	require.Eventually(s.t, func() bool {
		count, err := locator.Count()
		if err != nil {
			return false
		}
		if count == 0 {
			return true
		}
		visible, err := locator.IsVisible()
		return err == nil && !visible
	}, agentWaitTimeout, agentPollInterval, "expected suggestion %s to be hidden", id)
}

func (s *agentSuggestionSteps) clickSuggestion(id string) {
	// Hover the Agent control so the sticky card reopens if it was dismissed.
	trigger := q.TestID("agent-suggestions-trigger").Run(s.session)
	require.NoError(s.t, trigger.Hover())
	s.assertSuggestionVisible(id)
	s.session.Click(q.TestID("agent-suggestion-" + id))
}

func (s *agentSuggestionSteps) assertUserMessage(message string) {
	s.assertEventuallyVisible(q.Locator(fmt.Sprintf(`[data-testid="agent-user-message"]:has-text("%s")`, message)))
}

func (s *agentSuggestionSteps) assertAssistantMessage(message string) {
	s.assertEventuallyVisible(q.Locator(fmt.Sprintf(`[data-testid="agent-assistant-message"]:has-text("%s")`, message)))
}

func (s *agentSuggestionSteps) assertEventuallyVisible(query q.Query) {
	locator := query.Run(s.session)
	require.Eventually(s.t, func() bool {
		visible, err := locator.IsVisible()
		return err == nil && visible
	}, agentWaitTimeout, agentPollInterval)
}
