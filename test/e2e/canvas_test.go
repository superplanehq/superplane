package e2e

import (
	"fmt"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
)

func TestHomePage(t *testing.T) {
	ctx := NewTestContext(t)
	ctx.Start()

	steps := &TestHomePageSteps{ctx: ctx}

	t.Run("creating a new canvas", func(t *testing.T) {
		steps.Start()
		defer steps.Cleanup()

		steps.VisitHomePage()
		steps.FillInNewCanvasForm("Example Canvas")
		steps.AssertCanvasSavedInDB("Example Canvas")
	})

	// t.Run("creating a new component", func(t *testing.T) {
	// 	steps.Start()
	// 	defer steps.Cleanup()

	// 	steps.VisitHomePage()
	// 	steps.SwitchToComponentsTab()
	// 	steps.FillInNewComponentForm("Example Component")
	// 	steps.AssertComponentSavedInDB("Example Component")
	// })
}

//
// TestHomePageSteps contains step definitions for home page tests.
//

type TestHomePageSteps struct {
	ctx     *TestContext
	session *TestSession
}

func (steps *TestHomePageSteps) Start() {
	steps.session = steps.ctx.NewSession()
	steps.session.Start()
	steps.session.Login()
}

func (steps *TestHomePageSteps) Cleanup() {
	steps.session.Close()
}

func (steps *TestHomePageSteps) VisitHomePage() {
	steps.session.Visit("/" + steps.session.orgID + "/")
}

func (steps *TestHomePageSteps) AssertCanvasSavedInDB(canvasName string) {
	orgUUID := uuid.MustParse(steps.session.orgID)

	c, err := models.ListCanvases()
	if err != nil {
		steps.ctx.t.Fatalf("failed to list canvases: %v", err)
	}
	fmt.Println("AAAAAAAAAAAAAAAAAAAAAA")
	fmt.Println("AAAAAAAAAAAAAAAAAAAAAA")
	fmt.Println("AAAAAAAAAAAAAAAAAAAAAA")
	fmt.Println("AAAAAAAAAAAAAAAAAAAAAA")
	fmt.Println("AAAAAAAAAAAAAAAAAAAAAA")
	fmt.Println("AAAAAAAAAAAAAAAAAAAAAA")
	fmt.Println("AAAAAAAAAAAAAAAAAAAAAA")
	fmt.Println("AAAAAAAAAAAAAAAAAAAAAA")
	fmt.Println("AAAAAAAAAAAAAAAAAAAAAA")
	fmt.Println("AAAAAAAAAAAAAAAAAAAAAA")
	fmt.Println("AAAAAAAAAAAAAAAAAAAAAA")
	fmt.Println("AAAAAAAAAAAAAAAAAAAAAA")
	fmt.Println("AAAAAAAAAAAAAAAAAAAAAA")
	fmt.Println(c)

	canvas, err := models.FindCanvasByName(canvasName, orgUUID)

	if err != nil {
		steps.ctx.t.Fatalf("failed to find canvas in DB: %v", err)
	}

	if canvas.Name != canvasName {
		steps.ctx.t.Fatalf("expected canvas name %q, got %q", canvasName, canvas.Name)
	}
}

func (steps *TestHomePageSteps) FillInNewCanvasForm(canvasName string) {
	newCanvasButton := q.Text("New Canvas")
	saveCanvasButton := q.Text("Create canvas")
	canvasNameInput := q.TestID("canvas-name-input")

	steps.session.Click(newCanvasButton)
	steps.session.FillIn(canvasNameInput, canvasName)
	steps.session.Click(saveCanvasButton)
	steps.session.Sleep(500) // wait for canvas to be created
	steps.session.TakeScreenshot()
}

func (steps *TestHomePageSteps) AssertComponentSavedInDB(s string) {
	orgUUID := uuid.MustParse(steps.session.orgID)
	component, err := models.FindBlueprintByName(s, orgUUID)

	if err != nil {
		steps.ctx.t.Fatalf("failed to find component in DB: %v", err)
	}

	if component.Name != s {
		steps.ctx.t.Fatalf("expected component name %q, got %q", s, component.Name)
	}
}

func (steps *TestHomePageSteps) FillInNewComponentForm(name string) {
	newComponentButton := q.Text("New Component")
	saveComponentButton := q.Text("Create Component")
	componentNameInput := q.TestID("component-name-input")

	steps.session.Click(newComponentButton)
	steps.session.FillIn(componentNameInput, name)
	steps.session.Click(saveComponentButton)
}

func (steps *TestHomePageSteps) SwitchToComponentsTab() {
	componentsTab := q.Text("Components")
	steps.session.Click(componentsTab)
}
