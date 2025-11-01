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
	console   []string
	neterrs   []string
	orgID     string
}

func NewTestSession(t *testing.T) *TestSession {
	return &TestSession{t: t, timeoutMs: 10000}
}

func (s *TestSession) Start() {
	// Common server env
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

	// Always use Vite dev server and proxy in development mode
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

	// Capture console and network failures
	s.page.OnConsole(func(m pw.ConsoleMessage) { s.console = append(s.console, fmt.Sprintf("[%s] %s", m.Type(), m.Text())) })
	s.page.OnPageError(func(e error) { s.console = append(s.console, fmt.Sprintf("[pageerror] %v", e)) })
	s.page.OnRequestFailed(func(r pw.Request) { s.neterrs = append(s.neterrs, fmt.Sprintf("FAIL %s %s", r.Method(), r.URL())) })

	// Prepare an account, organization, and user, then authenticate via cookie
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

// Sleep pauses the test for the given milliseconds (avoid; prefer WaitForNetworkIdle).
func (s *TestSession) Sleep(ms int) { time.Sleep(time.Duration(ms) * time.Millisecond) }

// WaitForNetworkIdle waits until Playwright observes network idle.
// Uses the page load state 'networkidle' with the session timeout.
func (s *TestSession) WaitForNetworkIdle() {
	if err := s.page.WaitForLoadState(pw.PageWaitForLoadStateOptions{
		State:   pw.LoadStateNetworkidle,
		Timeout: pw.Float(s.timeoutMs),
	}); err != nil {
		s.t.Fatalf("wait for network idle: %v", err)
	}
}

func (s *TestSession) Login() {
	// Backward compatibility: keep calling the new setup
	s.setupUserAndOrganization()
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

	// Persist for convenience when visiting org-scoped routes
	s.orgID = organization.ID.String()

	// Create default org roles and assign owner to the user (best-effort)
	// This mirrors server org bootstrap but tests can proceed even if this fails.
	// Avoid introducing a hard dependency if RBAC files are not configured.
	if svc, err := authorization.NewAuthService(); err == nil {
		_ = svc.SetupOrganizationRoles(organization.ID.String())
		_ = svc.CreateOrganizationOwner(user.ID.String(), organization.ID.String())
	}

	// Authenticate via the account cookie
	secret := os.Getenv("JWT_SECRET")
	signer := spjwt.NewSigner(secret)
	token, err := signer.Generate(account.ID.String(), 24*time.Hour)
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

// ClickButton finds and clicks a button by its visible text.
func (s *TestSession) ClickButton(text string) {
	selector := fmt.Sprintf("button:has-text(\"%s\"), [role=button]:has-text(\"%s\")", text, text)
	if err := s.page.Locator(selector).First().Click(pw.LocatorClickOptions{Timeout: pw.Float(s.timeoutMs)}); err != nil {
		s.t.Fatalf("click button %q: %v", text, err)
	}
}

// FillIn fills a text input or textarea identified by its accessible label
// (preferred) or placeholder text as a fallback.
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

// VisitHomePage navigates to the org-scoped home route.
func (s *TestSession) VisitHomePage() {
	if s.orgID == "" {
		s.t.Fatalf("VisitHomePage called before org/user setup")
	}
	s.Visit("/" + s.orgID + "/")
}

func (s *TestSession) DumpBrowserLogs() {
	if len(s.console) > 0 {
		s.t.Logf("Console logs:\n%s", strings.Join(s.console, "\n"))
	}
	if len(s.neterrs) > 0 {
		s.t.Logf("Network failures:\n%s", strings.Join(s.neterrs, "\n"))
	}
}

// DumpPageSource saves the current page HTML to a file and logs the path.
func (s *TestSession) DumpPageSource() {
	content, err := s.page.Content()
	if err != nil {
		s.t.Logf("could not get page content: %v", err)
		return
	}
	s.t.Logf("Page source (len=%d):\n%s", len(content), content)
}

func (s *TestSession) startVite() {
	cmd := exec.Command("npm", "run", "dev", "--", "--host", "127.0.0.1", "--port", "5173")
	cmd.Dir = "../../web_src"
	cmd.Env = append(os.Environ(), "BROWSER=none")
	// Keep logs quiet in tests
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		s.t.Fatalf("vite start: %v", err)
	}
	s.viteCmd = cmd

	// Wait until Vite is listening
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
