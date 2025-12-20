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
	"github.com/superplanehq/superplane/test/e2e/session"
)

type TestContext struct {
	runner    *pw.Playwright
	browser   pw.Browser
	context   pw.BrowserContext
	timeoutMs float64
	viteCmd   *exec.Cmd

	baseURL string
}

func NewTestContext(t *testing.M) *TestContext {
	return &TestContext{timeoutMs: 10000}
}

func (s *TestContext) Start() {
	os.Setenv("DB_NAME", "superplane_test")
	os.Setenv("START_PUBLIC_API", "yes")
	os.Setenv("START_INTERNAL_API", "yes")
	os.Setenv("INTERNAL_API_PORT", "50052")
	os.Setenv("PUBLIC_API_BASE_PATH", "/api/v1")
	os.Setenv("START_WEB_SERVER", "yes")
	os.Setenv("WEB_BASE_PATH", "")
	os.Setenv("START_GRPC_GATEWAY", "yes")
	os.Setenv("GRPC_SERVER_ADDR", "127.0.0.1:50052")
	os.Setenv("START_EVENT_DISTRIBUTER", "yes")
	os.Setenv("START_CONSUMERS", "yes")
	os.Setenv("START_WORKFLOW_EVENT_ROUTER", "yes")
	os.Setenv("START_WORKFLOW_NODE_EXECUTOR", "yes")
	os.Setenv("START_BLUEPRINT_NODE_EXECUTOR", "yes")
	os.Setenv("START_WORKFLOW_NODE_QUEUE_WORKER", "yes")
	os.Setenv("START_NODE_REQUEST_WORKER", "yes")
	os.Setenv("START_WEBHOOK_PROVISIONER", "yes")
	os.Setenv("START_WEBHOOK_CLEANUP_WORKER", "yes")
	os.Setenv("NO_ENCRYPTION", "yes")
	os.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	os.Setenv("JWT_SECRET", "test-jwt-secret")
	os.Setenv("PUBLIC_API_PORT", "8001")
	os.Setenv("BASE_URL", "http://127.0.0.1:8001")
	os.Setenv("WEBHOOKS_BASE_URL", "https://superplane.sxmoon.com")
	os.Setenv("APP_ENV", "development")
	os.Setenv("OWNER_SETUP_ENABLED", "yes")

	s.startVite()
	s.startAppServer()
	s.startPlaywright()
	s.launchBrowser()
	s.setUpNavigationLogger()
}

func (s *TestContext) startPlaywright() {
	r, err := pw.Run()
	if err != nil {
		panic("playwright: " + err.Error())
	}

	s.runner = r
}

func (s *TestContext) launchBrowser() {
	b, err := s.runner.Chromium.Launch()
	if err != nil {
		panic("browser launch: " + err.Error())
	}

	c, err := b.NewContext(pw.BrowserNewContextOptions{
		Viewport: &pw.Size{
			Width:  2560,
			Height: 1440,
		},
	})
	if err != nil {
		panic("browser context: " + err.Error())
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
		panic("start vite: " + err.Error())
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

const initScript = `
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

			// Auto-accept all confirm dialogs in tests
			try {
				const originalConfirm = window.confirm;
				window.confirm = function(message) {
					return true;
				};
				window._spOriginalConfirm = originalConfirm;
			} catch (_) {
				// ignore
			}

			// Initial report
			if (window._spNav) { window._spNav(location.href); }
		} catch (_) { /* ignore */ }
	})();
`

func (s *TestContext) setUpNavigationLogger() {
	if err := s.context.AddInitScript(pw.Script{Content: pw.String(initScript)}); err != nil {
		panic("init script: " + err.Error())
	}
}

func (s *TestContext) NewSession(t *testing.T) *session.TestSession {
	p, err := s.context.NewPage()
	if err != nil {
		t.Fatalf("page: %v", err)
	}

	sess := session.NewTestSession(t, s.context, p, s.timeoutMs, s.baseURL)

	p.OnConsole(func(m pw.ConsoleMessage) {
		text := m.Text()

		// Ignore noisy dev-time logs from Vite and React DevTools suggestions
		if strings.Contains(text, "[vite] connecting") ||
			strings.Contains(text, "[vite] connected") ||
			strings.Contains(text, "React DevTools") ||
			strings.Contains(text, "Download the React DevTools") {
			return
		}

		t.Logf("[console.%s] %s", m.Type(), text)
	})

	p.OnPageError(func(err error) {
		t.Logf("[Browser Logs] %v", err)
	})

	p.OnRequestFailed(func(r pw.Request) {
		if err := r.Failure(); err != nil {
			if strings.Contains(err.Error(), "ERR_ABORTED") {
				return
			}
			t.Logf("[Browser Logs] %s (%s)", r.URL(), err.Error())
			return
		}
		t.Logf("[Browser Logs] %s (request failed)", r.URL())
	})

	p.OnResponse(func(resp pw.Response) {
		if status := resp.Status(); status >= 400 {
			t.Logf("[Browser Logs] %d %s", status, resp.URL())
		}
	})

	return sess
}
