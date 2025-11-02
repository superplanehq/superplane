package e2e

import (
	"testing"

	q "github.com/superplanehq/superplane/test/e2e/queries"
)

func TestHomePage(t *testing.T) {
	ctx := NewTestContext(t)
	ctx.Start()

	t.Run("creating a new canvas", func(t *testing.T) {
		s := ctx.NewSession()
		defer s.Close()

		newCanvasButton := q.Text("New Canvas")
		saveCanvasButton := q.Text("Create canvas")
		canvasNameInput := q.TestID("canvas-name-input")

		s.Start()
		s.Login()
		s.VisitHomePage()
		s.Click(newCanvasButton)
		s.FillIn(canvasNameInput, "Example Canvas")
		s.Click(saveCanvasButton)
		s.AssertText("Example Canvas")
	})

	t.Run("creating a new component", func(t *testing.T) {
		s := ctx.NewSession()
		defer s.Close()

		componentsTab := q.Text("Components")
		newComponentButton := q.Text("New Component")
		saveComponentButton := q.Text("Create Component")
		componentNameInput := q.TestID("component-name-input")

		s.Start()
		s.Login()
		s.VisitHomePage()
		s.Click(componentsTab)
		s.Click(newComponentButton)
		s.FillIn(componentNameInput, "Example Component")
		s.Click(saveComponentButton)
	})
}
