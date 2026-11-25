package e2e

import (
	"strings"
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
	s.addStage("QA")
	s.assertStageOrder("backlog", "phase-0", "verify", s.stageKey, "done")
	s.moveStage()
	s.assertStageOrder("backlog", s.stageKey, "phase-0", "verify", "done")
	s.reloadAndAssertStageVisible()
	s.moveAgentAndReload()
	s.removeStage()
	s.assertStageOrder("backlog", "phase-0", "verify", "done")
	s.reloadAndAssertStageHidden()
}

func TestFactoryLineCustomAutomationMove(t *testing.T) {
	s := &factoryLineStagesSteps{t: t}
	s.start()
	s.givenALine()
	s.openLine()
	s.addStage("QA")
	s.moveCustomAutomationAndReload()
}

func (s *factoryLineStagesSteps) moveCustomAutomationAndReload() {
	db := database.DB(s.t.Context())
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)
	resources := &support.ResourceRegistry{User: user.ID, Organization: &models.Organization{ID: s.session.OrgID}}
	app, _ := support.CreateFactoryAppWithOnRunTrigger(s.t, resources, s.factory.ID, "Check QA", "start")
	require.NoError(s.t, app.SetColumnKey(db, s.stageKey))
	s.reloadAndAssertStageVisible()
	s.enterStageEditing()
	page := s.session.Page()
	handle := pw.LocatorGetByRoleOptions{Name: "Move " + app.Name, Exact: pw.Bool(true)}
	require.NoError(s.t, page.GetByTestId("lines-stage-shell-"+s.stageKey).GetByRole("button", handle).DragTo(page.GetByTestId("lines-stage-shell-verify")))
	s.saveStageChanges()
	s.reloadAndAssertStageVisible()
	s.enterStageEditing()
	require.NoError(s.t, page.GetByTestId("lines-stage-shell-verify").GetByRole("button", handle).WaitFor())
	reloaded, err := models.FindCanvasInTransaction(db, s.session.OrgID, app.ID)
	require.NoError(s.t, err)
	require.Equal(s.t, "verify", reloaded.DisplayColumnKey)
	require.Equal(s.t, s.stageKey, *reloaded.ColumnKey)
	require.Equal(s.t, app.LiveVersionID, reloaded.LiveVersionID)
}

func (s *factoryLineStagesSteps) moveStage() {
	page := s.session.Page()
	s.enterStageEditing()
	handle := page.GetByTestId("lines-stage-reorder-" + s.stageKey)
	target := page.GetByTestId("lines-stage-shell-phase-0")
	require.NoError(s.t, handle.DragTo(target))
	s.saveStageChanges()
}

func (s *factoryLineStagesSteps) moveAgentAndReload() {
	page := s.session.Page()
	s.enterStageEditing()
	handle := pw.LocatorGetByRoleOptions{Name: "Move " + s.line.Stages[1].Name, Exact: pw.Bool(true)}
	target := page.GetByTestId("lines-stage-shell-" + s.stageKey)
	require.NoError(s.t, page.GetByTestId("lines-stage-shell-phase-0").GetByRole("button", handle).DragTo(target))
	s.saveStageChanges()
	s.reloadAndAssertStageVisible()
	s.enterStageEditing()
	require.NoError(s.t, target.GetByRole("button", handle).WaitFor())
	require.NoError(s.t, page.GetByTestId("line-stage-edit-toolbar").GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Done", Exact: pw.Bool(true)}).Click())
}

func (s *factoryLineStagesSteps) enterStageEditing() {
	page := s.session.Page()
	require.NoError(s.t, page.GetByTestId("lines-board-view-menu").Click())
	require.NoError(s.t, page.GetByTestId("lines-board-edit-stages").Click())
	s.session.AssertVisible(q.TestID("line-stage-edit-toolbar"))
}

func (s *factoryLineStagesSteps) saveStageChanges() {
	page := s.session.Page()
	toolbar := page.GetByTestId("line-stage-edit-toolbar")
	require.NoError(s.t, toolbar.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Save changes", Exact: pw.Bool(true)}).Click())
	dialog := page.GetByTestId("save-line-stages-dialog")
	require.NoError(s.t, dialog.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Save changes", Exact: pw.Bool(true)}).Click())
	require.NoError(s.t, toolbar.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateHidden}))
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

func (s *factoryLineStagesSteps) addStage(name string) {
	page := s.session.Page()
	s.enterStageEditing()
	toolbar := page.GetByTestId("line-stage-edit-toolbar")
	require.NoError(s.t, toolbar.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Add stage", Exact: pw.Bool(true)}).Click())
	dialog := page.GetByTestId("add-line-stage-dialog")
	require.NoError(s.t, dialog.GetByRole("textbox", pw.LocatorGetByRoleOptions{Name: "Name", Exact: pw.Bool(true)}).Fill(name))
	require.NoError(s.t, dialog.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Add stage", Exact: pw.Bool(true)}).Click())
	require.NoError(s.t, dialog.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateHidden}))
	s.saveStageChanges()
	automationDialog := page.GetByTestId("custom-automation-setup-modal")
	require.NoError(s.t, automationDialog.GetByRole("button", pw.LocatorGetByRoleOptions{Name: "Close", Exact: pw.Bool(true)}).Click())

	reloaded, err := s.factory.FindLine(database.DB(s.t.Context()), s.line.ID)
	require.NoError(s.t, err)
	for _, stage := range reloaded.StagesValue() {
		if strings.HasPrefix(stage.Key, "custom-") {
			s.stageKey = stage.Key
			break
		}
	}
	require.NotEmpty(s.t, s.stageKey)
}

func (s *factoryLineStagesSteps) assertStageOrder(keys ...string) {
	reloaded, err := s.factory.FindLine(database.DB(s.t.Context()), s.line.ID)
	require.NoError(s.t, err)
	actual := make([]string, len(reloaded.StagesValue()))
	for index, stage := range reloaded.StagesValue() {
		actual[index] = stage.Key
	}
	assert.Equal(s.t, keys, actual)
}

func (s *factoryLineStagesSteps) reloadAndAssertStageVisible() {
	_, err := s.session.Page().Reload()
	require.NoError(s.t, err)
	s.session.AssertVisible(q.TestID("lines-custom-stage-" + s.stageKey))
}

func (s *factoryLineStagesSteps) removeStage() {
	page := s.session.Page()
	s.enterStageEditing()
	require.NoError(s.t, page.GetByTestId("lines-stage-delete-"+s.stageKey).Click())
	s.saveStageChanges()
}

func (s *factoryLineStagesSteps) reloadAndAssertStageHidden() {
	_, err := s.session.Page().Reload()
	require.NoError(s.t, err)
	s.session.AssertHidden(q.TestID("lines-custom-stage-" + s.stageKey))
}
