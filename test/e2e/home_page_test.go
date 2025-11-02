package e2e

import (
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
)

func TestHomePage(t *testing.T) {
	steps := &TestHomePageSteps{t: t}

	t.Run("creating a new canvas", func(t *testing.T) {
		steps.Start()
		steps.VisitHomePage()
		steps.FillInNewCanvasForm("Example Canvas")
		steps.AssertCanvasSavedInDB("Example Canvas")
	})

	t.Run("creating a new component", func(t *testing.T) {
		steps.Start()
		steps.VisitHomePage()
		steps.SwitchToComponentsTab()
		steps.FillInNewComponentForm("Example Component")
		steps.AssertComponentSavedInDB("Example Component")
	})
}

type TestHomePageSteps struct {
	t       *testing.T
	session *TestSession
}

func (steps *TestHomePageSteps) Start() {
	steps.session = ctx.NewSession(steps.t)
	steps.session.Start()
	steps.session.Login()
}

func (steps *TestHomePageSteps) VisitHomePage() {
	steps.session.Visit("/" + steps.session.orgID + "/")
}

func (steps *TestHomePageSteps) AssertCanvasSavedInDB(canvasName string) {
	orgUUID := uuid.MustParse(steps.session.orgID)
	canvas, err := models.FindWorkflowByName(canvasName, orgUUID)

	assert.NoError(steps.t, err)
	assert.Equal(steps.t, canvasName, canvas.Name)
}

func (steps *TestHomePageSteps) FillInNewCanvasForm(canvasName string) {
	newCanvasButton := q.Text("New Canvas")
	saveCanvasButton := q.Text("Create canvas")
	canvasNameInput := q.TestID("canvas-name-input")

	steps.session.Click(newCanvasButton)
	steps.session.FillIn(canvasNameInput, canvasName)
	steps.session.Click(saveCanvasButton)
	steps.session.Sleep(500) // wait for canvas to be created
}

func (steps *TestHomePageSteps) AssertComponentSavedInDB(s string) {
	orgUUID := uuid.MustParse(steps.session.orgID)
	component, err := models.FindBlueprintByName(s, orgUUID)

	assert.NoError(steps.t, err)
	assert.Equal(steps.t, s, component.Name)
}

func (steps *TestHomePageSteps) FillInNewComponentForm(name string) {
	newComponentButton := q.Text("New Component")
	saveComponentButton := q.Text("Create Component")
	componentNameInput := q.TestID("component-name-input")

	steps.session.Click(newComponentButton)
	steps.session.FillIn(componentNameInput, name)
	steps.session.Click(saveComponentButton)
	steps.session.Sleep(500)
}

func (steps *TestHomePageSteps) SwitchToComponentsTab() {
	componentsTab := q.Text("Components")
	steps.session.Click(componentsTab)
}
