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
		const githubOwner = "e2e-github-owner"
		const githubTokenValue = "test-github-token"
		integrationName := githubOwner + "-account"

		steps.start()
		steps.visitIntegrationsSettingsPage()
		steps.createGitHubIntegration(githubOwner, githubTokenValue)
		steps.assertGithubVisibleInTheList(integrationName, githubOwner)
		steps.assertGithubPersisted(integrationName, githubOwner)
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
	addIntegrationButton := q.Text("Add Integration")
	githubTypeButton := q.Locator(`button:has-text("GitHub")`)
	ownerInput := q.Locator(`input[placeholder="Johndoe"]`)
	tokenInput := q.Locator(`input[placeholder="Enter your API token"]`)
	createButton := q.Locator(`button:has-text("Create")`)

	s.session.Click(addIntegrationButton)
	s.session.AssertText("Select Integration Type")

	s.session.Click(githubTypeButton)

	s.session.FillIn(ownerInput, ownerSlug)
	s.session.FillIn(tokenInput, token)
	s.session.Sleep(300)

	s.session.Click(createButton) // one click to unblur the input and validate
	s.session.Sleep(300)
	s.session.Click(createButton) // second click to actually submit
	s.session.Sleep(300)
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
	addIntegrationButton := q.Text("Add Integration")
	semaphoreTypeButton := q.Locator(`button:has-text("Semaphore")`)
	orgURLInput := q.Locator(`input[placeholder="https://your-org.semaphoreci.com"]`)
	tokenInput := q.Locator(`input[placeholder="Enter your API token"]`)
	createButton := q.Locator(`button:has-text("Create")`)

	s.session.Click(addIntegrationButton)
	s.session.AssertText("Select Integration Type")

	s.session.Click(semaphoreTypeButton)

	s.session.FillIn(orgURLInput, orgURL)
	s.session.FillIn(tokenInput, token)
	s.session.Sleep(300)

	s.session.Click(createButton) // one click to unblur the input and validate
	s.session.Sleep(300)
	s.session.Click(createButton) // second click to actually submit
	s.session.Sleep(300)
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
