package e2e

import (
	"testing"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestOrganizationCreation(t *testing.T) {
	t.Run("creating a new organization starts workspace setup", func(t *testing.T) {
		steps := &organizationCreationSteps{t: t}
		orgName := "E2E Created Organization"
		steps.start()
		steps.visitCreateOrganizationPage()
		steps.fillInOrganizationName(orgName)
		steps.submitOrganizationForm()
		steps.assertOrganizationSavedInDB(orgName)
		steps.assertRedirectedToWorkspaceSetup(orgName)
	})
}

type organizationCreationSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *organizationCreationSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *organizationCreationSteps) visitCreateOrganizationPage() {
	s.session.Visit("/create")
}

func (s *organizationCreationSteps) fillInOrganizationName(name string) {
	input := q.Locator(`input#name`)
	s.session.FillIn(input, name)
	s.session.Sleep(300)
}

func (s *organizationCreationSteps) submitOrganizationForm() {
	button := q.Locator(`button:has-text("Create Organization")`)
	s.session.Click(button)
	s.session.Sleep(1500)
}

func (s *organizationCreationSteps) assertOrganizationSavedInDB(name string) {
	org, err := models.FindOrganizationByName(name)
	require.NoError(s.t, err)
	require.Equal(s.t, name, org.Name)
}

// A new organization holds no workspace, so the owner must land in the setup
// steps of a new workspace.
func (s *organizationCreationSteps) assertRedirectedToWorkspaceSetup(name string) {
	org, err := models.FindOrganizationByName(name)
	require.NoError(s.t, err)

	waitErr := s.session.Page().WaitForURL("**/"+org.Slug+"/workspaces/*/setup**", pw.PageWaitForURLOptions{
		Timeout: pw.Float(30000),
	})
	require.NoError(s.t, waitErr)
}
