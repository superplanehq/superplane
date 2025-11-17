package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestOrganizationCreation(t *testing.T) {
	steps := &organizationCreationSteps{t: t}

	t.Run("creating a new organization from the create page", func(t *testing.T) {
		orgName := "E2E Created Organization"

		steps.start()
		steps.visitCreateOrganizationPage()
		steps.fillInOrganizationName(orgName)
		steps.submitOrganizationForm()
		steps.assertOrganizationSavedInDB(orgName)
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
