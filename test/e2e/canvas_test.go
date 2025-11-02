package e2e

import (
	"testing"
)

func TestHomePage(t *testing.T) {
	s := NewTestSession(t)
	defer s.Shutdown()

	t.Run("creating a new canvas", func(t *testing.T) {
		s.Start()
		s.Login()
		s.VisitHomePage()
		s.AssertText("New Canvas")
		s.ClickButton("New Canvas")
		s.FillIn("canvas-name-input", "E2E Canvas")
		s.Sleep(100)
		s.ClickButton("Create canvas")
		s.TakeScreenshot()

		s.t.Logf("DONE?")
	})
}
