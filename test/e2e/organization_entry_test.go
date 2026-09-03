package e2e

import (
	"testing"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestOrganizationEntry(t *testing.T) {
	t.Run("redirects the retired selection URL to the factory workspace", func(t *testing.T) {
		session := newLoggedInSession(t)

		require.NoError(t, models.EnableExperimentalFeature(session.OrgID, features.FeatureFactories))
		session.Visit("/?select=true")
		session.WaitForBrowserPath("/" + session.OrgSlug + "/workspaces")
	})

	t.Run("opens the legacy organization when factories are disabled", func(t *testing.T) {
		session := newLoggedInSession(t)

		require.NoError(t, models.DisableExperimentalFeature(session.OrgID, features.FeatureFactories))
		session.Visit("/")
		session.WaitForBrowserPath("/" + session.OrgSlug + "/apps/new")
	})

	t.Run("creates an organization and workspace from the GitHub account during onboarding", func(t *testing.T) {
		session := ctx.NewSession(t)
		session.StartWithoutUser()

		account, err := models.CreateAccount("GitHub User", "github-user@superplane.local")
		require.NoError(t, err)
		middleware.MarkOwnerSetupCompleted()
		require.NoError(t, models.SaveAccountLinkedAccount(
			database.DB(t.Context()),
			models.NewAccountLinkedAccount(account.ID, models.ProviderGitHub, "github-user-id", "github-owner", "GitHub Owner", ""),
		))
		session.Account = account
		session.Login()

		session.Visit("/")
		session.AssertText("Set up your workspace")
		session.Click(q.TestID("initial-onboarding-create-workspace"))
		waitErr := session.Page().WaitForURL("**/workspaces/*/setup**", pw.PageWaitForURLOptions{
			Timeout: pw.Float(30000),
		})
		require.NoError(t, waitErr)

		organization, err := models.FindOrganizationByName("GitHub Owner")
		require.NoError(t, err)

		workspaces, err := models.ListFactories(database.DB(t.Context()), organization.ID)
		require.NoError(t, err)
		require.Len(t, workspaces, 1)
	})

	t.Run("switches organization from the legacy navigation menu", func(t *testing.T) {
		session := newLoggedInSession(t)
		require.NoError(t, models.DisableExperimentalFeature(session.OrgID, features.FeatureFactories))
		secondOrganization := createOrganizationForAccount(t, session, "Second E2E Organization")
		require.NoError(t, models.DisableExperimentalFeature(secondOrganization.ID, features.FeatureFactories))

		session.Visit("/" + session.OrgSlug)
		session.Click(q.TestID("legacy-organization-menu"))
		session.Click(q.TestID("legacy-organization-switch"))
		currentOrganizationOption := session.Page().GetByTestId("legacy-organization-option-" + session.OrgID.String())
		currentOrganizationText, err := currentOrganizationOption.TextContent()
		require.NoError(t, err)
		require.Contains(t, currentOrganizationText, "Current")
		session.Click(q.TestID("legacy-organization-option-" + secondOrganization.ID.String()))
		session.WaitForBrowserPath("/" + secondOrganization.Slug)
	})
}

func newLoggedInSession(t *testing.T) *session.TestSession {
	session := ctx.NewSession(t)
	session.Start()
	session.Login()
	return session
}

func createOrganizationForAccount(t *testing.T, session *session.TestSession, name string) *models.Organization {
	organization, err := models.CreateOrganization(name, "")
	require.NoError(t, err)

	user, err := models.CreateUser(organization.ID, session.Account.ID, session.Account.Email, session.Account.Name)
	require.NoError(t, err)

	authService, err := authorization.NewAuthService()
	require.NoError(t, err)
	tx := database.DB(t.Context()).Begin()
	require.NoError(t, authService.SetupOrganization(tx, organization.ID.String(), user.ID.String()))
	require.NoError(t, tx.Commit().Error)

	return organization
}
