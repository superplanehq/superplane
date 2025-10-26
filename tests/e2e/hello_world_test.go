package e2e

import (
    "context"
    "testing"

    pw "github.com/playwright-community/playwright-go"
)

func TestHelloWorld(t *testing.T) {
    s := startBrowser(t)
    defer s.Close()

    s.Visit("/")
    s.AssertHTMLTitle("Hello")
		s.AssertText("hello")
}