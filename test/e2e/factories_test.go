package e2e

import (
	"testing"

	"github.com/google/uuid"
	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
)

func TestFactories(t *testing.T) {
	t.Run("updating factory name and description", func(t *testing.T) {
		steps := &factorySteps{t: t}
		originalName := support.RandomName("factory")
		updatedName := support.RandomName("factory-updated")

		steps.start()
		factory := steps.givenFactoryExists(originalName, "original description")
		steps.visitFactorySettings(factory)
		steps.fillFactorySettingsName(updatedName)
		steps.fillFactorySettingsDescription("updated description")
		steps.submitFactorySettings()
		steps.assertFactoryVisibleInSidebar(updatedName)
		steps.assertFactorySavedInDB(factory.ID, updatedName, "updated description")
	})

	t.Run("soft-deleting a factory", func(t *testing.T) {
		steps := &factorySteps{t: t}
		name := support.RandomName("factory")

		steps.start()
		factory := steps.givenFactoryExists(name, "will be deleted")
		steps.visitFactorySettings(factory)
		steps.clickDeleteFactory()
		steps.confirmDeleteFactory()
		steps.assertRedirectedToFactoriesList()
		steps.assertFactoryNotVisibleInList(name)
		steps.assertFactorySoftDeletedInDB(factory.ID, name)

		recreated := steps.givenFactoryExists(name, "reused after delete")
		require.NotEqual(t, factory.ID, recreated.ID)
		steps.visitFactoriesList()
		steps.assertFactoryVisibleInList(name)
	})

	t.Run("persists setup completion across reload", func(t *testing.T) {
		steps := &factorySteps{t: t}
		name := support.RandomName("factory")

		steps.start()
		factory := steps.givenFactoryExists(name, "")
		steps.visitFactoryOverview(factory)
		steps.session.AssertVisible(q.TestID("workspace-setup"))

		steps.completeFactoryOnboarding(factory)
		steps.visitFactoryOverview(factory)
		steps.session.AssertVisible(q.TestID("overview-lines-card"))
		steps.session.AssertHidden(q.TestID("workspace-setup"))
	})
}

type factorySteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *factorySteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
	require.NoError(s.t, models.EnableExperimentalFeature(s.session.OrgID, features.FeatureFactories))
}

func (s *factorySteps) givenFactoryExists(name, description string) *models.Factory {
	factory, err := models.CreateFactory(database.DB(s.t.Context()), s.session.OrgID, name, description, "")
	require.NoError(s.t, err)
	return factory
}

func (s *factorySteps) visitFactoriesList() {
	s.session.Visit("/" + s.session.OrgID.String() + "/workspaces")
	s.session.Sleep(500)
}

func (s *factorySteps) visitFactorySettings(factory *models.Factory) {
	s.session.Visit("/" + s.session.OrgID.String() + "/workspaces/" + factory.Key + "/settings/general")
	s.session.AssertVisible(q.TestID("factory-settings-name"))
}

func (s *factorySteps) visitFactoryOverview(factory *models.Factory) {
	s.session.Visit("/" + s.session.OrgID.String() + "/workspaces/" + factory.Key + "/overview")
}

func (s *factorySteps) completeFactoryOnboarding(factory *models.Factory) {
	vcsID := uuid.New().String()
	agentID := uuid.New().String()
	appRepository := "acme/app"
	backlogRepository := "acme/backlog"
	issuesSource := models.FactoryOnboardingIssuesSourceVCS
	agentHarness := models.FactoryOnboardingAgentHarnessClaudeCode
	appID := uuid.New().String()
	lineID := uuid.New().String()
	require.NoError(s.t, factory.CompleteOnboarding(database.DB(s.t.Context()), models.FactoryOnboardingPatch{
		VCSIntegrationID:   &vcsID,
		AgentIntegrationID: &agentID,
		AppRepository:      &appRepository,
		BacklogRepository:  &backlogRepository,
		IssuesSource:       &issuesSource,
		AgentHarness:       &agentHarness,
		ProvisionedAppID:   &appID,
		ProvisionedLineID:  &lineID,
	}))
}

func (s *factorySteps) fillFactorySettingsName(name string) {
	page := s.session.Page()
	err := page.GetByTestId("factory-settings-name").Fill(name)
	require.NoError(s.t, err)
	s.session.Sleep(200)
}

func (s *factorySteps) fillFactorySettingsDescription(description string) {
	page := s.session.Page()
	err := page.GetByTestId("factory-settings-description").Fill(description)
	require.NoError(s.t, err)
	s.session.Sleep(200)
}

func (s *factorySteps) submitFactorySettings() {
	s.session.Click(q.TestID("factory-settings-save"))
	s.session.Sleep(1000)
}

func (s *factorySteps) assertFactoryVisibleInSidebar(name string) {
	s.session.AssertText(name)
}

func (s *factorySteps) assertFactorySavedInDB(factoryID uuid.UUID, name, description string) {
	factory, err := models.FindFactory(database.DB(s.t.Context()), s.session.OrgID, factoryID)
	require.NoError(s.t, err)
	require.Equal(s.t, name, factory.Name)
	require.Equal(s.t, description, factory.Description)
}

func (s *factorySteps) clickDeleteFactory() {
	s.session.Click(q.TestID("factory-settings-delete-button"))
	s.session.Sleep(300)
}

func (s *factorySteps) confirmDeleteFactory() {
	s.session.AssertVisible(q.TestID("factory-delete-confirm-button"))
	s.session.Click(q.TestID("factory-delete-confirm-button"))
	s.session.Sleep(1000)
}

func (s *factorySteps) assertRedirectedToFactoriesList() {
	s.session.WaitForBrowserPath("/" + s.session.OrgID.String() + "/workspaces")
}

func (s *factorySteps) assertFactoryVisibleInList(name string) {
	s.session.AssertText(name)
}

func (s *factorySteps) assertFactoryNotVisibleInList(name string) {
	page := s.session.Page()
	locator := page.GetByText(name, pw.PageGetByTextOptions{Exact: pw.Bool(true)})
	count, err := locator.Count()
	require.NoError(s.t, err)
	require.Zero(s.t, count, "factory %q should not be visible in list", name)
}

func (s *factorySteps) assertFactorySoftDeletedInDB(factoryID uuid.UUID, originalName string) {
	_, err := models.FindFactory(database.DB(s.t.Context()), s.session.OrgID, factoryID)
	require.ErrorIs(s.t, err, models.ErrFactoryNotFound)

	var deleted models.Factory
	err = database.DB(s.t.Context()).Unscoped().
		Where("organization_id = ? AND id = ?", s.session.OrgID, factoryID).
		First(&deleted).
		Error
	require.NoError(s.t, err)
	require.True(s.t, deleted.DeletedAt.Valid)
	require.Contains(s.t, deleted.Name, "(deleted-")
	require.NotEqual(s.t, originalName, deleted.Name)
}
