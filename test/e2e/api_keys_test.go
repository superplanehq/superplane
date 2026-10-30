package e2e

import (
	"net/http"
	"slices"
	"testing"
	"time"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestAPIKeys(t *testing.T) {
	t.Run("operator key can be created edited rotated and revoked", func(t *testing.T) {
		steps := &apiKeySteps{t: t}
		steps.start()
		steps.visitAPIKeysPage()
		steps.clickCreateAPIKey()
		steps.fillName("ci-deploy-bot")
		steps.fillDescription("Deploys from CI")
		steps.selectRole("Operator")
		steps.submitCreate()
		originalToken := steps.readDisplayedToken()
		steps.assertTokenStatus(originalToken, http.StatusOK)
		steps.dismissTokenModal()
		steps.assertAPIKeySavedInDB("ci-deploy-bot", "Deploys from CI", models.RoleOrgOperator)

		steps.visitAPIKeysPage()
		steps.assertCreatorVisibleInListForAPIKey("ci-deploy-bot")
		steps.clickAPIKeyLink("ci-deploy-bot")
		steps.assertOnDetailPage("ci-deploy-bot")
		steps.assertCreatorOnDetailPage()
		steps.clickEditButton()
		steps.clearAndFillEditName("edited-bot")
		steps.clearAndFillEditDescription("Updated description")
		steps.submitEdit()
		steps.assertAPIKeyUpdatedInDB("edited-bot", "Updated description")

		steps.clickRegenerateToken()
		replacementToken := steps.readDisplayedToken()
		steps.assertTokenStatus(originalToken, http.StatusUnauthorized)
		steps.assertTokenStatus(replacementToken, http.StatusOK)
		steps.dismissTokenModal()
		steps.clickDeleteOnDetail()
		steps.assertAPIKeyDeletedFromDB("edited-bot")
		steps.assertTokenStatus(replacementToken, http.StatusUnauthorized)
	})

	t.Run("creating an API key with admin role", func(t *testing.T) {
		steps := &apiKeySteps{t: t}
		steps.start()
		steps.visitAPIKeysPage()
		steps.clickCreateAPIKey()
		steps.fillName("admin-bot")
		steps.fillDescription("Admin automation")
		steps.selectRole("Admin")
		steps.submitCreate()
		steps.readDisplayedToken()
		steps.dismissTokenModal()
		steps.assertAPIKeySavedInDB("admin-bot", "Admin automation", models.RoleOrgAdmin)
	})
}

type apiKeySteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *apiKeySteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	require.NoError(s.t, models.EnableExperimentalFeature(s.session.OrgID, features.FeatureFactories))
	s.session.Login()
}

func (s *apiKeySteps) visitAPIKeysPage() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/api-keys")
}

func (s *apiKeySteps) clickCreateAPIKey() {
	page := s.session.Page()
	createBtn := page.GetByTestId("api-key-create-btn")
	err := createBtn.First().Click()
	require.NoError(s.t, err)
}

func (s *apiKeySteps) fillName(name string) {
	page := s.session.Page()
	err := page.GetByTestId("api-key-create-name").Fill(name)
	require.NoError(s.t, err)
}

func (s *apiKeySteps) fillDescription(description string) {
	page := s.session.Page()
	err := page.GetByTestId("api-key-create-description").Fill(description)
	require.NoError(s.t, err)
}

func (s *apiKeySteps) selectRole(roleLabel string) {
	page := s.session.Page()

	trigger := page.GetByTestId("api-key-create-role")
	err := trigger.Click()
	require.NoError(s.t, err)

	option := page.GetByRole("option", pw.PageGetByRoleOptions{Name: roleLabel, Exact: pw.Bool(true)})
	err = option.Click()
	require.NoError(s.t, err)
}

func (s *apiKeySteps) submitCreate() {
	page := s.session.Page()
	err := page.GetByTestId("api-key-create-submit").Click()
	require.NoError(s.t, err)
}

func (s *apiKeySteps) readDisplayedToken() string {
	tokenInput := s.session.Page().GetByTestId("api-key-token-display")
	require.NoError(s.t, tokenInput.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible}))
	value, err := tokenInput.InputValue()
	require.NoError(s.t, err)
	require.NotEmpty(s.t, value, "token should not be empty")
	return value
}

func (s *apiKeySteps) assertTokenStatus(token string, expectedStatus int) {
	request, err := http.NewRequestWithContext(s.t.Context(), http.MethodGet, s.session.BaseURL+"/api/v1/canvases", nil)
	require.NoError(s.t, err)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("X-Organization-Id", s.session.OrgID.String())
	// A separate client has no browser cookies that could authenticate a broken token.
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	require.NoError(s.t, err)
	defer response.Body.Close()
	require.Equal(s.t, expectedStatus, response.StatusCode)
}

func (s *apiKeySteps) dismissTokenModal() {
	page := s.session.Page()
	err := page.GetByTestId("api-key-token-done").Click()
	require.NoError(s.t, err)
}

func (s *apiKeySteps) assertAPIKeySavedInDB(name, description, expectedRole string) {
	orgID := s.session.OrgID.String()
	apiKeys, err := models.FindAPIKeysByOrganization(database.DB(s.t.Context()), orgID)
	require.NoError(s.t, err)

	var found *models.User
	for i := range apiKeys {
		if apiKeys[i].Name == name {
			found = &apiKeys[i]
			break
		}
	}
	require.NotNil(s.t, found, "API key %q should exist in DB", name)
	require.Equal(s.t, models.UserTypeAPIKey, found.Type)
	require.NotNil(s.t, found.Description)
	require.Equal(s.t, description, *found.Description)
	require.NotEmpty(s.t, found.TokenHash, "token hash should be set")

	authService, err := authorization.NewAuthService()
	require.NoError(s.t, err)
	roles, err := authService.GetUserRolesForOrg(s.t.Context(), found.ID.String(), orgID)
	require.NoError(s.t, err)
	require.True(s.t, slices.ContainsFunc(roles, func(role *authorization.RoleDefinition) bool {
		return role.Name == expectedRole
	}), "API key should have role %s", expectedRole)
}

func (s *apiKeySteps) assertCreatorVisibleInListForAPIKey(name string) {
	// Same row as the API key name should show the human who created it (e2e seed user).
	row := s.session.Page().GetByRole("row", pw.PageGetByRoleOptions{Name: name})
	require.NoError(s.t, row.GetByText("E2E User", pw.LocatorGetByTextOptions{Exact: pw.Bool(true)}).WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(10000),
	}))
}

func (s *apiKeySteps) assertCreatorOnDetailPage() {
	s.session.AssertText("Created by")
	s.session.AssertText("E2E User")
}

func (s *apiKeySteps) clickAPIKeyLink(name string) {
	page := s.session.Page()
	link := page.GetByTestId("api-key-link").GetByText(name, pw.LocatorGetByTextOptions{Exact: pw.Bool(true)})
	err := link.Click()
	require.NoError(s.t, err)
}

func (s *apiKeySteps) assertOnDetailPage(name string) {
	s.session.AssertText(name)
	s.session.AssertText("API Token")
}

func (s *apiKeySteps) clickEditButton() {
	page := s.session.Page()
	err := page.GetByTestId("api-key-detail-edit").Click()
	require.NoError(s.t, err)
}

func (s *apiKeySteps) clearAndFillEditName(name string) {
	page := s.session.Page()
	input := page.GetByTestId("api-key-detail-edit-name")
	err := input.Fill(name)
	require.NoError(s.t, err)
}

func (s *apiKeySteps) clearAndFillEditDescription(description string) {
	page := s.session.Page()
	input := page.GetByTestId("api-key-detail-edit-description")
	err := input.Fill(description)
	require.NoError(s.t, err)
}

func (s *apiKeySteps) submitEdit() {
	page := s.session.Page()
	saveBtn := page.Locator("button:has-text('Save')").First()
	err := saveBtn.Click()
	require.NoError(s.t, err)
}

func (s *apiKeySteps) assertAPIKeyUpdatedInDB(name, description string) {
	require.Eventually(s.t, func() bool {
		apiKeys, err := models.FindAPIKeysByOrganization(database.DB(s.t.Context()), s.session.OrgID.String())
		if err != nil {
			return false
		}
		for _, apiKey := range apiKeys {
			if apiKey.Name == name && apiKey.Description != nil && *apiKey.Description == description {
				return true
			}
		}
		return false
	}, 10*time.Second, 100*time.Millisecond, "API key update should persist")
}

func (s *apiKeySteps) clickDeleteOnDetail() {
	page := s.session.Page()
	err := page.GetByTestId("api-key-detail-delete").Click()
	require.NoError(s.t, err)
}

func (s *apiKeySteps) assertAPIKeyDeletedFromDB(name string) {
	require.Eventually(s.t, func() bool {
		apiKeys, err := models.FindAPIKeysByOrganization(database.DB(s.t.Context()), s.session.OrgID.String())
		if err != nil {
			return false
		}
		for _, apiKey := range apiKeys {
			if apiKey.Name == name {
				return false
			}
		}
		return true
	}, 10*time.Second, 100*time.Millisecond, "API key should be deleted")
}

func (s *apiKeySteps) clickRegenerateToken() {
	page := s.session.Page()
	err := page.GetByTestId("api-key-detail-regenerate-token").Click()
	require.NoError(s.t, err)
}
