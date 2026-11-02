package e2e

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/secrets"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
)

func TestSettingsPermissionGuards(t *testing.T) {
	t.Run("operator can read settings resources but cannot manage them", func(t *testing.T) {
		steps := &settingsPermissionGuardSteps{t: t}
		steps.start()
		steps.givenGroupExists("readonly-team")
		steps.givenRoleExists("readonly-role")
		steps.givenAPIKeyExists("readonly-bot")
		memberEmail := steps.givenMemberExists("readonly-member")
		steps.loginAsOperator()

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
		steps.assertMemberUpdateDisabled(memberEmail)
		steps.assertInviteMemberHidden()
		steps.assertMemberDeleteDisabled(memberEmail)

		steps.visitAPIKeysSettings()
		steps.assertAPIKeyCreateDisabled()
		steps.openAPIKey("readonly-bot")
		steps.assertAPIKeyUpdateDisabled()
		steps.assertAPIKeyDeleteDisabled()
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

	t.Run("operator cannot open factory workspace settings by URL", func(t *testing.T) {
		steps := &settingsPermissionGuardSteps{t: t}
		steps.start()
		factory := steps.givenFactoryExists()
		steps.loginAsOperator()
		steps.visitFactoryWorkspaceSettings(factory)
		steps.assertPermissionDenied()
	})
}

type settingsPermissionGuardSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *settingsPermissionGuardSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	require.NoError(s.t, models.EnableExperimentalFeature(s.session.OrgID, features.FeatureFactories))
	s.session.Login()
}

func (s *settingsPermissionGuardSteps) loginAsOperator() {
	loginAsOperator(s.t, s.session)
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

func (s *settingsPermissionGuardSteps) visitAPIKeysSettings() {
	s.visitSettings("api-keys")
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

func (s *settingsPermissionGuardSteps) openAPIKey(name string) {
	link := s.session.Page().GetByTestId("api-key-link").GetByText(name, pw.LocatorGetByTextOptions{Exact: pw.Bool(true)})
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
	require.NoError(s.t, row.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Operator"}).WaitFor())
	disabled, err := row.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Operator"}).IsDisabled()
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

func (s *settingsPermissionGuardSteps) assertInviteMemberHidden() {
	s.session.AssertHidden(q.Text("Invite member"))
}

func (s *settingsPermissionGuardSteps) assertMemberUpdateDisabled(email string) {
	row := s.rowByText(email)
	role := row.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Role for"})
	require.NoError(s.t, role.WaitFor())
	disabled, err := role.IsDisabled()
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

func (s *settingsPermissionGuardSteps) assertAPIKeyCreateDisabled() {
	s.session.AssertDisabled(q.TestID("api-key-create-btn"))
}

func (s *settingsPermissionGuardSteps) assertAPIKeyUpdateDisabled() {
	s.session.AssertDisabled(q.TestID("api-key-detail-edit"))
	s.session.AssertDisabled(q.TestID("api-key-detail-regenerate-token"))
}

func (s *settingsPermissionGuardSteps) assertAPIKeyDeleteDisabled() {
	s.session.AssertDisabled(q.TestID("api-key-detail-delete"))
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
		models.RoleOrgOperator,
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
	account := createAccountForRole(s.t, s.session, label, models.RoleOrgOperator)
	return account.Email
}

func (s *settingsPermissionGuardSteps) givenSecretExists(name string) {
	encryptor, err := crypto.ProcessEncryptor()
	require.NoError(s.t, err)
	raw, err := json.Marshal(map[string]string{"KEY1": "value1", "KEY2": "value2"})
	require.NoError(s.t, err)
	encrypted, err := encryptor.Encrypt(s.t.Context(), raw, []byte(name))
	require.NoError(s.t, err)
	_, err = models.CreateSecret(name, secrets.ProviderLocal, s.session.Account.ID.String(), models.DomainTypeOrganization, s.session.OrgID, encrypted)
	require.NoError(s.t, err)
}

func (s *settingsPermissionGuardSteps) givenAPIKeyExists(name string) {
	user, err := models.FindActiveUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)
	description := "Permission guard test"
	_, err = models.CreateAPIKey(database.DB(s.t.Context()), s.session.OrgID, name, &description, user.ID, nil, nil)
	require.NoError(s.t, err)
}

func (s *settingsPermissionGuardSteps) givenIntegrationExists(name string) *models.Integration {
	integration, err := models.CreateIntegration(uuid.New(), s.session.OrgID, "github", name, nil)
	require.NoError(s.t, err)
	require.NoError(s.t, database.Conn().Model(integration).Update("state", models.IntegrationStateReady).Error)
	integration.State = models.IntegrationStateReady
	return integration
}

func (s *settingsPermissionGuardSteps) givenFactoryExists() *models.Factory {
	require.NoError(s.t, models.EnableExperimentalFeature(s.session.OrgID, features.FeatureFactories))
	factory, err := models.CreateFactory(database.DB(s.t.Context()), s.session.OrgID, support.RandomName("factory"), "permission guard", "")
	require.NoError(s.t, err)
	return factory
}

func (s *settingsPermissionGuardSteps) visitFactoryWorkspaceSettings(factory *models.Factory) {
	s.session.Visit("/" + s.session.OrgID.String() + "/workspaces/" + factory.Key + "/settings/workspace/general")
	s.session.Sleep(500)
}

func (s *settingsPermissionGuardSteps) assertPermissionDenied() {
	s.session.AssertVisible(q.TestID("permission-denied-page"))
	s.session.AssertText("Permission denied")
}

type permissionSpec struct {
	resource string
	action   string
}

func organizationPermission(resource, action string) *permissionSpec {
	return &permissionSpec{resource: resource, action: action}
}

func loginAsOperator(t *testing.T, sess *session.TestSession) {
	account := createAccountForRole(t, sess, "operator", models.RoleOrgOperator)
	sess.Account = account
	sess.Login()
}

func loginWithOrganizationPermissions(t *testing.T, sess *session.TestSession, roleLabel string, permissions ...*permissionSpec) {
	roleName := support.RandomName(roleLabel)
	rolePermissions := make([]*authorization.Permission, 0, len(permissions))
	for _, permission := range permissions {
		rolePermissions = append(rolePermissions, &authorization.Permission{
			Resource:   permission.resource,
			Action:     permission.action,
			DomainType: models.DomainTypeOrganization,
		})
	}

	authService, err := authorization.NewAuthService()
	require.NoError(t, err)

	err = authService.CreateCustomRole(sess.OrgID.String(), &authorization.RoleDefinition{
		Name:        roleName,
		DisplayName: roleLabel,
		DomainType:  models.DomainTypeOrganization,
		Description: "E2E permission guard role",
		Permissions: rolePermissions,
		InheritsFrom: &authorization.RoleDefinition{
			Name:       models.RoleOrgOperator,
			DomainType: models.DomainTypeOrganization,
		},
	})
	require.NoError(t, err)

	account := createAccountForRole(t, sess, roleLabel, roleName)
	sess.Account = account
	sess.Login()
}

func createAccountForRole(t *testing.T, sess *session.TestSession, label, roleName string) *models.Account {
	email := support.RandomName(label) + "@superplane.local"
	account, err := models.CreateAccount(label, email)
	require.NoError(t, err)

	user, err := models.CreateUser(sess.OrgID, account.ID, email, label)
	require.NoError(t, err)

	authService, err := authorization.NewAuthService()
	require.NoError(t, err)

	err = authService.AssignRole(user.ID.String(), roleName, sess.OrgID.String(), models.DomainTypeOrganization)
	require.NoError(t, err)

	return account
}
