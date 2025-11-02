package e2e

import (
	"testing"
)

func TestHomePage(t *testing.T) {
	ctx := NewTestContext(t)
	defer ctx.Shutdown()

	t.Run("creating a new canvas", func(t *testing.T) {
		ctx.Start()

		s := ctx.NewSession()
		defer s.Close()

		s.Start()
		s.Login()
		s.VisitHomePage()
		s.ClickButton("New Canvas")
		s.FillIn("canvas-name-input", "Example Canvas")
		s.ClickButton("Create canvas")
		s.AssertText("Example Canvas")
	})
}
