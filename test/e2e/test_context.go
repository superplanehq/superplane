package e2e

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	pw "github.com/playwright-community/playwright-go"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	spjwt "github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/server"
)

type TestContext struct {
	t         *testing.T
	runner    *pw.Playwright
	browser   pw.Browser
	context   pw.BrowserContext
	page      pw.Page
	timeoutMs float64
	viteCmd   *exec.Cmd

	orgID   string
	baseURL string
	account *models.Account
}

func NewTestContext(t *testing.T) *TestContext {
	return &TestContext{t: t, timeoutMs: 10000}
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
	s.streamBrowserLogs()
	s.resetDatabase()
	s.setupUserAndOrganization()

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

	p, err := c.NewPage()
	if err != nil {
		s.t.Fatalf("page: %v", err)
	}

	s.browser = b
	s.context = c
	s.page = p
}

func (s *TestContext) startAppServer() {
	go server.Start()
	time.Sleep(500 * time.Millisecond)
	s.baseURL = os.Getenv("BASE_URL")
}

func (s *TestContext) Visit(path string) {
	_, err := s.page.Goto(s.baseURL+path, pw.PageGotoOptions{WaitUntil: pw.WaitUntilStateDomcontentloaded, Timeout: pw.Float(s.timeoutMs)})
	if err != nil {
		s.t.Fatalf("goto: %v", err)
	}
}

func (s *TestContext) AssertText(text string) {
	if err := s.page.Locator("text=" + text).WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(s.timeoutMs)}); err != nil {
		s.t.Fatalf("text %q not found: %v", text, err)
	}
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

func (s *TestContext) TakeScreenshot() {
	path := fmt.Sprintf("/app/tmp/screenshots/%s-%d.png", s.t.Name(), time.Now().UnixMilli())
	s.t.Logf("Taking screenshot: %s", path)

	if _, err := s.page.Screenshot(pw.PageScreenshotOptions{Path: pw.String(path), FullPage: pw.Bool(true), Type: pw.ScreenshotTypePng}); err != nil {
		s.t.Fatalf("screenshot: %v", err)
	}
}

func (s *TestContext) Sleep(ms int) {
	s.t.Logf("Sleeping for %d ms", ms)
	time.Sleep(time.Duration(ms) * time.Millisecond)
	s.t.Logf("Woke up after %d ms", ms)
}

func (s *TestContext) resetDatabase() {
	//
	// resetDatabase truncates all public tables (except migration tables),
	// restarting identities and cascading to maintain referential integrity.
	//
	sql := `DO $$
    DECLARE r RECORD;
    BEGIN
        FOR r IN (
            SELECT tablename
            FROM pg_tables
            WHERE schemaname = 'public'
              AND tablename NOT IN ('schema_migrations')
        ) LOOP
            EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' RESTART IDENTITY CASCADE';
        END LOOP;
    END$$;`

	if err := database.Conn().Exec(sql).Error; err != nil {
		s.t.Fatalf("reset database: %v", err)
	}
}

func (s *TestContext) Login() {
	// Authenticate via the account cookie
	secret := os.Getenv("JWT_SECRET")
	signer := spjwt.NewSigner(secret)
	token, err := signer.Generate(s.account.ID.String(), 24*time.Hour)
	if err != nil {
		s.t.Fatalf("jwt: %v", err)
	}

	if err := s.context.AddCookies([]pw.OptionalCookie{{
		Name:     "account_token",
		Value:    token,
		URL:      pw.String(s.baseURL + "/"),
		HttpOnly: pw.Bool(true),
	}}); err != nil {
		s.t.Fatalf("add cookie: %v", err)
	}
}

// setupUserAndOrganization ensures there is an account, an organization,
// and a user (member of that org), then authenticates the browser context.
func (s *TestContext) setupUserAndOrganization() {
	// Ensure an account exists
	email := "e2e@superplane.local"
	name := "E2E User"
	account, err := models.FindAccountByEmail(email)
	if err != nil {
		account, err = models.CreateAccount(name, email)
		if err != nil {
			s.t.Fatalf("create account: %v", err)
		}
	}

	// Ensure an organization exists
	orgName := "e2e-org"
	organization, err := models.FindOrganizationByName(orgName)
	if err != nil {
		organization, err = models.CreateOrganization(orgName, "")
		if err != nil {
			s.t.Fatalf("create organization: %v", err)
		}
	}

	// Ensure the user exists in that organization
	user, err := models.FindMaybeDeletedUserByEmail(organization.ID.String(), email)
	if err != nil {
		user, err = models.CreateUser(organization.ID, account.ID, email, name)
		if err != nil {
			s.t.Fatalf("create user: %v", err)
		}
	} else if user.DeletedAt.Valid {
		// Restore soft-deleted user just in case
		if err := user.Restore(); err != nil {
			s.t.Fatalf("restore user: %v", err)
		}
	}

	if svc, err := authorization.NewAuthService(); err == nil {
		_ = svc.SetupOrganizationRoles(organization.ID.String())
		_ = svc.CreateOrganizationOwner(user.ID.String(), organization.ID.String())
	}

	s.orgID = organization.ID.String()
	s.account = account
}

func (s *TestContext) ClickButton(text string) {
	s.t.Logf("Clicking button: %q", text)

	selector := fmt.Sprintf("button:has-text(\"%s\"), [role=button]:has-text(\"%s\")", text, text)
	if err := s.page.Locator(selector).First().Click(pw.LocatorClickOptions{Timeout: pw.Float(s.timeoutMs)}); err != nil {
		s.t.Fatalf("click button %q: %v", text, err)
	}
}

func (s *TestContext) FillIn(label, value string) {
	s.t.Logf("Filling in %q with %q", label, value)

	if el := s.page.GetByTestId(label); el != nil {
		if err := el.Fill(value, pw.LocatorFillOptions{Timeout: pw.Float(s.timeoutMs)}); err == nil {
			return
		}
	}

	s.t.Fatalf("fill in %q failed", label)
}

func (s *TestContext) VisitHomePage() {
	s.Visit("/" + s.orgID + "/")
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
	// Expose a logging function to capture SPA (client-side) URL changes
	if err := s.page.ExposeFunction("_spNav", func(args ...interface{}) interface{} {
		if len(args) > 0 {
			if url, ok := args[0].(string); ok {
				s.t.Logf("[Browser Logs] Navigated to %s", url)
			}
		}
		return nil
	}); err != nil {
		s.t.Fatalf("expose function: %v", err)
	}

	// Inject a script that hooks into the History API and URL change events
	// to report client-side navigations (e.g., react-router)
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

	// Log when the main frame navigates to a new URL (full navigations)
	s.page.OnFrameNavigated(func(f pw.Frame) {
		if f.ParentFrame() == nil { // main frame only
			s.t.Logf("[Browser Logs] Navigated to %s", f.URL())
		}
	})
}

func (s *TestContext) streamBrowserLogs() {
	// Stream browser console and network issues to test output
	s.page.OnConsole(func(m pw.ConsoleMessage) {
		s.t.Logf("[console.%s] %s", m.Type(), m.Text())
	})

	// Non-navigation logs (leave here): console, errors, requests, responses

	s.page.OnPageError(func(err error) {
		s.t.Logf("[Browser Logs] %v", err)
	})

	s.page.OnRequestFailed(func(r pw.Request) {
		if err := r.Failure(); err != nil {
			// Ignore common benign cancellations during SPA navigations/HMR
			if strings.Contains(err.Error(), "ERR_ABORTED") {
				return
			}
			s.t.Logf("[Browser Logs] %s (%s)", r.URL(), err.Error())
			return
		}
		// No explicit failure info
		s.t.Logf("[Browser Logs] %s (request failed)", r.URL())
	})

	s.page.OnResponse(func(resp pw.Response) {
		status := resp.Status()
		if status >= 400 {
			s.t.Logf("[Browser Logs] %d %s", status, resp.URL())
		}
	})
}
