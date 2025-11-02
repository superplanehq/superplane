package e2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	pw "github.com/playwright-community/playwright-go"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	spjwt "github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

// TestSession handles per-test actions: db, auth, and page ops.
type TestSession struct {
	t         *testing.T
	context   pw.BrowserContext
	page      pw.Page
	timeoutMs float64

	baseURL string
	orgID   string
	account *models.Account
}

func (s *TestSession) Start() {
	s.resetDatabase()
	s.setupUserAndOrganization()
}

func (s *TestSession) Close() {
	if s.page != nil {
		_ = s.page.Close()
	}
}

func (s *TestSession) Visit(path string) {
	_, err := s.page.Goto(s.baseURL+path, pw.PageGotoOptions{WaitUntil: pw.WaitUntilStateDomcontentloaded, Timeout: pw.Float(s.timeoutMs)})
	if err != nil {
		s.t.Fatalf("goto: %v", err)
	}
}

func (s *TestSession) AssertText(text string) {
	if err := s.page.Locator("text=" + text).WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(s.timeoutMs)}); err != nil {
		s.t.Fatalf("text %q not found: %v", text, err)
	}
}

func (s *TestSession) TakeScreenshot() {
	path := fmt.Sprintf("/app/tmp/screenshots/%s-%d.png", s.t.Name(), time.Now().UnixMilli())
	s.t.Logf("Taking screenshot: %s", path)

	if _, err := s.page.Screenshot(pw.PageScreenshotOptions{Path: pw.String(path), FullPage: pw.Bool(true), Type: pw.ScreenshotTypePng}); err != nil {
		s.t.Fatalf("screenshot: %v", err)
	}
}

func (s *TestSession) Sleep(ms int) {
	s.t.Logf("Sleeping for %d ms", ms)
	time.Sleep(time.Duration(ms) * time.Millisecond)
	s.t.Logf("Woke up after %d ms", ms)
}

func (s *TestSession) resetDatabase() {
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

func (s *TestSession) setupUserAndOrganization() {
	email := "e2e@superplane.local"
	name := "E2E User"
	account, err := models.FindAccountByEmail(email)
	if err != nil {
		account, err = models.CreateAccount(name, email)
		if err != nil {
			s.t.Fatalf("create account: %v", err)
		}
	}

	orgName := "e2e-org"
	organization, err := models.FindOrganizationByName(orgName)
	if err != nil {
		organization, err = models.CreateOrganization(orgName, "")
		if err != nil {
			s.t.Fatalf("create organization: %v", err)
		}
	}

	user, err := models.FindMaybeDeletedUserByEmail(organization.ID.String(), email)
	if err != nil {
		user, err = models.CreateUser(organization.ID, account.ID, email, name)
		if err != nil {
			s.t.Fatalf("create user: %v", err)
		}
	} else if user.DeletedAt.Valid {
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
	s.t.Logf("Clicking button: %q", text)
	selector := fmt.Sprintf("button:has-text(\"%s\"), [role=button]:has-text(\"%s\")", text, text)
	if err := s.page.Locator(selector).First().Click(pw.LocatorClickOptions{Timeout: pw.Float(s.timeoutMs)}); err != nil {
		s.t.Fatalf("click button %q: %v", text, err)
	}
}

func (s *TestSession) FillIn(label, value string) {
	s.t.Logf("Filling in %q with %q", label, value)
	if el := s.page.GetByTestId(label); el != nil {
		if err := el.Fill(value, pw.LocatorFillOptions{Timeout: pw.Float(s.timeoutMs)}); err == nil {
			return
		}
	}
	s.t.Fatalf("fill in %q failed", label)
}

func (s *TestSession) VisitHomePage() {
	s.Visit("/" + s.orgID + "/")
}
