package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestOrganizationIntegrations(t *testing.T) {
	steps := &OrganizationIntegrationsSteps{t: t}

	t.Run("creating a new GitHub integration", func(t *testing.T) {
		steps.start()
		steps.visitIntegrationsSettingsPage()

		const githubOwner = "e2e-github-owner"
		const githubTokenValue = "test-github-token"
		originalIntegrationName := githubOwner + "-account"

		steps.createGitHubIntegration(githubOwner, githubTokenValue)
		steps.assertGithubVisibleInTheList(originalIntegrationName, githubOwner)
		steps.assertGithubPersisted(originalIntegrationName, githubOwner)
	})

	t.Run("creating a new Semaphore integration", func(t *testing.T) {
		const semaphoreOrgURL = "https://e2e-semaphore-org.semaphoreci.com"
		const semaphoreTokenValue = "test-semaphore-token"
		const integrationName = "e2e-semaphore-org-organization"

		steps.start()
		steps.visitIntegrationsSettingsPage()
		steps.createSemaphoreIntegration(semaphoreOrgURL, semaphoreTokenValue)
		steps.assertSemaphoreVisibleInTheList(integrationName, semaphoreOrgURL)
		steps.assertSemaphorePersisted(integrationName, semaphoreOrgURL)
	})

	t.Run("editing an existing GitHub integration", func(t *testing.T) {
		steps.start()
		steps.visitIntegrationsSettingsPage()

		const originalOwner = "e2e-github-owner"
		const originalToken = "test-github-token"
		originalIntegrationName := originalOwner + "-account"

		steps.createGitHubIntegration(originalOwner, originalToken)
		steps.assertGithubVisibleInTheList(originalIntegrationName, originalOwner)
		steps.assertGithubPersisted(originalIntegrationName, originalOwner)

		const updatedOwner = "e2e-github-owner-updated"
		const updatedToken = "test-github-token-updated"
		updatedIntegrationName := updatedOwner + "-account"

		steps.editGithubIntegration(originalIntegrationName, updatedOwner, updatedToken)
		steps.assertGithubVisibleInTheList(updatedIntegrationName, updatedOwner)
		steps.assertGithubPersisted(updatedIntegrationName, updatedOwner)
	})
}

type OrganizationIntegrationsSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *OrganizationIntegrationsSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *OrganizationIntegrationsSteps) visitIntegrationsSettingsPage() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/integrations")
	s.session.AssertText("Integrations")
}

func (s *OrganizationIntegrationsSteps) createGitHubIntegration(ownerSlug, token string) {
	ownerInput := q.Locator(`input[placeholder="Johndoe"]`)
	tokenInput := q.Locator(`input[placeholder="Enter your API token"]`)

	s.session.Click(q.Text("Add Integration"))
	s.session.AssertText("Select Integration Type")

	s.session.Click(q.Locator(`button:has-text("GitHub")`))

	s.session.FillIn(ownerInput, ownerSlug)
	s.session.FillIn(tokenInput, token)

	s.session.Click(q.TestID("create-integration-button"))
}

func (s *OrganizationIntegrationsSteps) assertGithubPersisted(integrationName, ownerSlug string) {
	integration, err := models.FindIntegrationByName(models.DomainTypeOrganization, s.session.OrgID, integrationName)
	require.NoError(s.t, err)

	require.Equal(s.t, models.IntegrationTypeGithub, integration.Type)
	require.Equal(s.t, models.IntegrationAuthTypeToken, integration.AuthType)
	require.Equal(s.t, models.DomainTypeOrganization, integration.DomainType)
	require.Equal(s.t, s.session.OrgID, integration.DomainID)
	require.Equal(s.t, "https://github.com/"+ownerSlug, integration.URL)
}

func (s *OrganizationIntegrationsSteps) assertGithubVisibleInTheList(integrationName, ownerSlug string) {
	s.session.AssertText("Organization Integrations")
	s.session.AssertText(integrationName)
	s.session.AssertText("https://github.com/" + ownerSlug)
}

func (s *OrganizationIntegrationsSteps) createSemaphoreIntegration(orgURL, token string) {
	orgURLInput := q.Locator(`input[placeholder="https://your-org.semaphoreci.com"]`)
	tokenInput := q.Locator(`input[placeholder="Enter your API token"]`)

	s.session.Click(q.Text("Add Integration"))
	s.session.AssertText("Select Integration Type")

	s.session.Click(q.Locator(`button:has-text("Semaphore")`))

	s.session.FillIn(orgURLInput, orgURL)
	s.session.FillIn(tokenInput, token)

	s.session.Click(q.TestID("create-integration-button"))
}

func (s *OrganizationIntegrationsSteps) editGithubIntegration(currentIntegrationName, newOwnerSlug, newToken string) {
	editButton := q.TestID("edit-integration-" + currentIntegrationName)
	ownerInput := q.Locator(`input[placeholder="Johndoe"]`)
	tokenInput := q.Locator(`input[placeholder="Enter new API token value"]`)

	s.session.Click(editButton)

	s.session.FillIn(ownerInput, newOwnerSlug)
	s.session.FillIn(tokenInput, newToken)

	s.session.Click(q.TestID("create-integration-button"))
}

func (s *OrganizationIntegrationsSteps) assertSemaphorePersisted(integrationName, orgURL string) {
	integration, err := models.FindIntegrationByName(models.DomainTypeOrganization, s.session.OrgID, integrationName)
	require.NoError(s.t, err)

	require.Equal(s.t, models.IntegrationTypeSemaphore, integration.Type)
	require.Equal(s.t, models.IntegrationAuthTypeToken, integration.AuthType)
	require.Equal(s.t, models.DomainTypeOrganization, integration.DomainType)
	require.Equal(s.t, s.session.OrgID, integration.DomainID)
	require.Equal(s.t, orgURL, integration.URL)
}

func (s *OrganizationIntegrationsSteps) assertSemaphoreVisibleInTheList(integrationName, orgURL string) {
	s.session.AssertText("Organization Integrations")
	s.session.AssertText(integrationName)
	s.session.AssertText(orgURL)
}
