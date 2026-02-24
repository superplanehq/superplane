package e2e

import (
	"testing"

	pw "github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestAgentMode(t *testing.T) {
	steps := &agentModeSteps{t: t}

	t.Run("configure button opens OpenAI key controls", func(t *testing.T) {
		steps.start()
		steps.visitGeneralPage()
		steps.assertAgentModeCardVisible()
		steps.assertOpenAIKeyInputHidden()
		steps.clickConfigureAgentMode()
		steps.assertConfigureModalVisible()
		steps.assertOpenAIKeyInputVisible()
		steps.assertAgentModeSettingsPersisted(false)
		steps.assertNoCredentialPersisted()
	})

	t.Run("saving OpenAI key persists credential and status", func(t *testing.T) {
		steps.start()
		steps.visitGeneralPage()
		steps.clickConfigureAgentMode()
		steps.assertConfigureModalVisible()
		steps.fillOpenAIKey("sk-test-agent-mode-e2e-key-1234")
		steps.saveOpenAIKey()
		steps.assertCredentialPersisted("1234")
		steps.assertKeyStatusVisible()
	})
}

type agentModeSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *agentModeSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *agentModeSteps) visitGeneralPage() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/general")
	s.session.Sleep(500)
}

func (s *agentModeSteps) assertAgentModeCardVisible() {
	page := s.session.Page()
	card := page.GetByTestId("agent-mode-settings-card")
	err := card.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible})
	require.NoError(s.t, err)
}

func (s *agentModeSteps) clickConfigureAgentMode() {
	page := s.session.Page()
	configure := page.GetByTestId("agent-mode-configure-button")
	err := configure.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible})
	require.NoError(s.t, err)

	err = configure.Click()
	require.NoError(s.t, err)
	s.session.Sleep(500)
}

func (s *agentModeSteps) assertConfigureModalVisible() {
	page := s.session.Page()
	modalTitle := page.GetByText("Configure Agent Mode", pw.PageGetByTextOptions{Exact: pw.Bool(true)})
	err := modalTitle.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible})
	require.NoError(s.t, err)
}

func (s *agentModeSteps) assertOpenAIKeyInputHidden() {
	page := s.session.Page()
	input := page.GetByTestId("agent-openai-key-input")
	err := input.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateHidden, Timeout: pw.Float(2000)})
	require.NoError(s.t, err)
}

func (s *agentModeSteps) assertOpenAIKeyInputVisible() {
	page := s.session.Page()
	input := page.GetByTestId("agent-openai-key-input")
	err := input.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible})
	require.NoError(s.t, err)
}

func (s *agentModeSteps) fillOpenAIKey(apiKey string) {
	page := s.session.Page()
	err := page.GetByTestId("agent-openai-key-input").Fill(apiKey)
	require.NoError(s.t, err)
}

func (s *agentModeSteps) saveOpenAIKey() {
	page := s.session.Page()
	err := page.GetByTestId("agent-openai-key-save").Click()
	require.NoError(s.t, err)
	s.session.Sleep(1000)
}

func (s *agentModeSteps) assertAgentModeSettingsPersisted(enabled bool) {
	settings, err := models.FindOrganizationAgentSettingsByOrganizationID(s.session.OrgID.String())
	require.NoError(s.t, err)
	assert.Equal(s.t, enabled, settings.AgentModeEnabled)
}

func (s *agentModeSteps) assertNoCredentialPersisted() {
	_, err := models.FindOrganizationAgentCredentialByOrganizationID(s.session.OrgID.String())
	require.Error(s.t, err)
}

func (s *agentModeSteps) assertCredentialPersisted(last4 string) {
	credential, err := models.FindOrganizationAgentCredentialByOrganizationID(s.session.OrgID.String())
	require.NoError(s.t, err)
	assert.Equal(s.t, models.OrganizationAgentCredentialProviderOpenAI, credential.Provider)
	assert.Equal(s.t, last4, credential.KeyLast4)

	settings, err := models.FindOrganizationAgentSettingsByOrganizationID(s.session.OrgID.String())
	require.NoError(s.t, err)
	assert.True(s.t, settings.AgentModeEnabled)
	assert.Contains(
		s.t,
		[]string{
			models.OrganizationAgentOpenAIKeyStatusValid,
			models.OrganizationAgentOpenAIKeyStatusInvalid,
			models.OrganizationAgentOpenAIKeyStatusUnchecked,
		},
		settings.OpenAIKeyStatus,
	)
}

func (s *agentModeSteps) assertKeyStatusVisible() {
	page := s.session.Page()
	status := page.GetByTestId("agent-openai-key-status")
	err := status.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible})
	require.NoError(s.t, err)
}
