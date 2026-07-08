package e2e

import (
	"testing"

	"github.com/google/uuid"
	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
)

func TestSettingsPermissionGuards(t *testing.T) {
	t.Run("viewer can read settings resources but cannot manage them", func(t *testing.T) {
		steps := &settingsPermissionGuardSteps{t: t}
		steps.start()
		steps.givenGroupExists("readonly-team")
		steps.givenRoleExists("readonly-role")
		steps.givenServiceAccountExists("readonly-bot")
		memberEmail := steps.givenMemberExists("readonly-member")
		steps.loginAsViewer()

		steps.visitGeneralSettings()
		steps.assertOrgUpdateDisabled()
		steps.assertOrgDeleteDisabled()

		steps.visitGroupsSettings()
		steps.assertGroupsCreateDisabled()
		steps.assertGroupUpdateDisabled("readonly-team")
		steps.assertGroupDeleteDisabled("readonly-team")

		steps.visitRolesSettings()
		steps.assertRolesCreateDisabled()
		steps.assertRoleUpdateDisabled("readonly-role")
		steps.assertRoleDeleteDisabled("readonly-role")

		steps.visitMembersSettings()
		steps.assertInviteLinkCreateDisabled()
		steps.assertMemberUpdateDisabled(memberEmail)
		steps.assertMemberDeleteDisabled(memberEmail)

		steps.visitServiceAccountsSettings()
		steps.assertServiceAccountCreateDisabled()
		steps.openServiceAccount("readonly-bot")
		steps.assertServiceAccountUpdateDisabled()
		steps.assertServiceAccountDeleteDisabled()
	})

	t.Run("secrets reader cannot create update or delete secrets", func(t *testing.T) {
		steps := &settingsPermissionGuardSteps{t: t}
		steps.start()
		steps.givenSecretExists("readonly-secret")
		steps.loginWithPermissions("secret-reader", organizationPermission("secrets", "read"))

		steps.visitSecretsSettings()
		steps.assertSecretCreateDisabled()
		steps.openSecret("readonly-secret")
		steps.assertSecretUpdateDisabled()
		steps.assertSecretDeleteDisabled()
	})

	t.Run("integrations reader cannot create update or delete integrations", func(t *testing.T) {
		steps := &settingsPermissionGuardSteps{t: t}
		steps.start()
		integration := steps.givenIntegrationExists("readonly-integration")
		steps.loginWithPermissions("integration-reader", organizationPermission("integrations", "read"))

		steps.visitIntegrationsSettings()
		steps.assertIntegrationCreateDisabled()
		steps.assertIntegrationUpdateDisabled()
		steps.visitIntegrationDetail(integration.ID.String())
		steps.assertIntegrationDeleteDisabled()
	})
}

type settingsPermissionGuardSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *settingsPermissionGuardSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *settingsPermissionGuardSteps) loginAsViewer() {
	loginAsViewer(s.t, s.session)
}

func (s *settingsPermissionGuardSteps) loginWithPermissions(roleLabel string, permissions ...*permissionSpec) {
	loginWithOrganizationPermissions(s.t, s.session, roleLabel, permissions...)
}

func (s *settingsPermissionGuardSteps) visitGeneralSettings() {
	s.visitSettings("general")
}

func (s *settingsPermissionGuardSteps) visitGroupsSettings() {
	s.visitSettings("groups")
}

func (s *settingsPermissionGuardSteps) visitRolesSettings() {
	s.visitSettings("roles")
}

func (s *settingsPermissionGuardSteps) visitMembersSettings() {
	s.visitSettings("members")
}

func (s *settingsPermissionGuardSteps) visitSecretsSettings() {
	s.visitSettings("secrets")
}

func (s *settingsPermissionGuardSteps) visitIntegrationsSettings() {
	s.visitSettings("integrations")
}

func (s *settingsPermissionGuardSteps) visitServiceAccountsSettings() {
	s.visitSettings("service-accounts")
}

func (s *settingsPermissionGuardSteps) visitSettings(section string) {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/" + section)
	s.session.Sleep(500)
}

func (s *settingsPermissionGuardSteps) visitIntegrationDetail(integrationID string) {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/integrations/" + integrationID)
	s.session.Sleep(500)
}

func (s *settingsPermissionGuardSteps) openSecret(name string) {
	link := s.session.Page().GetByTestId("secrets-secret-link").GetByText(name, pw.LocatorGetByTextOptions{Exact: pw.Bool(true)})
	require.NoError(s.t, link.Click())
	s.session.Sleep(500)
}

func (s *settingsPermissionGuardSteps) openServiceAccount(name string) {
	link := s.session.Page().GetByTestId("sa-link").GetByText(name, pw.LocatorGetByTextOptions{Exact: pw.Bool(true)})
	require.NoError(s.t, link.Click())
	s.session.Sleep(500)
}

func (s *settingsPermissionGuardSteps) assertOrgUpdateDisabled() {
	s.session.AssertDisabled(q.Locator(`button:has-text("Save Changes")`))
}

func (s *settingsPermissionGuardSteps) assertOrgDeleteDisabled() {
	s.session.AssertDisabled(q.Locator(`button:has-text("Delete Organization...")`))
}

func (s *settingsPermissionGuardSteps) assertGroupsCreateDisabled() {
	s.session.AssertDisabled(q.Locator(`button:has-text("Create New Group")`))
}

func (s *settingsPermissionGuardSteps) assertGroupUpdateDisabled(name string) {
	row := s.rowByText(name)
	require.NoError(s.t, row.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Viewer"}).WaitFor())
	disabled, err := row.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Viewer"}).IsDisabled()
	require.NoError(s.t, err)
	require.True(s.t, disabled)
}

func (s *settingsPermissionGuardSteps) assertGroupDeleteDisabled(name string) {
	row := s.rowByText(name)
	disabled, err := row.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Delete group"}).IsDisabled()
	require.NoError(s.t, err)
	require.True(s.t, disabled)
}

func (s *settingsPermissionGuardSteps) assertRolesCreateDisabled() {
	s.session.AssertDisabled(q.Locator(`button:has-text("New Organization Role")`))
}

func (s *settingsPermissionGuardSteps) assertRoleUpdateDisabled(name string) {
	row := s.rowByText(name)
	require.NoError(s.t, row.Locator(`[aria-label="Edit role"]`).WaitFor())
	linkCount, err := row.Locator(`a[aria-label="Edit role"]`).Count()
	require.NoError(s.t, err)
	require.Equal(s.t, 0, linkCount)
}

func (s *settingsPermissionGuardSteps) assertRoleDeleteDisabled(name string) {
	row := s.rowByText(name)
	disabled, err := row.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Delete role"}).IsDisabled()
	require.NoError(s.t, err)
	require.True(s.t, disabled)
}

func (s *settingsPermissionGuardSteps) assertInviteLinkCreateDisabled() {
	s.session.AssertDisabled(q.Locator(`[aria-label="Toggle invite link"]`))
}

func (s *settingsPermissionGuardSteps) assertMemberUpdateDisabled(email string) {
	row := s.rowByText(email)
	disabled, err := row.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Viewer"}).IsDisabled()
	require.NoError(s.t, err)
	require.True(s.t, disabled)
}

func (s *settingsPermissionGuardSteps) assertMemberDeleteDisabled(email string) {
	row := s.rowByText(email)
	buttons := row.GetByRole("button")
	count, err := buttons.Count()
	require.NoError(s.t, err)
	require.GreaterOrEqual(s.t, count, 2)
	disabled, err := buttons.Nth(count - 1).IsDisabled()
	require.NoError(s.t, err)
	require.True(s.t, disabled)
}

func (s *settingsPermissionGuardSteps) assertSecretCreateDisabled() {
	s.session.AssertDisabled(q.TestID("secrets-create-btn"))
}

func (s *settingsPermissionGuardSteps) assertSecretUpdateDisabled() {
	s.session.AssertDisabled(q.TestID("secret-detail-edit-name"))
	s.session.AssertDisabled(q.TestID("secret-detail-edit-key"))
	s.session.AssertDisabled(q.TestID("secret-detail-remove-key"))
	s.session.AssertDisabled(q.TestID("secret-detail-add-key"))
}

func (s *settingsPermissionGuardSteps) assertSecretDeleteDisabled() {
	s.session.AssertDisabled(q.TestID("secret-detail-delete"))
}

func (s *settingsPermissionGuardSteps) assertIntegrationCreateDisabled() {
	s.session.AssertDisabled(q.Locator(`button:has-text("Connect")`))
}

func (s *settingsPermissionGuardSteps) assertIntegrationUpdateDisabled() {
	s.session.AssertDisabled(q.Locator(`button:has-text("Configure")`))
}

func (s *settingsPermissionGuardSteps) assertIntegrationDeleteDisabled() {
	s.session.AssertDisabled(q.TestID("integration-detail-delete"))
}

func (s *settingsPermissionGuardSteps) assertServiceAccountCreateDisabled() {
	s.session.AssertDisabled(q.TestID("sa-create-btn"))
}

func (s *settingsPermissionGuardSteps) assertServiceAccountUpdateDisabled() {
	s.session.AssertDisabled(q.TestID("sa-detail-edit"))
	s.session.AssertDisabled(q.TestID("sa-detail-regenerate-token"))
}

func (s *settingsPermissionGuardSteps) assertServiceAccountDeleteDisabled() {
	s.session.AssertDisabled(q.TestID("sa-detail-delete"))
}

func (s *settingsPermissionGuardSteps) rowByText(text string) pw.Locator {
	row := s.session.Page().GetByRole("row", pw.PageGetByRoleOptions{Name: text})
	require.NoError(s.t, row.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(10000)}))
	return row
}

func (s *settingsPermissionGuardSteps) givenGroupExists(name string) {
	authService, err := authorization.NewAuthService()
	require.NoError(s.t, err)
	require.NoError(s.t, authService.CreateGroup(
		s.session.OrgID.String(),
		models.DomainTypeOrganization,
		name,
		models.RoleOrgViewer,
		name,
		name,
	))
}

func (s *settingsPermissionGuardSteps) givenRoleExists(displayName string) {
	authService, err := authorization.NewAuthService()
	require.NoError(s.t, err)
	require.NoError(s.t, authService.CreateCustomRole(s.session.OrgID.String(), &authorization.RoleDefinition{
		Name:        support.RandomName("readonly-role"),
		DisplayName: displayName,
		DomainType:  models.DomainTypeOrganization,
		Description: "E2E permission guard role",
		Permissions: []*authorization.Permission{
			{
				Resource:   "canvases",
				Action:     "read",
				DomainType: models.DomainTypeOrganization,
			},
		},
	}))
}

func (s *settingsPermissionGuardSteps) givenMemberExists(label string) string {
	account := createAccountForRole(s.t, s.session, label, models.RoleOrgViewer)
	return account.Email
}

func (s *settingsPermissionGuardSteps) givenSecretExists(name string) {
	secretSteps := &SecretsSteps{t: s.t, session: s.session}
	secretSteps.givenASecretExists(name, map[string]string{"KEY1": "value1", "KEY2": "value2"})
}

func (s *settingsPermissionGuardSteps) givenServiceAccountExists(name string) {
	serviceAccountSteps := &serviceAccountSteps{t: s.t, session: s.session}
	serviceAccountSteps.givenServiceAccountExists(name, "Permission guard test")
}

func (s *settingsPermissionGuardSteps) givenIntegrationExists(name string) *models.Integration {
	integration, err := models.CreateIntegration(uuid.New(), s.session.OrgID, "github", name, nil)
	require.NoError(s.t, err)
	require.NoError(s.t, database.Conn().Model(integration).Update("state", models.IntegrationStateReady).Error)
	integration.State = models.IntegrationStateReady
	return integration
}
