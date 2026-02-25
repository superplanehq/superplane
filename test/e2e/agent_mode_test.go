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

	t.Run("setting up agent mode", func(t *testing.T) {
		steps.givenILogIntoANewOrganization()
		steps.whenIVisitGeneralSettings()
		steps.thenISeeTheAgentModeCard()
		steps.thenISeeTheSetupAction()
		steps.whenIClickSetup()
		steps.thenISeeSetupModal()
		steps.whenIEnterOpenAIKey("sk-test-agent-mode-e2e-key-1234")
		steps.whenISaveAgentModeKey()
		steps.thenSetupModalCloses()
		steps.thenAgentModeIsEnabledInDatabase()
		steps.thenAgentKeyIsStoredWithLast4("1234")
		steps.thenISeeDisableAction()
	})

	t.Run("updating the api key", func(t *testing.T) {
		steps.givenILogIntoANewOrganization()
		steps.whenIVisitGeneralSettings()
		steps.whenIClickSetup()
		steps.whenIEnterOpenAIKey("sk-test-agent-mode-e2e-key-1234")
		steps.whenISaveAgentModeKey()
		steps.thenAgentKeyIsStoredWithLast4("1234")

		steps.whenIClickSetup()
		steps.thenISeeConfigureModal()
		steps.whenIEnterOpenAIKey("sk-test-agent-mode-e2e-key-5678")
		steps.whenISaveAgentModeKey()
		steps.thenAgentKeyIsStoredWithLast4("5678")
		steps.thenAgentModeIsEnabledInDatabase()
	})

	t.Run("disabling the agent mode", func(t *testing.T) {
		steps.givenILogIntoANewOrganization()
		steps.whenIVisitGeneralSettings()
		steps.whenIClickSetup()
		steps.whenIEnterOpenAIKey("sk-test-agent-mode-e2e-key-1234")
		steps.whenISaveAgentModeKey()
		steps.thenAgentModeIsEnabledInDatabase()
		steps.thenAgentKeyIsStoredWithLast4("1234")
		steps.whenIClickDisable()
		steps.thenAgentModeIsDisabledInDatabase()
		steps.thenNoAgentKeyIsStoredInDatabase()
		steps.thenISeeTheSetupAction()
	})
}

type agentModeSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *agentModeSteps) givenILogIntoANewOrganization() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *agentModeSteps) whenIVisitGeneralSettings() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/general")
	s.session.Sleep(500)
}

func (s *agentModeSteps) thenISeeTheAgentModeCard() {
	page := s.session.Page()
	card := page.GetByTestId("agent-mode-settings-card")
	err := card.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible})
	require.NoError(s.t, err)
}

func (s *agentModeSteps) thenISeeTheSetupAction() {
	page := s.session.Page()
	setup := page.GetByTestId("agent-mode-setup-button")
	err := setup.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible})
	require.NoError(s.t, err)
}

func (s *agentModeSteps) thenISeeDisableAction() {
	page := s.session.Page()
	disable := page.GetByTestId("agent-mode-disable-button")
	err := disable.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible})
	require.NoError(s.t, err)
}

func (s *agentModeSteps) whenIClickSetup() {
	page := s.session.Page()
	setup := page.GetByTestId("agent-mode-setup-button")
	if count, countErr := setup.Count(); countErr == nil && count > 0 {
		err := setup.Click()
		require.NoError(s.t, err)
		s.session.Sleep(500)
		return
	}

	update := page.GetByTestId("agent-mode-update-key-button")
	err := update.Click()
	require.NoError(s.t, err)
	s.session.Sleep(500)
}

func (s *agentModeSteps) whenIClickDisable() {
	page := s.session.Page()
	disable := page.GetByTestId("agent-mode-disable-button")
	err := disable.Click()
	require.NoError(s.t, err)
	s.session.Sleep(500)
}

func (s *agentModeSteps) thenISeeSetupModal() {
	page := s.session.Page()
	modalTitle := page.GetByText("Set up Agent Mode", pw.PageGetByTextOptions{Exact: pw.Bool(true)})
	err := modalTitle.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible})
	require.NoError(s.t, err)
}

func (s *agentModeSteps) thenISeeConfigureModal() {
	page := s.session.Page()
	modalTitle := page.GetByText("Configure Agent Mode", pw.PageGetByTextOptions{Exact: pw.Bool(true)})
	err := modalTitle.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible})
	require.NoError(s.t, err)
}

func (s *agentModeSteps) thenSetupModalCloses() {
	page := s.session.Page()
	modalTitle := page.GetByText("Set up Agent Mode", pw.PageGetByTextOptions{Exact: pw.Bool(true)})
	// Title may change in configured state, so hidden is robust for setup completion.
	err := modalTitle.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateHidden, Timeout: pw.Float(5000)})
	require.NoError(s.t, err)
}

func (s *agentModeSteps) whenIEnterOpenAIKey(apiKey string) {
	page := s.session.Page()
	err := page.GetByTestId("agent-openai-key-input").Fill(apiKey)
	require.NoError(s.t, err)
}

func (s *agentModeSteps) whenISaveAgentModeKey() {
	page := s.session.Page()
	err := page.GetByTestId("agent-openai-key-save").Click()
	require.NoError(s.t, err)
	s.session.Sleep(1000)
}

func (s *agentModeSteps) thenAgentModeIsDisabledInDatabase() {
	settings, err := models.FindOrganizationAgentSettingsByOrganizationID(s.session.OrgID.String())
	if err == nil {
		assert.False(s.t, settings.AgentModeEnabled)
		return
	}

	// If no settings row exists yet, that is also a valid zero state.
	require.Error(s.t, err)
}

func (s *agentModeSteps) thenAgentModeIsEnabledInDatabase() {
	settings, err := models.FindOrganizationAgentSettingsByOrganizationID(s.session.OrgID.String())
	require.NoError(s.t, err)
	assert.True(s.t, settings.AgentModeEnabled)
}

func (s *agentModeSteps) thenNoAgentKeyIsStoredInDatabase() {
	settings, err := models.FindOrganizationAgentSettingsByOrganizationID(s.session.OrgID.String())
	require.NoError(s.t, err)
	assert.Empty(s.t, settings.OpenAIApiKeyCiphertext)
	assert.Nil(s.t, settings.OpenAIKeyEncryptionKeyID)
	assert.Nil(s.t, settings.OpenAIKeyLast4)
}

func (s *agentModeSteps) thenAgentKeyIsStoredWithLast4(last4 string) {
	settings, err := models.FindOrganizationAgentSettingsByOrganizationID(s.session.OrgID.String())
	require.NoError(s.t, err)
	assert.NotEmpty(s.t, settings.OpenAIApiKeyCiphertext)
	assert.NotNil(s.t, settings.OpenAIKeyEncryptionKeyID)
	require.NotNil(s.t, settings.OpenAIKeyLast4)
	assert.Equal(s.t, last4, *settings.OpenAIKeyLast4)
}

func (s *agentModeSteps) thenOpenAIKeyInputIsHidden() {
	page := s.session.Page()
	input := page.GetByTestId("agent-openai-key-input")
	err := input.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateHidden, Timeout: pw.Float(2000)})
	require.NoError(s.t, err)
}

func (s *agentModeSteps) thenOpenAIKeyInputIsVisible() {
	page := s.session.Page()
	input := page.GetByTestId("agent-openai-key-input")
	err := input.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible})
	require.NoError(s.t, err)
}
