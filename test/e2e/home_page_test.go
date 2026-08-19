package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	t.Run("viewer cannot create canvases from empty home", func(t *testing.T) {
		steps := &TestHomePageSteps{t: t}
		steps.Start()
		steps.LoginAsViewer()
		steps.VisitHomePage()
		steps.AssertEmptyHomeVisible()
		steps.AssertNewAppDisabled()
		steps.AssertNotRedirectedToNewApp()
	})

	t.Run("viewer cannot open new app page directly", func(t *testing.T) {
		steps := &TestHomePageSteps{t: t}
		steps.Start()
		steps.LoginAsViewer()
		steps.VisitNewAppPage()
		steps.AssertNewAppPageNotFound()
	})

	t.Run("canvas creator without update cannot create inside a folder", func(t *testing.T) {
		steps := &TestHomePageSteps{t: t}
		steps.Start()
		folder := steps.GivenCanvasFolder("Deployments")
		steps.LoginWithCanvasPermissions("canvas-folder-creator", canvasPermission("create"))
		steps.VisitNewAppPageForFolder(folder.ID.String())
		steps.ClickStartFromScratch()
		steps.AssertUpdatePermissionToast()
		steps.AssertCanvasCount(0)
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

func (steps *TestHomePageSteps) VisitNewAppPage() {
	steps.session.Visit("/" + steps.session.OrgID.String() + "/apps/new")
}

func (steps *TestHomePageSteps) VisitNewAppPageForFolder(folderID string) {
	steps.session.Visit("/" + steps.session.OrgID.String() + "/apps/new?folderId=" + folderID)
	steps.session.AssertText("Create New App in Deployments Folder")
}

func (steps *TestHomePageSteps) AssertNavigatedToCanvas() {
	url := steps.session.Page().URL()
	assert.Regexp(steps.t, `/apps/[0-9a-f-]{36}`, url)
}

func (steps *TestHomePageSteps) AssertNotRedirectedToNewApp() {
	url := steps.session.Page().URL()
	assert.NotContains(steps.t, url, "/apps/new")
}

func (steps *TestHomePageSteps) AssertNewAppPageNotFound() {
	steps.session.AssertText("404")
	steps.session.AssertHidden(q.Text("Create a blank app"))
}

func (steps *TestHomePageSteps) AssertEmptyHomeVisible() {
	steps.session.AssertText("Apps")
	steps.session.AssertText("No apps yet")
}

func (steps *TestHomePageSteps) AssertNewAppDisabled() {
	steps.session.AssertDisabled(q.Locator(`button[aria-label="Create new app"]`))
	steps.session.AssertHidden(q.Text("Create a blank app"))
}

func (steps *TestHomePageSteps) AssertCanvasSavedInDB(canvasName string) {
	canvas, err := models.FindCanvasByName(canvasName, steps.session.OrgID)

	assert.NoError(steps.t, err)
	assert.Equal(steps.t, canvasName, canvas.Name)
}

func (steps *TestHomePageSteps) AssertCanvasCount(expected int) {
	count, err := models.CountCanvasesByOrganization(steps.session.OrgID.String())
	require.NoError(steps.t, err)
	assert.Equal(steps.t, int64(expected), count)
}

func (steps *TestHomePageSteps) GivenCanvasInFolder(canvasName, folderTitle string) {
	canvas := shared.NewCanvasSteps(canvasName, steps.t, steps.session)
	canvas.Create()

	folder := steps.GivenCanvasFolder(folderTitle)

	_, err := models.UpdateCanvasFolderMembership(steps.session.OrgID, canvas.WorkflowID, &folder.ID)
	assert.NoError(steps.t, err)
}

func (steps *TestHomePageSteps) GivenCanvasFolder(folderTitle string) *models.CanvasFolder {
	folder, err := models.CreateCanvasFolder(steps.session.OrgID, folderTitle, models.CanvasFolderColorBlue)
	require.NoError(steps.t, err)
	return folder
}

func (steps *TestHomePageSteps) AssertCanvasFolderVisible(folderTitle, canvasName string) {
	steps.session.AssertText(folderTitle)
	steps.session.AssertText(canvasName)
}

func (steps *TestHomePageSteps) LoginAsViewer() {
	loginAsViewer(steps.t, steps.session)
}

func (steps *TestHomePageSteps) LoginWithCanvasPermissions(roleLabel string, permissions ...*permissionSpec) {
	loginWithCanvasPermissions(steps.t, steps.session, roleLabel, permissions...)
}

func (steps *TestHomePageSteps) ClickStartFromScratch() {
	steps.session.Click(q.Text("Create a blank app"))
	steps.session.Sleep(500)
}

func (steps *TestHomePageSteps) AssertUpdatePermissionToast() {
	steps.session.AssertText("You don't have permission to update canvases.")
}

func (steps *TestHomePageSteps) ClickNewApp() {
	newAppButton := q.Locator(`button[aria-label="Create new app"]`).Run(steps.session)
	if visible, _ := newAppButton.IsVisible(); visible {
		steps.session.Click(q.Locator(`button[aria-label="Create new app"]`))
	}

	steps.session.Click(q.Text("Create a blank app"))
	steps.session.Sleep(3000)
}
