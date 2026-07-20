package e2e

import (
	"testing"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
)

func TestAPIKeys(t *testing.T) {
	t.Run("creating an API key with viewer role", func(t *testing.T) {
		steps := &apiKeySteps{t: t}
		steps.start()
		steps.visitAPIKeysPage()
		steps.clickCreateAPIKey()
		steps.fillName("ci-deploy-bot")
		steps.fillDescription("Deploys from CI")
		steps.selectRole("Viewer")
		steps.submitCreate()
		steps.assertTokenDisplayed()
		steps.dismissTokenModal()
		steps.assertAPIKeySavedInDB("ci-deploy-bot", "Deploys from CI", models.RoleOrgViewer)
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
		steps.assertTokenDisplayed()
		steps.dismissTokenModal()
		steps.assertAPIKeySavedInDB("admin-bot", "Admin automation", models.RoleOrgAdmin)
	})

	t.Run("viewing API keys in the list", func(t *testing.T) {
		steps := &apiKeySteps{t: t}
		steps.start()
		steps.givenAPIKeyExists("list-test-bot", "For listing test")
		steps.visitAPIKeysPage()
		steps.assertAPIKeyVisibleInList("list-test-bot")
		steps.assertCreatorVisibleInListForAPIKey("list-test-bot")
	})

	t.Run("navigating to API key detail", func(t *testing.T) {
		steps := &apiKeySteps{t: t}
		steps.start()
		steps.givenAPIKeyExists("detail-test-bot", "For detail test")
		steps.visitAPIKeysPage()
		steps.clickAPIKeyLink("detail-test-bot")
		steps.assertOnDetailPage("detail-test-bot")
		steps.assertCreatorOnDetailPage()
	})

	t.Run("editing an API key", func(t *testing.T) {
		steps := &apiKeySteps{t: t}
		steps.start()
		steps.givenAPIKeyExists("edit-test-bot", "Original description")
		steps.visitAPIKeysPage()
		steps.clickAPIKeyLink("edit-test-bot")
		steps.clickEditButton()
		steps.clearAndFillEditName("edited-bot")
		steps.clearAndFillEditDescription("Updated description")
		steps.submitEdit()
		steps.assertAPIKeyNameInDB("edited-bot")
	})

	t.Run("deleting an API key", func(t *testing.T) {
		steps := &apiKeySteps{t: t}
		steps.start()
		steps.givenAPIKeyExists("delete-test-bot", "Will be deleted")
		steps.visitAPIKeysPage()
		steps.assertAPIKeyVisibleInList("delete-test-bot")
		steps.clickAPIKeyLink("delete-test-bot")
		steps.clickDeleteOnDetail()
		steps.assertAPIKeyDeletedFromDB("delete-test-bot")
	})

	t.Run("regenerating an API key token", func(t *testing.T) {
		steps := &apiKeySteps{t: t}
		steps.start()
		steps.givenAPIKeyExists("regen-test-bot", "Token regen test")
		steps.visitAPIKeysPage()
		steps.clickAPIKeyLink("regen-test-bot")
		steps.clickRegenerateToken()
		steps.assertTokenDisplayed()
	})

	t.Run("viewer cannot create or manage API keys", func(t *testing.T) {
		steps := &apiKeySteps{t: t}
		steps.start()
		steps.givenAPIKeyExists("viewer-test-bot", "Viewer RBAC test")
		steps.loginAsViewer()
		steps.visitAPIKeysPage()
		steps.assertCreateButtonDisabled()
		steps.clickAPIKeyLink("viewer-test-bot")
		steps.assertEditButtonDisabled()
		steps.assertDeleteButtonDisabled()
	})
}

type apiKeySteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *apiKeySteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *apiKeySteps) visitAPIKeysPage() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/api-keys")
	s.session.Sleep(500)
}

func (s *apiKeySteps) clickCreateAPIKey() {
	page := s.session.Page()
	createBtn := page.GetByTestId("api-key-create-btn")
	err := createBtn.First().Click()
	require.NoError(s.t, err)
	s.session.Sleep(500)
}

func (s *apiKeySteps) fillName(name string) {
	page := s.session.Page()
	err := page.GetByTestId("api-key-create-name").Fill(name)
	require.NoError(s.t, err)
	s.session.Sleep(200)
}

func (s *apiKeySteps) fillDescription(description string) {
	page := s.session.Page()
	err := page.GetByTestId("api-key-create-description").Fill(description)
	require.NoError(s.t, err)
	s.session.Sleep(200)
}

func (s *apiKeySteps) selectRole(roleLabel string) {
	page := s.session.Page()

	trigger := page.GetByTestId("api-key-create-role")
	err := trigger.Click()
	require.NoError(s.t, err)
	s.session.Sleep(300)

	option := page.GetByRole("option", pw.PageGetByRoleOptions{Name: roleLabel, Exact: pw.Bool(true)})
	err = option.Click()
	require.NoError(s.t, err)
	s.session.Sleep(300)
}

func (s *apiKeySteps) submitCreate() {
	page := s.session.Page()
	err := page.GetByTestId("api-key-create-submit").Click()
	require.NoError(s.t, err)
	s.session.Sleep(1000)
}

func (s *apiKeySteps) assertTokenDisplayed() {
	page := s.session.Page()
	tokenInput := page.GetByTestId("api-key-token-display")
	err := tokenInput.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(5000)})
	require.NoError(s.t, err)

	value, err := tokenInput.InputValue()
	require.NoError(s.t, err)
	require.NotEmpty(s.t, value, "token should not be empty")
}

func (s *apiKeySteps) dismissTokenModal() {
	page := s.session.Page()
	err := page.GetByTestId("api-key-token-done").Click()
	require.NoError(s.t, err)
	s.session.Sleep(500)
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

	// Verify the role was assigned correctly via casbin
	var casbinRule struct {
		V0 string
		V1 string
	}
	err = database.Conn().
		Table("casbin_rule").
		Select("v0, v1").
		Where("ptype = 'g' AND v0 = ? AND v2 LIKE ?", "/users/"+found.ID.String(), "/org/%").
		First(&casbinRule).Error
	require.NoError(s.t, err)
	require.Equal(s.t, "/roles/"+expectedRole, casbinRule.V1)
}

func (s *apiKeySteps) assertAPIKeyVisibleInList(name string) {
	s.session.AssertText(name)
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
	s.session.Sleep(500)
}

func (s *apiKeySteps) assertOnDetailPage(name string) {
	s.session.AssertText(name)
	s.session.AssertText("API Token")
}

func (s *apiKeySteps) clickEditButton() {
	page := s.session.Page()
	err := page.GetByTestId("api-key-detail-edit").Click()
	require.NoError(s.t, err)
	s.session.Sleep(300)
}

func (s *apiKeySteps) clearAndFillEditName(name string) {
	page := s.session.Page()
	input := page.GetByTestId("api-key-detail-edit-name")
	err := input.Fill(name)
	require.NoError(s.t, err)
	s.session.Sleep(200)
}

func (s *apiKeySteps) clearAndFillEditDescription(description string) {
	page := s.session.Page()
	input := page.GetByTestId("api-key-detail-edit-description")
	err := input.Fill(description)
	require.NoError(s.t, err)
	s.session.Sleep(200)
}

func (s *apiKeySteps) submitEdit() {
	page := s.session.Page()
	saveBtn := page.Locator("button:has-text('Save')").First()
	err := saveBtn.Click()
	require.NoError(s.t, err)
	s.session.Sleep(1000)
}

func (s *apiKeySteps) assertAPIKeyNameInDB(name string) {
	apiKeys, err := models.FindAPIKeysByOrganization(database.DB(s.t.Context()), s.session.OrgID.String())
	require.NoError(s.t, err)

	for _, apiKey := range apiKeys {
		if apiKey.Name == name {
			return
		}
	}
	require.Fail(s.t, "API key %q not found in DB", name)
}

func (s *apiKeySteps) clickDeleteOnDetail() {
	page := s.session.Page()
	err := page.GetByTestId("api-key-detail-delete").Click()
	require.NoError(s.t, err)
	s.session.Sleep(1000)
}

func (s *apiKeySteps) assertAPIKeyDeletedFromDB(name string) {
	apiKeys, err := models.FindAPIKeysByOrganization(database.DB(s.t.Context()), s.session.OrgID.String())
	require.NoError(s.t, err)

	for _, apiKey := range apiKeys {
		if apiKey.Name == name {
			require.Fail(s.t, "API key %q should have been deleted", name)
		}
	}
}

func (s *apiKeySteps) clickRegenerateToken() {
	page := s.session.Page()
	err := page.GetByTestId("api-key-detail-regenerate-token").Click()
	require.NoError(s.t, err)
	s.session.Sleep(1000)
}

func (s *apiKeySteps) loginAsViewer() {
	viewerEmail := support.RandomName("viewer") + "@superplane.local"
	viewerAccount, err := models.CreateAccount("Viewer User", viewerEmail)
	require.NoError(s.t, err)

	viewerUser, err := models.CreateUser(s.session.OrgID, viewerAccount.ID, viewerEmail, "Viewer User")
	require.NoError(s.t, err)

	authService, err := authorization.NewAuthService()
	require.NoError(s.t, err)

	err = authService.AssignRole(viewerUser.ID.String(), models.RoleOrgViewer, s.session.OrgID.String(), models.DomainTypeOrganization)
	require.NoError(s.t, err)

	s.session.Account = viewerAccount
	s.session.Login()
}

func (s *apiKeySteps) assertCreateButtonDisabled() {
	s.session.AssertDisabled(q.TestID("api-key-create-btn"))
}

func (s *apiKeySteps) assertEditButtonDisabled() {
	s.session.AssertDisabled(q.TestID("api-key-detail-edit"))
}

func (s *apiKeySteps) assertDeleteButtonDisabled() {
	s.session.AssertDisabled(q.TestID("api-key-detail-delete"))
}

// givenAPIKeyExists creates an API key directly in the DB for test setup.
func (s *apiKeySteps) givenAPIKeyExists(name, description string) {
	// Look up the human user to use as created_by (the FK references users.id, not accounts.id)
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), "e2e@superplane.local")
	require.NoError(s.t, err)

	desc := description
	apiKey, err := models.CreateAPIKey(
		database.Conn(),
		s.session.OrgID,
		name,
		&desc,
		user.ID,
		nil,
		nil,
	)
	require.NoError(s.t, err)
	require.NotNil(s.t, apiKey)
}
