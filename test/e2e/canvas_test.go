package e2e

import (
	"testing"
)

func TestHomePage(t *testing.T) {
	s := NewTestSession(t)
	defer s.Shutdown()

	t.Run("creating a new canvas", func(t *testing.T) {
		s.Start()
		s.VisitHomePage()
		s.AssertText("New Canvas")
		s.ClickButton("New Canvas")
		s.FillIn("Canvas name", "E2E Canvas")
		s.ClickButton("Create canvas")
		s.Sleep(2000)
		s.AssertText("E2E Canvas")
	})
}
