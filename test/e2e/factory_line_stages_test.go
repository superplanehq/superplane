package e2e

import (
	"testing"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
)

func TestFactoryLineCustomStages(t *testing.T) {
	s := &factoryLineStagesSteps{t: t}
	s.start()
	s.givenALine()
	s.openLine()
	s.addAndReorderStage("QA")
	s.assertStageOrder("backlog", "phase", "custom", "verify", "done")
	s.reloadAndAssertStageVisible()
	s.removeStage("QA")
	s.assertStageOrder("backlog", "phase", "verify", "done")
	s.reloadAndAssertStageHidden()
}

type factoryLineStagesSteps struct {
	t        *testing.T
	session  *session.TestSession
	factory  *models.Factory
	line     *models.FactoryLine
	stageKey string
}

func (s *factoryLineStagesSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
	require.NoError(s.t, models.EnableExperimentalFeature(s.session.OrgID, features.FeatureFactories))
	require.NoError(s.t, models.EnableExperimentalFeature(s.session.OrgID, features.FeatureFactoryCustomStages))
}

func (s *factoryLineStagesSteps) givenALine() {
	factory, err := models.CreateFactory(database.DB(s.t.Context()), s.session.OrgID, support.RandomName("factory"), "", "")
	require.NoError(s.t, err)
	support.CompleteFactoryOnboarding(s.t, factory)
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)
	resources := &support.ResourceRegistry{User: user.ID, Organization: &models.Organization{ID: s.session.OrgID}}
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(s.t, resources, factory.ID, "Implement", "start")
	line, err := factory.CreateLine(database.DB(s.t.Context()), "Main", []models.FactoryLineStep{{
		Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint,
	}})
	require.NoError(s.t, err)
	s.factory = factory
	s.line = line
}

func (s *factoryLineStagesSteps) openLine() {
	s.session.Visit("/" + s.session.OrgID.String() + "/workspaces/" + s.factory.Key + "/lines/" + s.line.ID.String())
	s.session.AssertVisible(q.TestID("lines-phase-board"))
}

func (s *factoryLineStagesSteps) addAndReorderStage(name string) {
	page := s.session.Page()
	require.NoError(s.t, page.GetByTestId("lines-board-view-menu").Click())
	require.NoError(s.t, page.GetByRole("menuitem", pw.PageGetByRoleOptions{Name: "Manage stages", Exact: pw.Bool(true)}).Click())
	dialog := page.GetByTestId("manage-line-stages-dialog")
	require.NoError(s.t, dialog.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Add stage", Exact: pw.Bool(true)}).Click())
	require.NoError(s.t, dialog.GetByRole("textbox", pw.LocatorGetByRoleOptions{Name: "New stage name", Exact: pw.Bool(true)}).Fill(name))
	require.NoError(s.t, dialog.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Move " + name + " up", Exact: pw.Bool(true)}).Click())
	require.NoError(s.t, dialog.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Save stages", Exact: pw.Bool(true)}).Click())
	require.NoError(s.t, dialog.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateHidden}))

	reloaded, err := s.factory.FindLine(database.DB(s.t.Context()), s.line.ID)
	require.NoError(s.t, err)
	for _, stage := range reloaded.StagesValue() {
		if stage.Kind == models.FactoryLineStageCustom {
			s.stageKey = stage.Key
			break
		}
	}
	require.NotEmpty(s.t, s.stageKey)
}

func (s *factoryLineStagesSteps) assertStageOrder(kinds ...string) {
	reloaded, err := s.factory.FindLine(database.DB(s.t.Context()), s.line.ID)
	require.NoError(s.t, err)
	actual := make([]string, len(reloaded.StagesValue()))
	for index, stage := range reloaded.StagesValue() {
		actual[index] = stage.Kind
	}
	assert.Equal(s.t, kinds, actual)
}

func (s *factoryLineStagesSteps) reloadAndAssertStageVisible() {
	_, err := s.session.Page().Reload()
	require.NoError(s.t, err)
	s.session.AssertVisible(q.TestID("lines-custom-stage-" + s.stageKey))
}

func (s *factoryLineStagesSteps) removeStage(name string) {
	page := s.session.Page()
	require.NoError(s.t, page.GetByTestId("lines-board-view-menu").Click())
	require.NoError(s.t, page.GetByRole("menuitem", pw.PageGetByRoleOptions{Name: "Manage stages", Exact: pw.Bool(true)}).Click())
	dialog := page.GetByTestId("manage-line-stages-dialog")
	require.NoError(s.t, dialog.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Remove " + name, Exact: pw.Bool(true)}).Click())
	require.NoError(s.t, dialog.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Save stages", Exact: pw.Bool(true)}).Click())
	require.NoError(s.t, dialog.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateHidden}))
}

func (s *factoryLineStagesSteps) reloadAndAssertStageHidden() {
	_, err := s.session.Page().Reload()
	require.NoError(s.t, err)
	s.session.AssertHidden(q.TestID("lines-custom-stage-" + s.stageKey))
}
