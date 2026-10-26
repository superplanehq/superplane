package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestAdminDashboard(t *testing.T) {
	t.Run("non-admin user is redirected away from admin page", func(t *testing.T) {
		steps := &adminSteps{t: t}
		steps.start()
		steps.session.Login()
		steps.session.Visit("/admin")
		// The frontend AdminLayout checks installation_admin and redirects to /
		steps.assertNotOnAdminPage()
	})

	t.Run("admin user can access admin dashboard and see organizations", func(t *testing.T) {
		steps := &adminSteps{t: t}
		steps.start()
		steps.promoteToAdmin()
		steps.session.Login()
		steps.session.Visit("/admin")
		steps.assertOnAdminPage()
		steps.assertOrganizationVisible("e2e-org")
	})

	t.Run("admin user can start and end a read-only support session", func(t *testing.T) {
		steps := &adminSteps{t: t}
		steps.start()
		steps.createSupportSessionTarget()
		steps.promoteToAdmin()
		steps.session.Login()

		steps.session.Visit("/admin/accounts")
		steps.session.Sleep(500)

		steps.clickStartSupportSession()
		steps.session.Sleep(2000)

		steps.assertSupportSessionBannerVisible()
		steps.assertServerRejectsWrites()

		steps.clickEndSupportSession()
		steps.session.Sleep(2000)

		// Should be back on admin page
		steps.assertOnAdminPage()
	})

	t.Run("admin user can update installation network settings", func(t *testing.T) {
		steps := &adminSteps{t: t}
		steps.start()
		steps.promoteToAdmin()
		steps.session.Login()
		steps.session.Visit("/admin/settings")
		steps.assertOnInstallationSettingsPage()
		steps.openNetworkSettings()
		steps.enablePrivateNetworkAccess()
		steps.saveInstallationSettings()
		steps.assertPrivateNetworkAccessEnabled()
	})

	t.Run("admin user can update installation smtp settings", func(t *testing.T) {
		steps := &adminSteps{t: t}
		steps.start()
		steps.promoteToAdmin()
		steps.session.Login()
		steps.session.Visit("/admin/settings")
		steps.assertOnInstallationSettingsPage()
		steps.openEmailSettings()
		steps.assertSMTPSettingsSaveDisabled()
		steps.enableSMTPSettings()
		steps.fillInSMTPSettings("smtp.example.com", "587", "smtp-user", "smtp-pass", "SuperPlane", "noreply@example.com", true)
		steps.saveSMTPSettings()
		steps.assertSMTPSettingsEnabled()
	})

	t.Run("admin user can view the installation license", func(t *testing.T) {
		steps := &adminSteps{t: t}
		steps.start()
		steps.promoteToAdmin()
		steps.session.Login()
		steps.session.Visit("/admin/license")
		steps.session.AssertText("SuperPlane Enterprise")
		steps.session.AssertText("Managed license")
		steps.session.AssertVisible(q.TestID("license-features"))
	})
}

func TestAdminOwnerSetupPromotion(t *testing.T) {
	t.Cleanup(func() {
		middleware.MarkOwnerSetupCompleted()
	})
	t.Run("owner setup promotes first account to installation admin", func(t *testing.T) {
		steps := &adminSetupSteps{t: t}
		steps.start()
		steps.visitSetupPage()
		steps.fillInOwnerDetailsAndSubmit()
		steps.assertAccountIsInstallationAdmin()
	})
}

// -- Admin dashboard step helpers --

type adminSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *adminSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
}

func (s *adminSteps) createSupportSessionTarget() {
	account, err := models.CreateAccount("Other User", "other@superplane.local")
	require.NoError(s.t, err)
	_, err = models.CreateUser(s.session.OrgID, account.ID, account.Email, account.Name)
	require.NoError(s.t, err)
}

func (s *adminSteps) promoteToAdmin() {
	err := models.PromoteToInstallationAdmin(s.session.Account.ID.String())
	require.NoError(s.t, err)
}

func (s *adminSteps) assertOnAdminPage() {
	s.session.AssertText("Installation Admin")
	s.session.AssertText("All Organizations")
}

func (s *adminSteps) assertNotOnAdminPage() {
	s.session.WaitUntilURLDoesNotContain("/admin")
}

func (s *adminSteps) assertOrganizationVisible(name string) {
	s.session.AssertText(name)
}

func (s *adminSteps) clickStartSupportSession() {
	s.session.Click(q.Text("Start support session"))
}

func (s *adminSteps) assertSupportSessionBannerVisible() {
	s.session.AssertText("Support session for")
	s.session.AssertText("This session is read-only.")
	s.session.AssertText("End support session")
}

// XMLHttpRequest bypasses the fetch guard in the UI, so the request
// reaches the server with the support session cookie.
func (s *adminSteps) assertServerRejectsWrites() {
	const organizationName = "support-session-write"
	status, err := s.session.Page().Evaluate(`(name) => new Promise((resolve) => {
		const xhr = new XMLHttpRequest();
		xhr.open("POST", "/organizations");
		xhr.setRequestHeader("Content-Type", "application/json");
		xhr.onload = () => resolve(xhr.status);
		xhr.onerror = () => resolve(0);
		xhr.send(JSON.stringify({ name }));
	})`, organizationName)
	require.NoError(s.t, err)
	assert.EqualValues(s.t, 403, status)

	var count int64
	require.NoError(s.t, database.Conn().Model(&models.Organization{}).Where("name = ?", organizationName).Count(&count).Error)
	assert.Zero(s.t, count)
}

func (s *adminSteps) clickEndSupportSession() {
	s.session.Click(q.Text("End support session"))
}

func (s *adminSteps) assertOnInstallationSettingsPage() {
	s.session.AssertText("Installation Settings")
	s.session.AssertText("Reset one automation")
}

func (s *adminSteps) openNetworkSettings() {
	s.session.Click(q.Locator("role=tab[name='Network']"))
	s.session.AssertText("Private network access")
}

func (s *adminSteps) openEmailSettings() {
	s.session.Click(q.Locator("role=tab[name='Email']"))
	s.session.AssertText("SMTP configuration")
}

func (s *adminSteps) enablePrivateNetworkAccess() {
	s.session.Click(q.TestID("installation-network-switch"))
}

func (s *adminSteps) saveInstallationSettings() {
	s.session.Click(q.TestID("installation-network-save"))
	s.session.AssertText("Network settings updated")
}

func (s *adminSteps) assertPrivateNetworkAccessEnabled() {
	metadata, err := models.GetInstallationMetadata(database.Conn())
	require.NoError(s.t, err)
	assert.True(s.t, metadata.AllowPrivateNetworkAccess)
}

func (s *adminSteps) enableSMTPSettings() {
	s.session.Click(q.TestID("installation-smtp-switch"))
}

func (s *adminSteps) fillInSMTPSettings(host, port, username, password, fromName, fromEmail string, useTLS bool) {
	s.session.FillIn(q.Locator(`input[placeholder="smtp.example.com"]`), host)
	s.session.FillIn(q.Locator(`input[placeholder="587"]`), port)
	s.session.FillIn(q.Locator(`input[placeholder="smtp-user"]`), username)
	s.session.FillIn(q.TestID("installation-smtp-password"), password)
	s.session.FillIn(q.Locator(`input[placeholder="SuperPlane"]`), fromName)
	s.session.FillIn(q.Locator(`input[placeholder="noreply@example.com"]`), fromEmail)

	if !useTLS {
		s.session.Click(q.TestID("installation-smtp-tls-switch"))
	}
}

func (s *adminSteps) saveSMTPSettings() {
	s.session.Click(q.TestID("installation-smtp-save"))
	s.session.AssertText("SMTP settings updated")
}

func (s *adminSteps) assertSMTPSettingsSaveDisabled() {
	s.session.AssertDisabled(q.TestID("installation-smtp-save"))
}

func (s *adminSteps) assertSMTPSettingsEnabled() {
	settings, err := models.FindEmailSettings(models.EmailProviderSMTP)
	require.NoError(s.t, err)
	assert.Equal(s.t, "smtp.example.com", settings.SMTPHost)
	assert.Equal(s.t, 587, settings.SMTPPort)
	assert.Equal(s.t, "smtp-user", settings.SMTPUsername)
	encryptor, err := crypto.ProcessEncryptor()
	require.NoError(s.t, err)
	password, err := encryptor.Decrypt(s.t.Context(), settings.SMTPPassword, []byte("smtp_password"))
	require.NoError(s.t, err)
	assert.Equal(s.t, "smtp-pass", string(password))
	assert.Equal(s.t, "SuperPlane", settings.SMTPFromName)
	assert.Equal(s.t, "noreply@example.com", settings.SMTPFromEmail)
	assert.True(s.t, settings.SMTPUseTLS)
}

// -- Owner setup + admin promotion step helpers --

type adminSetupSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *adminSetupSteps) start() {
	middleware.ResetOwnerSetupStateForTests()
	s.session = ctx.NewSession(s.t)
	s.session.StartWithoutUser()
}

func (s *adminSetupSteps) visitSetupPage() {
	s.session.Visit("/setup")
}

func (s *adminSetupSteps) fillInOwnerDetailsAndSubmit() {
	s.session.FillIn(q.Locator(`input[type="email"]`), "admin-test@superplane.local")
	s.session.FillIn(q.Locator(`input[placeholder="First name"]`), "Admin")
	s.session.FillIn(q.Locator(`input[placeholder="Last name"]`), "User")
	s.session.FillIn(q.Locator(`input[placeholder="Password"]`), "Password1")
	s.session.FillIn(q.Locator(`input[placeholder="Confirm password"]`), "Password1")
	s.session.Click(q.Text("Continue"))
	s.waitForSetupToComplete()
	skipRemainingOwnerSetupSteps(s.t, s.session)
}

func (s *adminSetupSteps) waitForSetupToComplete() {
	for i := 0; i < 50; i++ {
		var count int64
		err := database.Conn().Model(&models.Organization{}).Count(&count).Error
		if err == nil && count > 0 {
			s.session.Sleep(500)
			return
		}
		s.session.Sleep(200)
	}
	s.t.Fatal("timed out waiting for owner setup to complete")
}

func (s *adminSetupSteps) assertAccountIsInstallationAdmin() {
	account, err := models.FindAccountByEmail("admin-test@superplane.local")
	require.NoError(s.t, err)
	assert.True(s.t, account.IsInstallationAdmin(), "owner account should be an installation admin")
}
