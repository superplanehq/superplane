package e2e

import (
    "io"
    "net/http"
    "os"
    "os/exec"
    "strings"
    "testing"
    "time"

    pw "github.com/playwright-community/playwright-go"
    "github.com/superplanehq/superplane/pkg/server"
)

type TestContext struct {
	t         *testing.T
	runner    *pw.Playwright
	browser   pw.Browser
	context   pw.BrowserContext
	timeoutMs float64
	viteCmd   *exec.Cmd

	baseURL string
}

func NewTestContext(t *testing.T) *TestContext { return &TestContext{t: t, timeoutMs: 10000} }

func (s *TestContext) Start() {
	os.Setenv("START_PUBLIC_API", "yes")
	os.Setenv("START_INTERNAL_API", "yes")
	os.Setenv("PUBLIC_API_BASE_PATH", "/api/v1")
	os.Setenv("START_WEB_SERVER", "yes")
	os.Setenv("WEB_BASE_PATH", "")
	os.Setenv("START_GRPC_GATEWAY", "yes")
	os.Setenv("CASBIN_AUTO_RELOAD", "yes")
	os.Setenv("START_EVENT_DISTRIBUTER", "no")
	os.Setenv("NO_ENCRYPTION", "yes")
	os.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	os.Setenv("JWT_SECRET", "test-jwt-secret")
	os.Setenv("PUBLIC_API_PORT", "8001")
	os.Setenv("BASE_URL", "http://127.0.0.1:8001")
	os.Setenv("APP_ENV", "development")

	s.startVite()
	s.startAppServer()
	s.startPlaywright()
	s.launchBrowser()

	s.setUpNavigationLogger()
	s.streamBrowserLogs()

}

func (s *TestContext) startPlaywright() {
	r, err := pw.Run()
	if err != nil {
		s.t.Fatalf("playwright: %v", err)
	}

	s.runner = r
}

func (s *TestContext) launchBrowser() {
	b, err := s.runner.Chromium.Launch()
	if err != nil {
		s.t.Fatalf("browser: %v", err)
	}

	c, err := b.NewContext()
	if err != nil {
		s.t.Fatalf("context: %v", err)
	}

	s.browser = b
	s.context = c
}

func (s *TestContext) startAppServer() {
	go server.Start()
	time.Sleep(500 * time.Millisecond)
	s.baseURL = os.Getenv("BASE_URL")
}

func (s *TestContext) Shutdown() {
	if s.browser != nil {
		s.browser.Close()
	}
	if s.runner != nil {
		s.runner.Stop()
	}
	if s.viteCmd != nil && s.viteCmd.Process != nil {
		_ = s.viteCmd.Process.Kill()
	}
}

func (s *TestContext) startVite() {
	cmd := exec.Command("npm", "run", "dev", "--", "--host", "127.0.0.1", "--port", "5173")
	cmd.Dir = "../../web_src"
	// Point Vite proxy at the test server's API port
	cmd.Env = append(os.Environ(), "BROWSER=none", "API_PORT=8001")

	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		s.t.Fatalf("vite start: %v", err)
	}

	s.viteCmd = cmd

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://127.0.0.1:5173/")
		if err == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (s *TestContext) setUpNavigationLogger() {
	// Inject a script that hooks into the History API and URL change events
	// to report client-side navigations (e.g., react-router). The receiver
	// function (window._spNav) is exposed per-page in NewSession().
	if err := s.context.AddInitScript(pw.Script{Content: pw.String(`
        (() => {
          try {
            let last = location.href;
            const notify = () => {
              const href = location.href;
              if (href !== last) {
                last = href;
                if (window._spNav) {
                  window._spNav(href);
                }
              }
            };
            const push = history.pushState;
            const replace = history.replaceState;
            history.pushState = function(...args){ const r = push.apply(this, args); notify(); return r; };
            history.replaceState = function(...args){ const r = replace.apply(this, args); notify(); return r; };
            window.addEventListener('popstate', notify);
            window.addEventListener('hashchange', notify);
            // Initial report
            if (window._spNav) { window._spNav(location.href); }
          } catch (_) { /* ignore */ }
        })();
    `)}); err != nil {
		s.t.Fatalf("init script: %v", err)
	}

	// Note: page-level logging attached in NewSession()
}
func (s *TestContext) streamBrowserLogs() { /* per-session page handlers */ }

// NewSession creates a new browser page bound to this context.
func (s *TestContext) NewSession() *TestSession {
	p, err := s.context.NewPage()
	if err != nil {
		s.t.Fatalf("page: %v", err)
	}
	sess := &TestSession{
		t:         s.t,
		context:   s.context,
		page:      p,
		timeoutMs: s.timeoutMs,
		baseURL:   s.baseURL,
	}

	// Expose SPA navigation logger and page-level hooks
	if err := p.ExposeFunction("_spNav", func(args ...interface{}) interface{} {
		if len(args) > 0 {
			if url, ok := args[0].(string); ok {
				s.t.Logf("[Browser Logs] Navigated to %s", url)
			}
		}
		return nil
	}); err != nil {
		s.t.Fatalf("expose function: %v", err)
	}

	// Page-level logging hooks
	p.OnFrameNavigated(func(f pw.Frame) {
		if f.ParentFrame() == nil {
			s.t.Logf("[Browser Logs] Navigated to %s", f.URL())
		}
	})
	p.OnConsole(func(m pw.ConsoleMessage) {
		s.t.Logf("[console.%s] %s", m.Type(), m.Text())
	})
	p.OnPageError(func(err error) { s.t.Logf("[Browser Logs] %v", err) })
	p.OnRequestFailed(func(r pw.Request) {
		if err := r.Failure(); err != nil {
			if strings.Contains(err.Error(), "ERR_ABORTED") {
				return
			}
			s.t.Logf("[Browser Logs] %s (%s)", r.URL(), err.Error())
			return
		}
		s.t.Logf("[Browser Logs] %s (request failed)", r.URL())
	})
	p.OnResponse(func(resp pw.Response) {
		if status := resp.Status(); status >= 400 {
			s.t.Logf("[Browser Logs] %d %s", status, resp.URL())
		}
	})

	return sess
}
