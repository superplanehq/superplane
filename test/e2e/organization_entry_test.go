package e2e

import (
	"testing"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestOrganizationEntry(t *testing.T) {
	t.Run("opens workspaces when factories are enabled and the org home is requested", func(t *testing.T) {
		session := newLoggedInSession(t)

		require.NoError(t, models.EnableExperimentalFeature(session.OrgID, features.FeatureFactories))
		session.Visit("/" + session.OrgSlug)
		waitForWorkspaceSetup(t, session)
	})

	t.Run("creates an organization and workspace from the account name during onboarding", func(t *testing.T) {
		session := ctx.NewSession(t)
		session.StartWithoutUser()

		account, err := models.CreateAccount("GitHub User", "github-user@superplane.local")
		require.NoError(t, err)
		middleware.MarkOwnerSetupCompleted()
		session.Account = account
		session.Login()

		session.Visit("/")
		waitErr := session.Page().WaitForURL("**/onboarding?attempt=*", pw.PageWaitForURLOptions{
			Timeout: pw.Float(30000),
		})
		require.NoError(t, waitErr)
		session.AssertVisible(q.TestID("workspace-setup"))
		session.AssertVisible(q.TestID("first-run-log-out"))
		session.AssertHidden(q.TestID("first-run-organization-switch"))

		organization, err := models.FindOrganizationByName("GitHub User")
		require.NoError(t, err)

		workspaces, err := models.ListFactories(database.DB(t.Context()), organization.ID)
		require.NoError(t, err)
		require.Len(t, workspaces, 1)
	})

}

func waitForWorkspaceSetup(t *testing.T, session *session.TestSession) {
	t.Helper()

	waitErr := session.Page().WaitForURL("**/"+session.OrgSlug+"/workspaces/**/setup", pw.PageWaitForURLOptions{
		Timeout: pw.Float(30000),
	})
	require.NoError(t, waitErr)
	session.AssertVisible(q.TestID("workspace-setup"))
}

func newLoggedInSession(t *testing.T) *session.TestSession {
	session := ctx.NewSession(t)
	session.Start()
	session.Login()
	return session
}
