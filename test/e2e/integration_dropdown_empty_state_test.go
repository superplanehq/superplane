package e2e

// -----------------------------------------------------------------------------
// Empty integration dropdown — desired behavior.
//
// Components with a FieldTypeIntegration configuration field (e.g. the
// "Run Claude Code" runner) render the Integration picker from the org's
// connected integrations (IntegrationFieldRenderer.tsx). Before the fix, zero
// matching integrations produced a popover with zero items: an empty ~128x10px
// sliver positioned below the viewport (at y=1440 on the 1440px-high test
// viewport), so the dropdown looked completely dead.
//
// With the fix (mirroring SecretPickerFieldRenderer):
//  1. a user who can create integrations sees a "Connect an integration" CTA
//     item, so the popover has content and opens on screen. On type-filtered
//     fields (claude credentials) the CTA opens IntegrationCreateDialog in
//     place — the canvas stays put and creation auto-selects the integration;
//  2. on unfiltered fields ("Environment from" -> Integration) there is no
//     single provider to create, so the CTA opens integration settings in a
//     NEW tab and the canvas URL does not change;
//  3. with a ready github integration connected, the unfiltered dropdown lists
//     it alongside the CTA.
// -----------------------------------------------------------------------------

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

const integrationConnectOptionSelector = `[role="listbox"] [data-testid="integration-picker-connect-option"]`
const secretAddNewOptionSelector = `[role="listbox"] [data-testid="secret-field-add-new-option"]`

func TestIntegrationDropdownEmptyState(t *testing.T) {
	t.Run("the empty claude-filtered credentials dropdown opens on screen with a connect CTA that opens the create dialog in place", func(t *testing.T) {
		steps := &integrationDropdownEmptyStateSteps{t: t}
		steps.start()
		steps.givenACanvasExists()
		steps.whenIAddAClaudeCodeRunnerBlock()
		steps.whenISetCredentialsSourceToIntegration()
		steps.whenIOpenTheCredentialsIntegrationDropdown()
		steps.thenTheDropdownPopoverIsOnScreenWithTheConnectCTA("integration-dropdown-fixed-1-cta-visible")
		steps.whenIClickTheConnectCTA()
		steps.thenTheCreateIntegrationDialogOpensInPlace("integration-dropdown-fixed-4-dialog-open")
		steps.whenISubmitTheClaudeDialogWithADummyAPIKey()
		steps.thenTheDialogClosesAndTheCreatedIntegrationIsSelected()
		steps.thenTheIntegrationExistsInErrorState("claude")
	})

	t.Run("the empty environment-from secret dropdown offers an add-new CTA", func(t *testing.T) {
		steps := &integrationDropdownEmptyStateSteps{t: t}
		steps.start()
		steps.givenACanvasExists()
		steps.whenIAddAClaudeCodeRunnerBlock()
		steps.whenIAddAnEnvironmentFromSource()
		steps.whenISwitchTheEnvironmentFromSourceToSecret()
		steps.whenIOpenTheEnvironmentFromSecretDropdown()
		steps.thenTheSecretDropdownPopoverIsOnScreenWithTheAddNewCTA("secret-dropdown-fixed-1-add-new-visible")
	})

	t.Run("the empty environment-from integration dropdown opens on screen with a connect CTA that opens settings in a new tab", func(t *testing.T) {
		steps := &integrationDropdownEmptyStateSteps{t: t}
		steps.start()
		steps.givenACanvasExists()
		steps.whenIAddAClaudeCodeRunnerBlock()
		steps.whenIAddAnEnvironmentFromSource()
		steps.whenIOpenTheEnvironmentFromIntegrationDropdown()
		steps.thenTheDropdownPopoverIsOnScreenWithTheConnectCTA("env-from-fixed-2-cta-visible")
		steps.thenClickingTheConnectCTAOpensIntegrationSettingsInANewTab()
	})

	t.Run("with a ready github integration the unfiltered dropdown lists it alongside the connect CTA", func(t *testing.T) {
		steps := &integrationDropdownEmptyStateSteps{t: t}
		steps.start()
		steps.givenAReadyGithubIntegrationExists("gh-e2e-integration")
		steps.givenACanvasExists()
		steps.whenIAddAClaudeCodeRunnerBlock()
		steps.whenIAddAnEnvironmentFromSource()
		steps.whenIOpenTheEnvironmentFromIntegrationDropdown()
		steps.thenTheDropdownListsTheIntegrationAndTheConnectCTA("gh-e2e-integration", "integration-dropdown-fixed-3-populated-lists-github")
	})
}

type integrationDropdownEmptyStateSteps struct {
	t         *testing.T
	session   *session.TestSession
	canvas    *shared.CanvasSteps
	canvasURL string
}

func (s *integrationDropdownEmptyStateSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

// givenAReadyGithubIntegrationExists reuses the established pattern from
// settings_permission_guards_test.go: create the installation row directly
// and mark it ready.
func (s *integrationDropdownEmptyStateSteps) givenAReadyGithubIntegrationExists(name string) {
	integration, err := models.CreateIntegration(uuid.New(), s.session.OrgID, "github", name, nil)
	require.NoError(s.t, err)
	require.NoError(s.t, database.Conn().Model(integration).Update("state", models.IntegrationStateReady).Error)
}

func (s *integrationDropdownEmptyStateSteps) givenACanvasExists() {
	s.canvas = shared.NewCanvasSteps("Integration Dropdown Empty State", s.t, s.session)
	s.canvas.Create()
	s.canvas.EnterEditMode()
}

func (s *integrationDropdownEmptyStateSteps) whenIAddAClaudeCodeRunnerBlock() {
	s.canvas.OpenBuildingBlockCategory("Runners")
	s.canvas.AddBuildingBlockByTestID("building-block-runnerclaudecode", models.Position{X: 700, Y: 250})

	// The component sidebar for the dropped block should now be open.
	s.session.AssertVisible(q.TestID("node-name-input"))
}

func (s *integrationDropdownEmptyStateSteps) whenISetCredentialsSourceToIntegration() {
	// credentials.source select (object sub-fields keep their bare names).
	sourceSelect := q.TestID("field-source-select")
	s.session.AssertVisible(sourceSelect)
	s.session.Click(sourceSelect)
	s.session.Click(q.Locator(`div[role="option"]:has-text("Integration")`))
	s.session.Sleep(300)
}

func (s *integrationDropdownEmptyStateSteps) whenIOpenTheCredentialsIntegrationDropdown() {
	s.openIntegrationDropdown(`[data-testid="integration-field-integration"] button`)
}

// whenIAddAnEnvironmentFromSource clicks "Add Source" under "Environment from",
// which adds a list item whose Source sub-select defaults to "Integration",
// making the item's Integration select visible immediately.
func (s *integrationDropdownEmptyStateSteps) whenIAddAnEnvironmentFromSource() {
	s.session.Click(q.Locator(`button:has-text("Add Source")`))
	s.session.AssertVisible(q.TestID("list-item-row"))
	s.session.Sleep(300)
}

func (s *integrationDropdownEmptyStateSteps) whenIOpenTheEnvironmentFromIntegrationDropdown() {
	s.openIntegrationDropdown(`[data-testid="list-item-row"] [data-testid="integration-field-integration"] button`)
}

func (s *integrationDropdownEmptyStateSteps) whenISwitchTheEnvironmentFromSourceToSecret() {
	itemSourceSelect := q.Locator(`[data-testid="list-item-row"] [data-testid="field-source-select"]`)
	s.session.AssertVisible(itemSourceSelect)
	s.session.Click(itemSourceSelect)
	s.session.Click(q.Locator(`div[role="option"]:has-text("Secret")`))
	s.session.Sleep(300)
}

func (s *integrationDropdownEmptyStateSteps) whenIOpenTheEnvironmentFromSecretDropdown() {
	field := q.Locator(`[data-testid="list-item-row"] [data-testid="secret-field-secret"] button`)
	s.session.AssertVisible(field)

	// Wait for the secrets query to finish loading (the trigger shows
	// "Loading secrets..." while pending).
	trigger := field.Run(s.session)
	require.Eventually(s.t, func() bool {
		text, err := trigger.InnerText()
		return err == nil && !strings.Contains(text, "Loading secrets")
	}, 15*time.Second, 200*time.Millisecond, "secret select should finish loading")

	require.NoError(s.t, trigger.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.Sleep(300)
}

// whenISubmitTheClaudeDialogWithADummyAPIKey fills the (sensitive, hence
// password-typed) API Key field with a bogus key and submits the dialog.
func (s *integrationDropdownEmptyStateSteps) whenISubmitTheClaudeDialogWithADummyAPIKey() {
	apiKeyInput := s.session.Page().Locator(`[role="dialog"] input[type="password"]`).First()
	require.NoError(s.t, apiKeyInput.Fill("sk-ant-dummy-e2e-key"))
	s.takeNamedScreenshot("claude-dialog-submit-1-filled")
	s.session.Click(q.Locator(`[role="dialog"] button:has-text("Connect")`))
}

// thenTheDialogClosesAndTheCreatedIntegrationIsSelected documents the server
// behavior with a bogus key: CreateIntegration runs the claude Sync (key
// verification) synchronously, stores the integration in state "error", but
// still returns success — so the dialog closes and the created installation is
// auto-selected in the field.
func (s *integrationDropdownEmptyStateSteps) thenTheDialogClosesAndTheCreatedIntegrationIsSelected() {
	dialog := s.session.Page().Locator(`[role="dialog"]`)
	require.Eventually(s.t, func() bool {
		count, err := dialog.Count()
		return err == nil && count == 0
	}, 45*time.Second, 500*time.Millisecond, "the create dialog should close after submit")

	trigger := s.session.Page().Locator(`[data-testid="integration-field-integration"] button`)
	require.Eventually(s.t, func() bool {
		text, err := trigger.InnerText()
		return err == nil && strings.Contains(text, "claude")
	}, 10*time.Second, 200*time.Millisecond, "the created integration should be auto-selected in the field")

	s.takeNamedScreenshot("claude-dialog-submit-2-auto-selected")
}

func (s *integrationDropdownEmptyStateSteps) thenTheIntegrationExistsInErrorState(name string) {
	require.Eventually(s.t, func() bool {
		integration, err := models.FindIntegrationByName(database.Conn(), s.session.OrgID, name)
		return err == nil && integration.State == models.IntegrationStateError && integration.StateDescription != ""
	}, 30*time.Second, 500*time.Millisecond,
		"the claude integration should exist in error state after the bogus-key sync")
}

func (s *integrationDropdownEmptyStateSteps) openIntegrationDropdown(triggerSelector string) {
	field := q.Locator(triggerSelector)
	s.session.AssertVisible(field)

	// Wait for the integrations query to finish loading (the trigger shows
	// "Loading integrations..." while pending).
	trigger := field.Run(s.session)
	require.Eventually(s.t, func() bool {
		text, err := trigger.InnerText()
		return err == nil && !strings.Contains(text, "Loading integrations")
	}, 15*time.Second, 200*time.Millisecond, "integration select should finish loading")

	require.NoError(s.t, trigger.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.Sleep(300)
}

// thenTheDropdownPopoverIsOnScreenWithTheConnectCTA asserts the fix: the open
// popover contains the connect CTA and sits fully inside the viewport. Before
// the fix the empty popover was a 128x10px sliver at (0, 1440) — below the
// 2560x1440 test viewport — so the dropdown appeared dead.
func (s *integrationDropdownEmptyStateSteps) thenTheDropdownPopoverIsOnScreenWithTheConnectCTA(screenshotName string) {
	s.assertPopoverOnScreenWithCTA(integrationConnectOptionSelector,
		"connect CTA should be visible in the open integration dropdown", screenshotName)
}

// thenTheSecretDropdownPopoverIsOnScreenWithTheAddNewCTA is the secret-field
// variant of the same assertion (SecretFieldRenderer had the identical bug).
func (s *integrationDropdownEmptyStateSteps) thenTheSecretDropdownPopoverIsOnScreenWithTheAddNewCTA(screenshotName string) {
	s.assertPopoverOnScreenWithCTA(secretAddNewOptionSelector,
		"add-new CTA should be visible in the open secret dropdown", screenshotName)
}

func (s *integrationDropdownEmptyStateSteps) assertPopoverOnScreenWithCTA(
	ctaSelector string,
	ctaDescription string,
	screenshotName string,
) {
	listbox := q.Locator(`[role="listbox"]`).Run(s.session)
	require.NoError(s.t, listbox.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(10000),
	}), "dropdown popover should open")

	cta := s.session.Page().Locator(ctaSelector)
	require.NoError(s.t, cta.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(10000),
	}), ctaDescription)

	s.takeNamedScreenshot(screenshotName)

	box, err := listbox.BoundingBox()
	require.NoError(s.t, err)
	require.NotNil(s.t, box)
	viewport := s.session.Page().ViewportSize()
	require.NotNil(s.t, viewport)
	s.t.Logf("INTEGRATION DROPDOWN POPOVER: %.0fx%.0f at (%.0f, %.0f), viewport %dx%d",
		box.Width, box.Height, box.X, box.Y, viewport.Width, viewport.Height)

	require.GreaterOrEqual(s.t, box.X, 0.0, "popover should not overflow the left viewport edge")
	require.GreaterOrEqual(s.t, box.Y, 0.0, "popover should not overflow the top viewport edge")
	require.LessOrEqual(s.t, box.X+box.Width, float64(viewport.Width), "popover should not overflow the right viewport edge")
	require.LessOrEqual(s.t, box.Y+box.Height, float64(viewport.Height), "popover should not overflow the bottom viewport edge")
	require.GreaterOrEqual(s.t, box.Height, 30.0, "popover should have real content, not a collapsed sliver")
}

func (s *integrationDropdownEmptyStateSteps) whenIClickTheConnectCTA() {
	s.canvasURL = s.session.Page().URL()
	s.session.Click(q.Locator(integrationConnectOptionSelector))
}

// thenTheCreateIntegrationDialogOpensInPlace asserts the type-filtered CTA
// opens IntegrationCreateDialog inside the canvas page: no navigation, and the
// component sidebar stays mounted behind the dialog.
func (s *integrationDropdownEmptyStateSteps) thenTheCreateIntegrationDialogOpensInPlace(screenshotName string) {
	dialog := s.session.Page().Locator(`[role="dialog"]:has-text("Configure Claude")`)
	require.NoError(s.t, dialog.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(10000),
	}), "the Configure Claude dialog should open in place")

	require.Equal(s.t, s.canvasURL, s.session.Page().URL(),
		"opening the create dialog should not navigate away from the canvas")
	s.session.AssertVisible(q.TestID("node-name-input"))

	s.takeNamedScreenshot(screenshotName)
}

// thenClickingTheConnectCTAOpensIntegrationSettingsInANewTab asserts the
// unfiltered CTA opens /:orgId/settings/integrations in a new tab while the
// canvas page stays put.
func (s *integrationDropdownEmptyStateSteps) thenClickingTheConnectCTAOpensIntegrationSettingsInANewTab() {
	canvasURL := s.session.Page().URL()

	popup, err := s.session.Page().ExpectPopup(func() error {
		return s.session.Page().Locator(integrationConnectOptionSelector).Click()
	})
	require.NoError(s.t, err, "clicking the connect CTA should open a new tab")
	require.NoError(s.t, popup.WaitForLoadState())
	require.True(s.t, strings.HasSuffix(popup.URL(), "/settings/integrations"),
		"new tab should open integration settings, got %s", popup.URL())
	s.takeNamedScreenshotOfPage(popup, "env-from-fixed-3-settings-new-tab")
	require.NoError(s.t, popup.Close())

	require.Equal(s.t, canvasURL, s.session.Page().URL(), "the canvas page should stay put")
}

func (s *integrationDropdownEmptyStateSteps) thenTheDropdownListsTheIntegrationAndTheConnectCTA(name string, screenshotName string) {
	listbox := q.Locator(`[role="listbox"]`).Run(s.session)
	require.NoError(s.t, listbox.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(10000),
	}), "integration dropdown popover should open")

	// Exactly two options: the ready github integration and the connect CTA.
	options := s.session.Page().Locator(`[role="listbox"] [role="option"]`)
	require.Eventually(s.t, func() bool {
		count, err := options.Count()
		return err == nil && count == 2
	}, 10*time.Second, 200*time.Millisecond, "dropdown should list the ready github integration and the connect CTA")

	text, err := listbox.InnerText()
	require.NoError(s.t, err)
	require.Contains(s.t, text, name)

	cta := s.session.Page().Locator(integrationConnectOptionSelector)
	visible, err := cta.IsVisible()
	require.NoError(s.t, err)
	require.True(s.t, visible, "connect CTA should be listed after the existing integrations")

	s.takeNamedScreenshot(screenshotName)
}

func (s *integrationDropdownEmptyStateSteps) takeNamedScreenshot(name string) {
	s.takeNamedScreenshotOfPage(s.session.Page(), name)
}

func (s *integrationDropdownEmptyStateSteps) takeNamedScreenshotOfPage(page pw.Page, name string) {
	path := fmt.Sprintf("/app/tmp/screenshots/%s.png", name)
	s.t.Logf("Taking screenshot: %s", path)
	if _, err := page.Screenshot(pw.PageScreenshotOptions{
		Path:     pw.String(path),
		FullPage: pw.Bool(true),
		Type:     pw.ScreenshotTypePng,
	}); err != nil {
		s.t.Logf("screenshot error: %v", err)
	}
}
