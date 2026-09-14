package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestWebhookResetSecret(t *testing.T) {
	t.Run("reset webhook secret shows new key", func(t *testing.T) {
		steps := &WebhookResetSteps{t: t}
		steps.start()
		steps.givenACanvasWithWebhook("Webhook Reset Canvas", "Webhook")
		steps.openWebhookConfiguration("Webhook")
		steps.waitForWebhookURL()
		steps.resetWebhookSecret()
		steps.assertNewSecretVisible()
	})
}

type WebhookResetSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *WebhookResetSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *WebhookResetSteps) givenACanvasWithWebhook(canvasName, nodeName string) {
	s.canvas = shared.NewCanvasSteps(canvasName, s.t, s.session)
	s.canvas.Create()
	s.canvas.EnterEditMode()
	s.addWebhookTrigger(nodeName, models.Position{X: 500, Y: 200})
	s.canvas.Save()
	s.canvas.CommitAndPublish()
}

func (s *WebhookResetSteps) addWebhookTrigger(name string, pos models.Position) {
	s.canvas.AddBuildingBlockByTestID("building-block-webhook", pos)
	s.session.WaitForEnabled(q.TestID("node-name-input"))
	s.session.FillIn(q.TestID("node-name-input"), name)
}

func (s *WebhookResetSteps) openWebhookConfiguration(nodeName string) {
	s.canvas.EnterEditMode()
	s.canvas.StartEditingNode(nodeName)
	s.session.Click(q.Text("Configuration"))
	s.session.WaitUntil(func() bool {
		return s.webhookURLValue() != ""
	}, "webhook URL field did not appear")
}

func (s *WebhookResetSteps) webhookURLValue() string {
	value, err := s.session.Page().Locator("#webhook-url-input").InputValue()
	if err != nil {
		return ""
	}
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "URL GENERATED") {
		return ""
	}
	return value
}

func (s *WebhookResetSteps) waitForWebhookURL() string {
	var value string
	require.Eventually(s.t, func() bool {
		value = s.webhookURLValue()
		return strings.HasPrefix(value, "http")
	}, 20*time.Second, 200*time.Millisecond, "timed out waiting for webhook URL")
	return value
}

func (s *WebhookResetSteps) resetWebhookSecret() {
	s.session.Click(q.Text("Reset Signature Key"))
}

func (s *WebhookResetSteps) assertNewSecretVisible() {
	s.session.AssertText("New signature key generated")
	secretBlock := q.Locator(`div:has-text("New signature key generated") pre`)
	loc := secretBlock.Run(s.session)
	value, err := loc.TextContent()
	require.NoError(s.t, err)
	require.NotEmpty(s.t, strings.TrimSpace(value))
}
