package e2e

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	pw "github.com/playwright-community/playwright-go"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	spjwt "github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/server"
)

type TestSession struct {
	t         *testing.T
	runner    *pw.Playwright
	browser   pw.Browser
	context   pw.BrowserContext
	page      pw.Page
	timeoutMs float64
	viteCmd   *exec.Cmd

	orgID   string
	account *models.Account
}

func NewTestSession(t *testing.T) *TestSession {
	return &TestSession{t: t, timeoutMs: 10000}
}

func (s *TestSession) Start() {
	os.Setenv("START_PUBLIC_API", "yes")
	os.Setenv("PUBLIC_API_BASE_PATH", "/api/v1")
	os.Setenv("START_WEB_SERVER", "yes")
	os.Setenv("WEB_BASE_PATH", "")
	os.Setenv("START_GRPC_GATEWAY", "no")
	os.Setenv("START_EVENT_DISTRIBUTER", "no")
	os.Setenv("NO_ENCRYPTION", "yes")
	os.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	os.Setenv("JWT_SECRET", "test-jwt-secret")
	os.Setenv("BASE_URL", "http://127.0.0.1:8000")

	os.Setenv("QUIET_PROXY_LOGS", "yes")
	os.Setenv("QUIET_HTTP_LOGS", "yes")

	os.Setenv("APP_ENV", "development")
	_ = os.Unsetenv("ASSETS_ROOT")
	s.startVite()

	go server.Start()
	time.Sleep(500 * time.Millisecond)

	r, err := pw.Run()
	if err != nil {
		s.t.Fatalf("playwright: %v", err)
	}
	s.runner = r

	b, err := r.Chromium.Launch()
	if err != nil {
		s.t.Fatalf("browser: %v", err)
	}
	s.browser = b

	c, err := b.NewContext()
	if err != nil {
		s.t.Fatalf("context: %v", err)
	}
	s.context = c

	p, err := c.NewPage()
	if err != nil {
		s.t.Fatalf("page: %v", err)
	}
	s.page = p

	// Stream browser console and network issues to test output
	s.page.OnConsole(func(m pw.ConsoleMessage) {
		s.t.Logf("[console.%s] %s", m.Type(), m.Text())
	})

	s.page.OnPageError(func(err error) {
		s.t.Logf("[Browser Logs] %v", err)
	})

	s.page.OnRequestFailed(func(r pw.Request) {
		reason := ""
		if err := r.Failure(); err != nil {
			reason = err.Error()
		}
		s.t.Logf("[Browser Logs] %s (%s)", r.URL(), reason)
	})

	s.page.OnResponse(func(resp pw.Response) {
		status := resp.Status()
		if status >= 400 {
			s.t.Logf("[Browser Logs] %d %s", status, resp.URL())
		}
	})

	s.resetDatabase()
	s.setupUserAndOrganization()
}

func (s *TestSession) Visit(path string) {
	_, err := s.page.Goto("http://127.0.0.1:8000"+path, pw.PageGotoOptions{WaitUntil: pw.WaitUntilStateDomcontentloaded, Timeout: pw.Float(s.timeoutMs)})
	if err != nil {
		s.t.Fatalf("goto: %v", err)
	}
}

func (s *TestSession) AssertText(text string) {
	if err := s.page.Locator("text=" + text).WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(s.timeoutMs)}); err != nil {
		s.t.Fatalf("text %q not found: %v", text, err)
	}
}

func (s *TestSession) Shutdown() {
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

func (s *TestSession) TakeScreenshot() {
	path := fmt.Sprintf("/app/tmp/screenshots/%s-%d.png", s.t.Name(), time.Now().UnixMilli())
	if _, err := s.page.Screenshot(pw.PageScreenshotOptions{Path: pw.String(path), FullPage: pw.Bool(true), Type: pw.ScreenshotTypePng}); err != nil {
		s.t.Fatalf("screenshot: %v", err)
	}
}

func (s *TestSession) Sleep(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func (s *TestSession) resetDatabase() {
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

func (s *TestSession) Login() {
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
		URL:      pw.String("http://127.0.0.1:8000/"),
		HttpOnly: pw.Bool(true),
	}}); err != nil {
		s.t.Fatalf("add cookie: %v", err)
	}
}

// setupUserAndOrganization ensures there is an account, an organization,
// and a user (member of that org), then authenticates the browser context.
func (s *TestSession) setupUserAndOrganization() {
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

func (s *TestSession) ClickButton(text string) {
	selector := fmt.Sprintf("button:has-text(\"%s\"), [role=button]:has-text(\"%s\")", text, text)
	if err := s.page.Locator(selector).First().Click(pw.LocatorClickOptions{Timeout: pw.Float(s.timeoutMs)}); err != nil {
		s.t.Fatalf("click button %q: %v", text, err)
	}
}

func (s *TestSession) FillIn(label, value string) {
	// Try by accessible name via role=textbox
	if el := s.page.GetByRole("textbox", pw.PageGetByRoleOptions{Name: label}); el != nil {
		if err := el.Fill(value, pw.LocatorFillOptions{Timeout: pw.Float(s.timeoutMs)}); err == nil {
			return
		}
	}

	// Fallback: try input/textarea with matching placeholder
	selector := fmt.Sprintf("input[placeholder=\"%s\"], textarea[placeholder=\"%s\"]", label, label)
	if err := s.page.Locator(selector).First().Fill(value, pw.LocatorFillOptions{Timeout: pw.Float(s.timeoutMs)}); err == nil {
		return
	}

	// Fallback: try label text proximity (label immediately before input/textarea)
	neighbor := fmt.Sprintf("label:has-text(\"%s\")", label)
	loc := s.page.Locator(neighbor).First()
	// Attempt to find the next input/textarea in DOM order
	if err := s.page.Locator(neighbor+" >> xpath=following::input[1]").First().Fill(value, pw.LocatorFillOptions{Timeout: pw.Float(s.timeoutMs)}); err == nil {
		return
	}
	if err := s.page.Locator(neighbor+" >> xpath=following::textarea[1]").First().Fill(value, pw.LocatorFillOptions{Timeout: pw.Float(s.timeoutMs)}); err == nil {
		return
	}

	// If we got here, we failed all strategies
	box, _ := loc.TextContent()
	s.t.Fatalf("fill in %q failed; last label text: %q", label, box)
}

func (s *TestSession) VisitHomePage() {
	s.Visit("/" + s.orgID + "/")
}

func (s *TestSession) startVite() {
	cmd := exec.Command("npm", "run", "dev", "--", "--host", "127.0.0.1", "--port", "5173")
	cmd.Dir = "../../web_src"
	cmd.Env = append(os.Environ(), "BROWSER=none")

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
