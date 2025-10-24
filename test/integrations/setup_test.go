package integrations

import (
    "os"
    "testing"
    "time"

    pw "github.com/playwright-community/playwright-go"
)

var (
    baseURL   = envOr("BASE_URL", "http://localhost:8000")
    pwBrowser pw.Browser
    pwInst    *pw.Playwright
)

func TestMain(m *testing.M) {
    var err error
    // Assume Playwright and browsers are preinstalled via `make e2e.setup`
    pwInst, err = pw.Run()
    if err != nil {
        panic(err)
    }
    defer func() {
        if pwInst != nil {
            _ = pwInst.Stop()
        }
    }()

    pwBrowser, err = pwInst.Chromium.Launch(pw.BrowserTypeLaunchOptions{
        Headless: pw.Bool(true),
    })
    if err != nil {
        panic(err)
    }
    defer func() { _ = pwBrowser.Close() }()

    // Best-effort wait for server to be reachable
    waitForServer(baseURL, 30*time.Second)

    code := m.Run()
    os.Exit(code)
}

func waitForServer(url string, timeout time.Duration) {
    // Lightweight poll using Playwright context to avoid extra deps
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if pwBrowser != nil {
            page, err := pwBrowser.NewPage()
            if err == nil {
                _, err = page.Goto(url, pw.PageGotoOptions{Timeout: pw.Float(2000)})
                _ = page.Close()
                if err == nil {
                    return
                }
            }
        }
        time.Sleep(500 * time.Millisecond)
    }
}

func envOr(k, def string) string {
    if v := os.Getenv(k); v != "" {
        return v
    }
    return def
}
