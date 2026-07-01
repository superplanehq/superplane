package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestHomePage(t *testing.T) {
	t.Run("creating a new canvas", func(t *testing.T) {
		steps := &TestHomePageSteps{t: t}
		steps.Start()
		steps.VisitHomePage()
		steps.ClickNewApp()
		steps.AssertNavigatedToCanvas()
	})

	t.Run("showing canvases in folders", func(t *testing.T) {
		steps := &TestHomePageSteps{t: t}
		steps.Start()
		steps.GivenCanvasInFolder("Foldered Canvas", "Deployments")
		steps.VisitHomePage()
		steps.AssertCanvasFolderVisible("Deployments", "Foldered Canvas")
	})
}

type TestHomePageSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (steps *TestHomePageSteps) Start() {
	steps.session = ctx.NewSession(steps.t)
	steps.session.Start()
	steps.session.Login()
}

func (steps *TestHomePageSteps) VisitHomePage() {
	steps.session.Visit("/" + steps.session.OrgID.String() + "/")
}

func (steps *TestHomePageSteps) AssertNavigatedToCanvas() {
	url := steps.session.Page().URL()
	assert.Regexp(steps.t, `/apps/[0-9a-f-]{36}`, url)
}

func (steps *TestHomePageSteps) AssertCanvasSavedInDB(canvasName string) {
	canvas, err := models.FindCanvasByName(canvasName, steps.session.OrgID)

	assert.NoError(steps.t, err)
	assert.Equal(steps.t, canvasName, canvas.Name)
}

func (steps *TestHomePageSteps) GivenCanvasInFolder(canvasName, folderTitle string) {
	canvas := shared.NewCanvasSteps(canvasName, steps.t, steps.session)
	canvas.Create()

	folder, err := models.CreateCanvasFolder(steps.session.OrgID, folderTitle, models.CanvasFolderColorBlue)
	assert.NoError(steps.t, err)

	_, err = models.UpdateCanvasFolderMembership(steps.session.OrgID, canvas.WorkflowID, &folder.ID)
	assert.NoError(steps.t, err)
}

func (steps *TestHomePageSteps) AssertCanvasFolderVisible(folderTitle, canvasName string) {
	steps.session.AssertText(folderTitle)
	steps.session.AssertText(canvasName)
}

func (steps *TestHomePageSteps) ClickNewApp() {
	newAppButton := q.Locator(`button[aria-label="Create new app"]`).Run(steps.session)
	if visible, _ := newAppButton.IsVisible(); visible {
		steps.session.Click(q.Locator(`button[aria-label="Create new app"]`))
	}

	steps.session.Click(q.Text("Start from scratch"))
	steps.session.Sleep(3000)
}
