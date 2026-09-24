package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestHomePage(t *testing.T) {
	t.Run("creating a new canvas", func(t *testing.T) {
		steps := &TestHomePageSteps{t: t}
		steps.Start()
		steps.VisitHomePage()
		steps.ClickNewApp()
		steps.AssertNavigatedToCanvas()
	})

	t.Run("operator cannot create canvases from empty home", func(t *testing.T) {
		steps := &TestHomePageSteps{t: t}
		steps.Start()
		steps.LoginAsOperator()
		steps.VisitHomePage()
		steps.AssertEmptyHomeVisible()
		steps.AssertNewAppDisabled()
		steps.AssertNotRedirectedToNewApp()
	})

	t.Run("operator cannot open new app page directly", func(t *testing.T) {
		steps := &TestHomePageSteps{t: t}
		steps.Start()
		steps.LoginAsOperator()
		steps.VisitNewAppPage()
		steps.AssertNewAppPagePermissionDenied()
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

func (steps *TestHomePageSteps) AssertNavigatedToCanvas() {
	url := steps.session.Page().URL()
	assert.Regexp(steps.t, `/apps/[0-9a-f-]{36}`, url)
}

func (steps *TestHomePageSteps) AssertNotRedirectedToNewApp() {
	url := steps.session.Page().URL()
	assert.NotContains(steps.t, url, "/apps/new")
}

func (steps *TestHomePageSteps) AssertNewAppPagePermissionDenied() {
	steps.session.AssertVisible(q.TestID("permission-denied-page"))
	steps.session.AssertText("Permission denied")
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

func (steps *TestHomePageSteps) LoginAsOperator() {
	loginAsOperator(steps.t, steps.session)
}

func (steps *TestHomePageSteps) ClickStartFromScratch() {
	steps.session.Click(q.Text("Create a blank app"))
	steps.session.Sleep(500)
}

func (steps *TestHomePageSteps) ClickNewApp() {
	// An empty org opens /apps/new. The home toolbar is not on that page.
	steps.session.WaitForBrowserPath("/" + steps.session.OrgSlug + "/apps/new")
	steps.ClickStartFromScratch()
	steps.session.Sleep(2500)
}
