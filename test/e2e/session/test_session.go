package session

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	spjwt "github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/test/e2e/queries"
)

// TestSession handles per-test actions: db, auth, and page ops.
type TestSession struct {
	t         *testing.T
	context   pw.BrowserContext
	page      pw.Page
	timeoutMs float64

	BaseURL string
	OrgID   uuid.UUID
	OrgSlug string
	Account *models.Account
}

func NewTestSession(t *testing.T, context pw.BrowserContext, page pw.Page, timeoutMs float64, baseURL string) *TestSession {
	sess := &TestSession{
		t:         t,
		context:   context,
		page:      page,
		timeoutMs: timeoutMs,
		BaseURL:   baseURL,
	}

	t.Cleanup(func() {
		if t.Failed() {
			sess.TakeScreenshot()
		}
		sess.Close()
	})

	return sess
}

func (s *TestSession) Start() {
	s.resetDatabase()
	s.setupUserAndOrganization()
	middleware.MarkOwnerSetupCompleted()
}

// StartWithoutUser resets the database but does not seed any user or
// organization records. This is useful for flows that expect an empty
// instance, such as the first-run owner setup.
func (s *TestSession) StartWithoutUser() {
	s.resetDatabase()
}

func (s *TestSession) Close() {
	if s.page != nil {
		// Close the page first to finalize the video
		videoPath, _ := s.page.Video().Path()
		_ = s.page.Close()

		// If test failed, keep the video, otherwise delete it
		if !s.t.Failed() && videoPath != "" {
			_ = os.Remove(videoPath)
		}
	}
}

// Page exposes the underlying playwright Page to satisfy queries.Runner.
func (s *TestSession) Page() pw.Page { return s.page }

func (s *TestSession) Visit(path string) {
	_, err := s.page.Goto(s.BaseURL+path, pw.PageGotoOptions{WaitUntil: pw.WaitUntilStateDomcontentloaded, Timeout: pw.Float(s.timeoutMs)})
	if err != nil {
		s.t.Fatalf("goto: %v", err)
	}
}

func (s *TestSession) AssertText(text string) {
	locator := s.page.Locator("text=" + text).First()
	if err := locator.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(s.timeoutMs)}); err != nil {
		s.t.Fatalf("text %q not found: %v", text, err)
	}
}

func (s *TestSession) TakeScreenshot() {
	path := fmt.Sprintf("/app/tmp/screenshots/%s-%d.png", s.t.Name(), time.Now().UnixMilli())
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		s.t.Logf("screenshot mkdir %s: %v", dir, err)
		return
	}

	s.t.Logf("Taking screenshot: %s", path)

	if _, err := s.page.Screenshot(pw.PageScreenshotOptions{
		Path:     pw.String(path),
		FullPage: pw.Bool(true),
		Type:     pw.ScreenshotTypePng,
	}); err != nil {
		s.t.Logf("screenshot error: %v", err)
	}
}

// WaitUntil polls condition until it is true or the session timeout expires.
func (s *TestSession) WaitUntil(condition func() bool, message string) {
	require.Eventually(s.t, condition, time.Duration(s.timeoutMs)*time.Millisecond, 100*time.Millisecond, message)
}

// WaitForEnabled waits until the locator is visible and not disabled.
func (s *TestSession) WaitForEnabled(q queries.Query) {
	s.t.Logf("Waiting for %q to be enabled", q.Describe())
	loc := q.Run(s)
	s.WaitUntil(func() bool {
		visible, err := loc.IsVisible()
		if err != nil || !visible {
			return false
		}
		disabled, err := loc.IsDisabled()
		return err == nil && !disabled
	}, fmt.Sprintf("%s did not become enabled", q.Describe()))
}

// WaitForURL waits until the current page URL matches a Playwright glob.
func (s *TestSession) WaitForURL(glob string) {
	s.t.Logf("Waiting for URL %q", glob)
	if err := s.page.WaitForURL(glob, pw.PageWaitForURLOptions{Timeout: pw.Float(s.timeoutMs)}); err != nil {
		s.t.Fatalf("wait for URL %q: %v (last URL %q)", glob, err, s.page.URL())
	}
}

func (s *TestSession) resetDatabase() {
	// Guard against truncating a developer's database: the E2E bootstrap
	// points DB_NAME at superplane_test, but if that ever regresses we want
	// a loud failure here instead of silently wiping superplane_dev.
	if err := database.VerifyTestDatabase(database.Conn()); err != nil {
		s.t.Fatalf("reset database: %v", err)
	}

	sql := `DO $$
    DECLARE r RECORD;
    BEGIN
        FOR r IN (
            SELECT tablename
            FROM pg_tables
            WHERE schemaname = 'public'
              AND tablename NOT IN ('schema_migrations', 'usage_price_books', 'usage_price_book_rates')
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
	token, err := authentication.GenerateAccountToken(signer, s.Account.ID.String(), time.Now(), 24*time.Hour)
	if err != nil {
		s.t.Fatalf("jwt: %v", err)
	}

	if err := s.context.AddCookies([]pw.OptionalCookie{{
		Name:     "account_token",
		Value:    token,
		URL:      pw.String(s.BaseURL + "/"),
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

	// New orgs enable factories by default. Classic canvas and Apps home
	// tests must stay on /:org/apps/:id. Factory tests turn the flag on.
	if err := models.DisableExperimentalFeature(organization.ID, features.FeatureFactories); err != nil {
		s.t.Fatalf("disable factories: %v", err)
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
		tx := database.Conn().Begin()
		err = svc.SetupOrganization(tx, organization.ID.String(), user.ID.String())
		if err != nil {
			tx.Rollback()
			s.t.Fatalf("setup organization error: %v", err)
		}

		err = tx.Commit().Error
		if err != nil {
			s.t.Fatalf("commit transaction: %v", err)
		}
	}

	s.OrgID = organization.ID
	s.OrgSlug = organization.Slug
	s.Account = account
}

func (s *TestSession) Click(q queries.Query) {
	s.t.Logf("Clicking button %q", q.Describe())

	if err := q.Run(s).Click(pw.LocatorClickOptions{Timeout: pw.Float(s.timeoutMs)}); err != nil {
		s.t.Fatalf("click button %q: %v", q.Describe(), err)
	}
}

// ClickWithControlOrMeta clicks with the multi-selection modifier (Control on Windows/Linux, Meta on macOS).
func (s *TestSession) ClickWithControlOrMeta(q queries.Query) {
	s.t.Logf("Clicking with ControlOrMeta modifier %q", q.Describe())

	opts := pw.LocatorClickOptions{
		Timeout:   pw.Float(s.timeoutMs),
		Modifiers: []pw.KeyboardModifier{*pw.KeyboardModifierControlOrMeta},
	}
	if err := q.Run(s).Click(opts); err != nil {
		s.t.Fatalf("click with ControlOrMeta %q: %v", q.Describe(), err)
	}
}

func (s *TestSession) FillIn(q queries.Query, value string) {
	s.t.Logf("Filling in %q with %q", q.Describe(), value)

	el := q.Run(s)
	if el == nil {
		s.t.Fatalf("fill in %q: query returned no element", q.Describe())
	}

	if err := el.Fill(value, pw.LocatorFillOptions{Timeout: pw.Float(s.timeoutMs)}); err != nil {
		s.t.Fatalf("fill in %q with %q: %v", q.Describe(), value, err)
	}
}

func (s *TestSession) VisitHomePage() {
	s.Visit("/" + s.OrgID.String() + "/")
}

func (s *TestSession) DragAndDrop(source queries.Query, target queries.Query, offsetX, offsetY int) {
	s.t.Logf("Dragging element %q to %q with offset (%d, %d)", source.Describe(), target.Describe(), offsetX, offsetY)

	srcEl := source.Run(s)
	tgtEl := target.Run(s)

	if err := srcEl.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(s.timeoutMs)}); err != nil {
		s.t.Fatalf("waiting for source %q to be visible: %v", source.Describe(), err)
	}

	if err := tgtEl.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(s.timeoutMs)}); err != nil {
		s.t.Fatalf("waiting for target %q to be visible: %v", target.Describe(), err)
	}

	srcBox, err := srcEl.BoundingBox()
	if err != nil || srcBox == nil {
		s.t.Fatalf("getting bounding box of source %q: %v", source.Describe(), err)
	}

	tgtBox, err := tgtEl.BoundingBox()
	if err != nil || tgtBox == nil {
		s.t.Fatalf("getting bounding box of target %q: %v", target.Describe(), err)
	}

	startX := srcBox.X + srcBox.Width/2
	startY := srcBox.Y + srcBox.Height/2
	endX := tgtBox.X + float64(offsetX)
	endY := tgtBox.Y + float64(offsetY)

	if err := s.page.Mouse().Move(startX, startY); err != nil {
		s.t.Fatalf("moving mouse to source %q: %v", source.Describe(), err)
	}
	if err := s.page.Mouse().Down(); err != nil {
		s.t.Fatalf("mouse down on source %q: %v", source.Describe(), err)
	}
	if err := s.page.Mouse().Move(endX, endY, pw.MouseMoveOptions{Steps: pw.Int(10)}); err != nil {
		s.t.Fatalf("moving mouse to target %q: %v", target.Describe(), err)
	}
	if err := s.page.Mouse().Up(); err != nil {
		s.t.Fatalf("mouse up on target %q: %v", target.Describe(), err)
	}
}

func (s *TestSession) AssertVisible(q queries.Query) {
	s.t.Logf("Asserting visibility of %q", q.Describe())

	if err := q.Run(s).WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible, Timeout: pw.Float(s.timeoutMs)}); err != nil {
		s.t.Fatalf("asserting visibility of %q: %v", q.Describe(), err)
	}
}

func (s *TestSession) AssertHidden(q queries.Query) {
	s.t.Logf("Asserting %q is hidden", q.Describe())

	if err := q.Run(s).WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateHidden, Timeout: pw.Float(s.timeoutMs)}); err != nil {
		s.t.Fatalf("asserting %q is hidden: %v", q.Describe(), err)
	}
}

func (s *TestSession) AssertDisabled(q queries.Query) {
	s.t.Logf("Asserting %q is disabled", q.Describe())
	loc := q.Run(s)
	s.WaitUntil(func() bool {
		disabled, err := loc.IsDisabled()
		return err == nil && disabled
	}, fmt.Sprintf("expected %q to be disabled", q.Describe()))
}

func (s *TestSession) PressKey(key string) {
	s.t.Logf("Pressing key %q", key)

	if err := s.page.Keyboard().Press(key); err != nil {
		s.t.Fatalf("pressing key %q: %v", key, err)
	}
}

// DragSelectOnCanvas performs a rubber-band selection on the React Flow canvas.
// It holds the platform multi-selection key (Ctrl/Meta) while dragging from
// (startX, startY) to (endX, endY) in viewport coordinates.
func (s *TestSession) DragSelectOnCanvas(target queries.Query, startX, startY, endX, endY int) {
	s.t.Logf("Drag-selecting on canvas from (%d,%d) to (%d,%d)", startX, startY, endX, endY)

	el := target.Run(s)
	box, err := el.BoundingBox()
	if err != nil || box == nil {
		s.t.Fatalf("getting bounding box of canvas: %v", err)
	}

	absStartX := box.X + float64(startX)
	absStartY := box.Y + float64(startY)
	absEndX := box.X + float64(endX)
	absEndY := box.Y + float64(endY)

	key := "Control"
	if isMac() {
		key = "Meta"
	}

	if err := s.page.Keyboard().Down(key); err != nil {
		s.t.Fatalf("pressing %s down: %v", key, err)
	}

	if err := s.page.Mouse().Move(absStartX, absStartY); err != nil {
		s.t.Fatalf("moving mouse to start: %v", err)
	}
	if err := s.page.Mouse().Down(); err != nil {
		s.t.Fatalf("mouse down: %v", err)
	}
	if err := s.page.Mouse().Move(absEndX, absEndY, pw.MouseMoveOptions{Steps: pw.Int(10)}); err != nil {
		s.t.Fatalf("moving mouse to end: %v", err)
	}
	if err := s.page.Mouse().Up(); err != nil {
		s.t.Fatalf("mouse up: %v", err)
	}

	if err := s.page.Keyboard().Up(key); err != nil {
		s.t.Fatalf("releasing %s: %v", key, err)
	}
}

func (s *TestSession) HoverOver(q queries.Query) {
	s.t.Logf("Hovering over %q", q.Describe())

	if err := q.Run(s).Hover(pw.LocatorHoverOptions{Timeout: pw.Float(s.timeoutMs)}); err != nil {
		s.t.Fatalf("hover over %q: %v", q.Describe(), err)
	}
}

func (s *TestSession) WaitUntilURLDoesNotContain(part string) {
	s.t.Logf("Waiting for URL to drop %q", part)
	s.WaitUntil(func() bool {
		return !strings.Contains(s.page.URL(), part)
	}, fmt.Sprintf("timed out waiting for URL to drop %q, last URL was %q", part, s.page.URL()))
}

// WaitUntilURLContains waits until the current URL contains part. Use this
// for redirects that a single-page app resolves after several async requests
// (for example, the post-login redirect into an organization).
func (s *TestSession) WaitUntilURLContains(part string) {
	s.WaitForURL("**" + part + "**")
}

func (s *TestSession) AssertURLContains(part string) {
	s.t.Logf("Asserting URL contains %q", part)
	current := s.page.URL()
	if !strings.Contains(current, part) {
		s.t.Fatalf("expected URL to contain %q, got %q", part, current)
	}
}

// WaitForBrowserPath polls until the URL path (ignoring query and fragment) equals expectedPath
// after normalizing trailing slashes. expectedPath is typically an app pathname such as
// "/<orgID>/apps/<appID>" (not including the origin or query string).
func (s *TestSession) WaitForBrowserPath(expectedPath string) {
	want := normalizeE2EBrowserPath(expectedPath)
	s.t.Logf("Waiting for browser path %q", want)
	last := s.page.URL()
	deadline := time.Now().Add(time.Duration(s.timeoutMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		last = s.page.URL()
		u, err := url.Parse(last)
		if err == nil && normalizeE2EBrowserPath(u.Path) == want {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.t.Fatalf("timed out waiting for browser path %q, last URL was %q", want, last)
}

// WaitForBrowserPathPrefix polls until the URL path starts with prefix after
// normalizing trailing slashes. Use this for factory routes that continue
// into a workspace, such as /{org}/workspaces/newwo/setup.
func (s *TestSession) WaitForBrowserPathPrefix(prefix string) {
	want := normalizeE2EBrowserPath(prefix)
	s.t.Logf("Waiting for browser path prefix %q", want)
	last := s.page.URL()
	deadline := time.Now().Add(time.Duration(s.timeoutMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		last = s.page.URL()
		u, err := url.Parse(last)
		if err == nil && strings.HasPrefix(normalizeE2EBrowserPath(u.Path), want) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.t.Fatalf("timed out waiting for browser path prefix %q, last URL was %q", want, last)
}

func normalizeE2EBrowserPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "/" {
		return "/"
	}
	return strings.TrimSuffix(p, "/")
}

func (s *TestSession) ScrollToTheBottomOfPage() {
	s.t.Log("Scrolling to the bottom of the page")

	script := `
		() => {
			try {
				// Scroll main window
				const doc = document.scrollingElement || document.documentElement || document.body;
				if (doc) {
					doc.scrollTo(0, doc.scrollHeight);
				}

				// Also scroll any large scrollable containers
				const candidates = Array.from(document.querySelectorAll('*'))
					.filter(el => {
						const style = window.getComputedStyle(el);
						return (style.overflowY === 'auto' || style.overflowY === 'scroll') && el.scrollHeight > el.clientHeight;
					});

				for (const el of candidates) {
					el.scrollTop = el.scrollHeight;
				}
			} catch (e) {
				console.error('scroll error', e);
			}
		}
	`

	if _, err := s.page.Evaluate(script, nil); err != nil {
		s.t.Fatalf("scrolling to the bottom of the page: %v", err)
	}

	s.WaitUntil(func() bool {
		result, err := s.page.Evaluate(`() => {
			const doc = document.scrollingElement || document.documentElement || document.body;
			if (!doc) {
				return false;
			}
			return doc.scrollTop + doc.clientHeight >= doc.scrollHeight - 2;
		}`, nil)
		reached, _ := result.(bool)
		return err == nil && reached
	}, "page did not finish scrolling to the bottom")
}

func isMac() bool {
	return runtime.GOOS == "darwin"
}
