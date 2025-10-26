package e2e

import (
	"fmt"
	"testing"
	"time"

	pw "github.com/playwright-community/playwright-go"
	"github.com/superplanehq/superplane/pkg/server"
)

// TestSmoke starts the server and uses playwright-go to open the root page.
func TestSmoke(t *testing.T) {
	// Start the Go server in-process
	go server.Start()

	// Give the server a brief moment to bind
	time.Sleep(500 * time.Millisecond)

	// Launch a headless browser and navigate to the root
	pwRunner, err := pw.Run()
	if err != nil {
		t.Fatalf("playwright run failed: %v", err)
	}
	defer pwRunner.Stop()

	browser, err := pwRunner.Chromium.Launch()
	if err != nil {
		t.Fatalf("browser launch failed: %v", err)
	}
	defer browser.Close()

	context, err := browser.NewContext()
	if err != nil {
		t.Fatalf("context create failed: %v", err)
	}
	page, err := context.NewPage()
	if err != nil {
		t.Fatalf("page create failed: %v", err)
	}

	resp, err := page.Goto("http://127.0.0.1:8000/", pw.PageGotoOptions{
		WaitUntil: pw.WaitUntilStateDomcontentloaded,
		Timeout:   pw.Float(20000),
	})
	if err != nil {
		t.Fatalf("navigation error: %v", err)
	}
	if resp == nil {
		t.Fatalf("no response from navigation")
	}
	status := resp.Status()
	if status >= 500 {
		t.Fatalf("server returned 5xx: %d", status)
	}

	takeScreenshot(t, page, "hello")
}

func takeScreenshot(t *testing.T, page pw.Page, name string) {
	path := fmt.Sprintf("/app/tmp/screenshots/%s.png", name)

	_, err := page.Screenshot(pw.PageScreenshotOptions{
		Path:     pw.String(path),
		FullPage: pw.Bool(true),
		Type:     pw.ScreenshotTypePng,
	})

	if err != nil {
		t.Fatalf("Failed to take screenshot: %v", err)
	}

	t.Logf("Saved screenshot to %s", path)
}
