package e2e

import (
	"testing"

	"github.com/google/uuid"
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
		steps.visitFactory(factory.ID)
		steps.openFactoryActionsMenu()
		steps.clickEditFactory()
		steps.fillEditFactoryName(updatedName)
		steps.fillEditFactoryDescription("updated description")
		steps.submitEditFactory()
		steps.assertFactoryVisibleOnDetail(updatedName)
		steps.assertFactorySavedInDB(factory.ID, updatedName, "updated description")
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
	factory, err := models.CreateFactory(database.DB(s.t.Context()), s.session.OrgID, name, description)
	require.NoError(s.t, err)
	return factory
}

func (s *factorySteps) visitFactory(factoryID uuid.UUID) {
	s.session.Visit("/" + s.session.OrgID.String() + "/factories/" + factoryID.String())
	s.session.Sleep(500)
}

func (s *factorySteps) openFactoryActionsMenu() {
	s.session.Click(q.TestID("factory-actions-menu"))
	s.session.Sleep(300)
}

func (s *factorySteps) clickEditFactory() {
	s.session.Click(q.TestID("factory-edit-action"))
	s.session.Sleep(300)
}

func (s *factorySteps) fillEditFactoryName(name string) {
	page := s.session.Page()
	err := page.GetByTestId("edit-factory-name-input").Fill(name)
	require.NoError(s.t, err)
	s.session.Sleep(200)
}

func (s *factorySteps) fillEditFactoryDescription(description string) {
	page := s.session.Page()
	err := page.GetByTestId("edit-factory-description-input").Fill(description)
	require.NoError(s.t, err)
	s.session.Sleep(200)
}

func (s *factorySteps) submitEditFactory() {
	s.session.Click(q.TestID("factory-edit-save-button"))
	s.session.Sleep(1000)
}

func (s *factorySteps) assertFactoryVisibleOnDetail(name string) {
	s.session.AssertText(name)
}

func (s *factorySteps) assertFactorySavedInDB(factoryID uuid.UUID, name, description string) {
	factory, err := models.FindFactory(database.DB(s.t.Context()), s.session.OrgID, factoryID)
	require.NoError(s.t, err)
	require.Equal(s.t, name, factory.Name)
	require.Equal(s.t, description, factory.Description)
}
