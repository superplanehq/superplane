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

func TestSendEmailComponent(t *testing.T) {
	t.Run("adding send email with user recipient", func(t *testing.T) {
		steps := &SendEmailSteps{t: t}
		steps.start()
		steps.givenACanvasExists("Send Email User")
		steps.addSendEmailWithUser("Notify User", "Test Subject", "Test Body")
		steps.assertSendEmailSavedToDB("Notify User", "Test Subject")
	})

	t.Run("running send email in a canvas flow", func(t *testing.T) {
		steps := &SendEmailSteps{t: t}
		steps.start()
		steps.givenSMTPSettingsExist()
		steps.givenCanvasWithManualTriggerSendEmailAndOutput()
		steps.runManualTrigger()
		steps.assertSendEmailExecutionFinished()
	})

	t.Run("send email fails when no email provider configured", func(t *testing.T) {
		steps := &SendEmailSteps{t: t}
		steps.start()
		steps.givenCanvasWithManualTriggerSendEmailAndOutput()
		steps.runManualTriggerAndWaitForFinish()
		steps.assertSendEmailExecutionFailed()
	})
}

type SendEmailSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *SendEmailSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *SendEmailSteps) givenACanvasExists(canvasName string) {
	s.canvas = shared.NewCanvasSteps(canvasName, s.t, s.session)
	s.canvas.Create()
}

func (s *SendEmailSteps) addSendEmailWithUser(nodeName, subject, body string) {
	s.canvas.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-sendemail")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, 500, 250)
	s.session.Sleep(500)

	s.session.FillIn(q.TestID("node-name-input"), nodeName)

	s.session.Click(q.TestID("field-type-select"))
	s.session.Click(q.Locator(`div[role="option"]:has-text("Specific user")`))

	s.session.Click(q.Locator(`button:has-text("Select user")`))
	s.session.Click(q.Locator(`div[role="option"]:has-text("e2e@superplane.local")`))

	s.session.FillIn(q.Locator("textarea[data-testid='string-field-subject']"), subject)

	s.typeIntoMonacoEditor(body)

	s.session.Click(q.TestID("save-node-button"))
	s.session.Sleep(500)
}

func (s *SendEmailSteps) typeIntoMonacoEditor(text string) {
	editor := q.Locator(".monaco-editor .view-lines")
	s.session.Click(editor)
	s.session.Sleep(200)

	if err := s.session.Page().Keyboard().Type(text, pw.KeyboardTypeOptions{}); err != nil {
		s.t.Fatalf("typing into monaco editor: %v", err)
	}
}

func (s *SendEmailSteps) assertSendEmailSavedToDB(nodeName, expectedSubject string) {
	node := s.canvas.GetNodeFromDB(nodeName)
	config := node.Configuration.Data()

	assert.Equal(s.t, expectedSubject, config["subject"])

	recipients, ok := config["recipients"].([]any)
	require.True(s.t, ok, "expected recipients to be a list")
	require.NotEmpty(s.t, recipients, "expected at least one recipient")

	firstRecipient, ok := recipients[0].(map[string]any)
	require.True(s.t, ok, "expected recipient to be a map")
	assert.Equal(s.t, "user", firstRecipient["type"])
	assert.NotEmpty(s.t, firstRecipient["user"], "expected user ID to be set")
}

func (s *SendEmailSteps) givenCanvasWithManualTriggerSendEmailAndOutput() {
	s.canvas = shared.NewCanvasSteps("Send Email Flow", s.t, s.session)
	s.canvas.Create()

	s.canvas.AddManualTrigger("Start", models.Position{X: 600, Y: 200})
	s.addSendEmailNode("Send Email", models.Position{X: 1000, Y: 200})
	s.canvas.AddNoop("Output", models.Position{X: 1400, Y: 200})

	s.canvas.Connect("Start", "Send Email")
	s.canvas.Connect("Send Email", "Output")

	s.canvas.Save()
}

func (s *SendEmailSteps) addSendEmailNode(nodeName string, pos models.Position) {
	s.canvas.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-sendemail")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(500)

	s.session.FillIn(q.TestID("node-name-input"), nodeName)

	s.session.Click(q.TestID("field-type-select"))
	s.session.Click(q.Locator(`div[role="option"]:has-text("Specific user")`))

	s.session.Click(q.Locator(`button:has-text("Select user")`))
	s.session.Click(q.Locator(`div[role="option"]:has-text("e2e@superplane.local")`))

	s.session.FillIn(q.Locator("textarea[data-testid='string-field-subject']"), "Test notification")

	s.typeIntoMonacoEditor("This is a test email body")

	s.session.Click(q.TestID("save-node-button"))
	s.session.Sleep(500)
}

func (s *SendEmailSteps) runManualTrigger() {
	s.canvas.RunManualTrigger("Start")
	s.canvas.WaitForExecution(
		"Send Email",
		models.CanvasNodeExecutionStateFinished,
		15*time.Second,
	)
}

func (s *SendEmailSteps) givenSMTPSettingsExist() {
	err := models.UpsertEmailSettings(&models.EmailSettings{
		Provider:      models.EmailProviderSMTP,
		SMTPHost:      "localhost",
		SMTPPort:      1025,
		SMTPFromName:  "SuperPlane Test",
		SMTPFromEmail: "test@superplane.local",
	})
	require.NoError(s.t, err)
}

func (s *SendEmailSteps) runManualTriggerAndWaitForFinish() {
	s.canvas.RunManualTrigger("Start")
	s.canvas.WaitForExecution(
		"Send Email",
		models.CanvasNodeExecutionStateFinished,
		15*time.Second,
	)
}

func (s *SendEmailSteps) assertSendEmailExecutionFinished() {
	sendEmailExecs := s.canvas.GetExecutionsForNode("Send Email")
	outputExecs := s.canvas.GetExecutionsForNode("Output")

	require.Len(s.t, sendEmailExecs, 1, "expected one execution for send email node")
	require.Len(s.t, outputExecs, 1, "expected one execution for output node")

	require.Equal(s.t, models.CanvasNodeExecutionStateFinished, sendEmailExecs[0].State)
	require.Equal(s.t, models.CanvasNodeExecutionStateFinished, outputExecs[0].State)
}

func (s *SendEmailSteps) assertSendEmailExecutionFailed() {
	sendEmailExecs := s.canvas.GetExecutionsForNode("Send Email")
	require.Len(s.t, sendEmailExecs, 1, "expected one execution for send email node")
	require.Equal(s.t, models.CanvasNodeExecutionStateFinished, sendEmailExecs[0].State)
	require.Equal(s.t, models.CanvasNodeExecutionResultFailed, sendEmailExecs[0].Result)
}
