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

	c, err := b.NewContext()
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

func (s *TestContext) setUpNavigationLogger() {
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

		panic("init script: " + err.Error())
	}
}

func (s *TestContext) NewSession(t *testing.T) *TestSession {
	p, err := s.context.NewPage()
	if err != nil {
		t.Fatalf("page: %v", err)
	}

	sess := &TestSession{
		t:         t,
		context:   s.context,
		page:      p,
		timeoutMs: s.timeoutMs,
		baseURL:   s.baseURL,
	}

	// Expose SPA navigation logger and page-level hooks
	if err := p.ExposeFunction("_spNav", func(args ...interface{}) interface{} {
		if len(args) > 0 {
			if url, ok := args[0].(string); ok {
				t.Logf("[Browser Logs] Navigated to %s", url)
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("expose function: %v", err)
	}

	p.OnFrameNavigated(func(f pw.Frame) {
		if f.ParentFrame() == nil {
			t.Logf("[Browser Logs] Navigated to %s", f.URL())
		}
	})

	p.OnConsole(func(m pw.ConsoleMessage) {
		t.Logf("[console.%s] %s", m.Type(), m.Text())
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
