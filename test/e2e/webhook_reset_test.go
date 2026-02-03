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
	steps := &WebhookResetSteps{t: t}

	t.Run("reset webhook secret shows new key", func(t *testing.T) {
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
	s.addWebhookTrigger(nodeName, models.Position{X: 500, Y: 200})
	s.canvas.Save()
}

func (s *WebhookResetSteps) addWebhookTrigger(name string, pos models.Position) {
	s.canvas.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-webhook")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(500)

	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.Click(q.TestID("save-node-button"))
	s.session.Sleep(500)
}

func (s *WebhookResetSteps) openWebhookConfiguration(nodeName string) {
	s.canvas.StartEditingNode(nodeName)
	s.session.Click(q.Text("Configuration"))
	s.session.Sleep(200)
}

func (s *WebhookResetSteps) waitForWebhookURL() string {
	start := time.Now()
	input := q.Locator(`label:has-text("Webhook URL") + div input[type="text"]`)

	for time.Since(start) < 20*time.Second {
		loc := input.Run(s.session)
		value, err := loc.InputValue()
		if err == nil && strings.TrimSpace(value) != "" && !strings.Contains(value, "URL GENERATED") {
			return value
		}
		s.session.Sleep(500)
	}

	s.t.Fatalf("timed out waiting for webhook URL")
	return ""
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
