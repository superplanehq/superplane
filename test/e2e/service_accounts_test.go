package e2e

import (
	"testing"

	pw "github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
)

func TestServiceAccounts(t *testing.T) {
	steps := &serviceAccountSteps{t: t}

	t.Run("creating a service account with viewer role", func(t *testing.T) {
		steps.start()
		steps.visitServiceAccountsPage()
		steps.clickCreateServiceAccount()
		steps.fillName("ci-deploy-bot")
		steps.fillDescription("Deploys from CI")
		steps.selectRole("Viewer")
		steps.submitCreate()
		steps.assertTokenDisplayed()
		steps.dismissTokenModal()
		steps.assertServiceAccountSavedInDB("ci-deploy-bot", "Deploys from CI", models.RoleOrgViewer)
	})

	t.Run("creating a service account with admin role", func(t *testing.T) {
		steps.start()
		steps.visitServiceAccountsPage()
		steps.clickCreateServiceAccount()
		steps.fillName("admin-bot")
		steps.fillDescription("Admin automation")
		steps.selectRole("Admin")
		steps.submitCreate()
		steps.assertTokenDisplayed()
		steps.dismissTokenModal()
		steps.assertServiceAccountSavedInDB("admin-bot", "Admin automation", models.RoleOrgAdmin)
	})

	t.Run("viewing service accounts in the list", func(t *testing.T) {
		steps.start()
		steps.givenServiceAccountExists("list-test-bot", "For listing test")
		steps.visitServiceAccountsPage()
		steps.assertServiceAccountVisibleInList("list-test-bot")
	})

	t.Run("navigating to service account detail", func(t *testing.T) {
		steps.start()
		steps.givenServiceAccountExists("detail-test-bot", "For detail test")
		steps.visitServiceAccountsPage()
		steps.clickServiceAccountLink("detail-test-bot")
		steps.assertOnDetailPage("detail-test-bot")
	})

	t.Run("editing a service account", func(t *testing.T) {
		steps.start()
		steps.givenServiceAccountExists("edit-test-bot", "Original description")
		steps.visitServiceAccountsPage()
		steps.clickServiceAccountLink("edit-test-bot")
		steps.clickEditButton()
		steps.clearAndFillEditName("edited-bot")
		steps.clearAndFillEditDescription("Updated description")
		steps.submitEdit()
		steps.assertServiceAccountNameInDB("edited-bot")
	})

	t.Run("deleting a service account", func(t *testing.T) {
		steps.start()
		steps.givenServiceAccountExists("delete-test-bot", "Will be deleted")
		steps.visitServiceAccountsPage()
		steps.assertServiceAccountVisibleInList("delete-test-bot")
		steps.clickServiceAccountLink("delete-test-bot")
		steps.clickDeleteOnDetail()
		steps.assertServiceAccountDeletedFromDB("delete-test-bot")
	})

	t.Run("regenerating a service account token", func(t *testing.T) {
		steps.start()
		steps.givenServiceAccountExists("regen-test-bot", "Token regen test")
		steps.visitServiceAccountsPage()
		steps.clickServiceAccountLink("regen-test-bot")
		steps.clickRegenerateToken()
		steps.assertTokenDisplayed()
	})

	t.Run("viewer cannot create or manage service accounts", func(t *testing.T) {
		steps.start()
		steps.givenServiceAccountExists("viewer-test-bot", "Viewer RBAC test")
		steps.loginAsViewer()
		steps.visitServiceAccountsPage()
		steps.assertCreateButtonDisabled()
		steps.clickServiceAccountLink("viewer-test-bot")
		steps.assertEditButtonDisabled()
		steps.assertDeleteButtonDisabled()
	})
}

type serviceAccountSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *serviceAccountSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *serviceAccountSteps) visitServiceAccountsPage() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/service-accounts")
	s.session.Sleep(500)
}

func (s *serviceAccountSteps) clickCreateServiceAccount() {
	page := s.session.Page()
	createBtn := page.GetByTestId("sa-create-btn")
	err := createBtn.First().Click()
	require.NoError(s.t, err)
	s.session.Sleep(500)
}

func (s *serviceAccountSteps) fillName(name string) {
	page := s.session.Page()
	err := page.GetByTestId("sa-create-name").Fill(name)
	require.NoError(s.t, err)
	s.session.Sleep(200)
}

func (s *serviceAccountSteps) fillDescription(description string) {
	page := s.session.Page()
	err := page.GetByTestId("sa-create-description").Fill(description)
	require.NoError(s.t, err)
	s.session.Sleep(200)
}

func (s *serviceAccountSteps) selectRole(roleLabel string) {
	page := s.session.Page()

	trigger := page.GetByTestId("sa-create-role")
	err := trigger.Click()
	require.NoError(s.t, err)
	s.session.Sleep(300)

	option := page.GetByRole("option", pw.PageGetByRoleOptions{Name: roleLabel, Exact: pw.Bool(true)})
	err = option.Click()
	require.NoError(s.t, err)
	s.session.Sleep(300)
}

func (s *serviceAccountSteps) submitCreate() {
	page := s.session.Page()
	err := page.GetByTestId("sa-create-submit").Click()
	require.NoError(s.t, err)
	s.session.Sleep(1000)
}

func (s *serviceAccountSteps) assertTokenDisplayed() {
	page := s.session.Page()
	tokenInput := page.GetByTestId("sa-token-display")
	err := tokenInput.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(5000)})
	require.NoError(s.t, err)

	value, err := tokenInput.InputValue()
	require.NoError(s.t, err)
	require.NotEmpty(s.t, value, "token should not be empty")
}

func (s *serviceAccountSteps) dismissTokenModal() {
	page := s.session.Page()
	err := page.GetByTestId("sa-token-done").Click()
	require.NoError(s.t, err)
	s.session.Sleep(500)
}

func (s *serviceAccountSteps) assertServiceAccountSavedInDB(name, description, expectedRole string) {
	orgID := s.session.OrgID.String()
	serviceAccounts, err := models.FindServiceAccountsByOrganization(orgID)
	require.NoError(s.t, err)

	var found *models.User
	for i := range serviceAccounts {
		if serviceAccounts[i].Name == name {
			found = &serviceAccounts[i]
			break
		}
	}
	require.NotNil(s.t, found, "service account %q should exist in DB", name)
	require.Equal(s.t, models.UserTypeServiceAccount, found.Type)
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

func (s *serviceAccountSteps) assertServiceAccountVisibleInList(name string) {
	s.session.AssertText(name)
}

func (s *serviceAccountSteps) clickServiceAccountLink(name string) {
	page := s.session.Page()
	link := page.GetByTestId("sa-link").GetByText(name, pw.LocatorGetByTextOptions{Exact: pw.Bool(true)})
	err := link.Click()
	require.NoError(s.t, err)
	s.session.Sleep(500)
}

func (s *serviceAccountSteps) assertOnDetailPage(name string) {
	s.session.AssertText(name)
	s.session.AssertText("API Token")
}

func (s *serviceAccountSteps) clickEditButton() {
	page := s.session.Page()
	err := page.GetByTestId("sa-detail-edit").Click()
	require.NoError(s.t, err)
	s.session.Sleep(300)
}

func (s *serviceAccountSteps) clearAndFillEditName(name string) {
	page := s.session.Page()
	input := page.GetByTestId("sa-detail-edit-name")
	err := input.Fill(name)
	require.NoError(s.t, err)
	s.session.Sleep(200)
}

func (s *serviceAccountSteps) clearAndFillEditDescription(description string) {
	page := s.session.Page()
	input := page.GetByTestId("sa-detail-edit-description")
	err := input.Fill(description)
	require.NoError(s.t, err)
	s.session.Sleep(200)
}

func (s *serviceAccountSteps) submitEdit() {
	page := s.session.Page()
	saveBtn := page.Locator("button:has-text('Save')").First()
	err := saveBtn.Click()
	require.NoError(s.t, err)
	s.session.Sleep(1000)
}

func (s *serviceAccountSteps) assertServiceAccountNameInDB(name string) {
	serviceAccounts, err := models.FindServiceAccountsByOrganization(s.session.OrgID.String())
	require.NoError(s.t, err)

	for _, sa := range serviceAccounts {
		if sa.Name == name {
			return
		}
	}
	require.Fail(s.t, "service account %q not found in DB", name)
}

func (s *serviceAccountSteps) clickDeleteOnDetail() {
	page := s.session.Page()
	err := page.GetByTestId("sa-detail-delete").Click()
	require.NoError(s.t, err)
	s.session.Sleep(1000)
}

func (s *serviceAccountSteps) assertServiceAccountDeletedFromDB(name string) {
	serviceAccounts, err := models.FindServiceAccountsByOrganization(s.session.OrgID.String())
	require.NoError(s.t, err)

	for _, sa := range serviceAccounts {
		if sa.Name == name {
			require.Fail(s.t, "service account %q should have been deleted", name)
		}
	}
}

func (s *serviceAccountSteps) clickRegenerateToken() {
	page := s.session.Page()
	err := page.GetByTestId("sa-detail-regenerate-token").Click()
	require.NoError(s.t, err)
	s.session.Sleep(1000)
}

func (s *serviceAccountSteps) loginAsViewer() {
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

func (s *serviceAccountSteps) assertCreateButtonDisabled() {
	s.session.AssertDisabled(q.TestID("sa-create-btn"))
}

func (s *serviceAccountSteps) assertEditButtonDisabled() {
	s.session.AssertDisabled(q.TestID("sa-detail-edit"))
}

func (s *serviceAccountSteps) assertDeleteButtonDisabled() {
	s.session.AssertDisabled(q.TestID("sa-detail-delete"))
}

// givenServiceAccountExists creates a service account directly in the DB for test setup.
func (s *serviceAccountSteps) givenServiceAccountExists(name, description string) {
	// Look up the human user to use as created_by (the FK references users.id, not accounts.id)
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), "e2e@superplane.local")
	require.NoError(s.t, err)

	desc := description
	sa, err := models.CreateServiceAccount(
		database.Conn(),
		s.session.OrgID,
		name,
		&desc,
		user.ID,
	)
	require.NoError(s.t, err)
	require.NotNil(s.t, sa)
}
