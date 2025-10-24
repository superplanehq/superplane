package integrations

import (
    "context"
    "testing"
    "time"

    pw "github.com/playwright-community/playwright-go"
)

func TestWorkflowPage_Smoke(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    page, err := pwBrowser.NewPage()
    if err != nil {
        t.Fatalf("new page: %v", err)
    }
    defer func() { _ = page.Close() }()

    if _, err := page.Goto(baseURL, pw.PageGotoOptions{WaitUntil: pw.WaitUntilStateNetworkidle}); err != nil {
        t.Fatalf("goto %s: %v", baseURL, err)
    }

    // Basic smoke: page should have a title or root app element
    if title, err := page.Title(); err == nil && title != "" {
        return
    }

    // Fallback: check that some root element exists
    _, err = page.WaitForSelector("body", pw.PageWaitForSelectorOptions{Timeout: pw.Float(5000)})
    if err != nil {
        t.Fatalf("app did not render: %v", err)
    }

    _ = ctx // reserved for future steps
}
