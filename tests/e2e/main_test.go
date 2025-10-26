package e2e

import (
    "os"
    "testing"

    pw "github.com/playwright-community/playwright-go"
)

var (
    pwRunner *pw.Playwright
    appURL   string
    headless bool
)

// TestMain sets up Playwright once for the e2e package and
// tears it down after all tests. Individual tests can launch
// browsers using the shared runner.
func TestMain(m *testing.M) {
    // Ensure browsers are installed (no-op if already present)
    if err := pw.Install(); err != nil {
        // If install fails, exit non-zero so tests are marked failed.
        // Testing framework is not initialized yet, so print and exit.
        println("failed to install playwright:", err.Error())
        os.Exit(1)
    }

    var err error
    pwRunner, err = pw.Run()
    if err != nil {
        println("failed to run playwright:", err.Error())
        os.Exit(1)
    }

    // Shared config derived from env
    appURL = os.Getenv("APP_URL")
    if appURL == "" {
        appURL = "http://localhost:8000"
    }
    headless = os.Getenv("E2E_HEADFUL") == ""

    code := m.Run()

    if pwRunner != nil {
        _ = pwRunner.Stop()
    }

    os.Exit(code)
}

// newBrowserPage returns a fresh browser and page using shared settings.
// Caller is responsible for closing both (use returned closers in defer).
func newBrowserPage(t *testing.T) (pw.Browser, pw.Page) {
    t.Helper()
    if pwRunner == nil {
        t.Fatalf("playwright runner not initialized")
    }
    browser, err := pwRunner.Chromium.Launch(pw.BrowserTypeLaunchOptions{Headless: &headless})
    if err != nil {
        t.Fatalf("failed to launch browser: %v", err)
    }
    page, err := browser.NewPage()
    if err != nil {
        _ = browser.Close()
        t.Fatalf("failed to create page: %v", err)
    }
    return browser, page
}

// BrowserSession provides a tiny fluent API for e2e tests.
type BrowserSession struct {
    t       *testing.T
    browser pw.Browser
    page    pw.Page
}

// startBrowser creates a new session. Call Close() when done (or use defer).
func startBrowser(t *testing.T) *BrowserSession {
    t.Helper()
    b, p := newBrowserPage(t)
    return &BrowserSession{t: t, browser: b, page: p}
}

func (s *BrowserSession) Close() {
    _ = s.page.Close()
    _ = s.browser.Close()
}

// Visit navigates to a path relative to appURL or an absolute URL.
func (s *BrowserSession) Visit(path string) *BrowserSession {
    s.t.Helper()
    url := path
    if !(len(path) >= 4 && (path[:4] == "http")) {
        // ensure leading slash
        if path == "" || path[0] != '/' {
            path = "/" + path
        }
        url = appURL + path
    }
    if _, err := s.page.Goto(url, pw.PageGotoOptions{}); err != nil {
        s.t.Fatalf("failed to goto %s: %v", url, err)
    }
    return s
}

// AssertHTMLTitle checks the page title equals expected.
func (s *BrowserSession) AssertHTMLTitle(expected string) *BrowserSession {
    s.t.Helper()
    title, err := s.page.Title()
    if err != nil {
        s.t.Fatalf("failed to get title: %v", err)
    }
    if title != expected {
        s.t.Fatalf("unexpected title. got %q want %q", title, expected)
    }
    return s
}

// AssertText waits for visible text containing the given substring.
func (s *BrowserSession) AssertText(substr string) *BrowserSession {
    s.t.Helper()
    // Use :text matcher via locator with contains semantics
    // Fallback to simple locator search if not available.
    locator := s.page.Locator("text=" + substr)
    if _, err := locator.First().WaitFor(pw.LocatorWaitForOptions{}); err != nil {
        s.t.Fatalf("failed to find text %q: %v", substr, err)
    }
    return s
}

