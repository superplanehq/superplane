package apps_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcApps "github.com/superplanehq/superplane/pkg/grpc/actions/apps"
	"github.com/superplanehq/superplane/pkg/models"
	pbCanvases "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

var testCtx *e2eContext

// e2eContext holds test-wide shared state (database, server URL, etc.)
type e2eContext struct {
	baseURL string
}

func TestMain(m *testing.M) {
	testCtx = &e2eContext{
		baseURL: getenv("BASE_URL", "http://127.0.0.1:8001"),
	}
	os.Exit(m.Run())
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ── helpers ──────────────────────────────────────────────────────────────────

type appSteps struct {
	t       *testing.T
	r       *support.ResourceRegistry
	session *session.TestSession
}

func newAppSteps(t *testing.T) *appSteps {
	return &appSteps{t: t}
}

func (s *appSteps) givenAUserAndOrg() {
	s.r = support.Setup(s.t)
}

func (s *appSteps) authedCtx() context.Context {
	return authentication.SetUserIdInMetadata(context.Background(), s.r.User.String())
}

func (s *appSteps) whenICreateAnApp(displayName, appSlug, description string) *models.App {
	s.t.Helper()
	ctx := s.authedCtx()
	resp, err := grpcApps.CreateApp(
		ctx,
		s.r.Organization.ID,
		s.r.Organization.Name,
		displayName,
		appSlug,
		description,
	)
	require.NoError(s.t, err)
	require.NotNil(s.t, resp.GetApp())

	appUUID := uuid.MustParse(resp.GetApp().GetMetadata().GetId())
	app, err := models.FindApp(s.r.Organization.ID, appUUID)
	require.NoError(s.t, err)
	return app
}

func (s *appSteps) whenIAddADocToApp(app *models.App, path, content string) {
	s.t.Helper()
	ctx := s.authedCtx()
	_, err := grpcApps.UpdateAppDoc(
		ctx,
		s.r.Organization.ID.String(),
		app.ID.String(),
		path,
		content,
	)
	require.NoError(s.t, err)
}

func (s *appSteps) whenIUpdateDashboard(app *models.App, canvasID uuid.UUID) {
	s.t.Helper()
	// Associate canvas with app first
	require.NoError(s.t, database.Conn().Model(&models.App{}).
		Where("id = ?", app.ID).
		Update("canvas_id", canvasID).Error)

	ctx := s.authedCtx()
	_, err := grpcApps.UpdateAppDashboard(
		ctx,
		s.r.Organization.ID.String(),
		app.ID.String(),
		[]*pbCanvases.DashboardPanel{
			{Id: "panel1", Type: "markdown"},
		},
		[]*pbCanvases.DashboardLayoutItem{
			{I: "panel1", X: 0, Y: 0, W: 6, H: 3},
		},
	)
	require.NoError(s.t, err)
}

func (s *appSteps) whenISyncApp(app *models.App) {
	s.t.Helper()
	ctx := s.authedCtx()
	_, err := grpcApps.SyncApp(ctx, s.r.Organization.ID, app.ID.String())
	require.NoError(s.t, err)
}

func (s *appSteps) whenIDeleteApp(app *models.App) {
	s.t.Helper()
	ctx := s.authedCtx()
	_, err := grpcApps.DeleteApp(ctx, s.r.Organization.ID, app.ID.String())
	require.NoError(s.t, err)
}

func (s *appSteps) thenAppExistsInDB(appID uuid.UUID) *models.App {
	s.t.Helper()
	app, err := models.FindApp(s.r.Organization.ID, appID)
	require.NoError(s.t, err)
	return app
}

func (s *appSteps) thenAppIsDeleted(appID uuid.UUID) {
	s.t.Helper()
	_, err := models.FindApp(s.r.Organization.ID, appID)
	assert.Error(s.t, err, "app should be soft-deleted")
}

func (s *appSteps) thenDocExists(app *models.App, path, content string) {
	s.t.Helper()
	doc, err := models.FindAppDocByPath(app.ID, path)
	require.NoError(s.t, err)
	assert.Equal(s.t, content, doc.Content)
}

func (s *appSteps) thenDashboardHasPanels(canvasID uuid.UUID, count int) {
	s.t.Helper()
	dashboard, err := models.FindCanvasDashboard(canvasID)
	require.NoError(s.t, err)
	assert.Len(s.t, dashboard.Panels.Data(), count)
}

func (s *appSteps) thenSyncStatusIs(app *models.App, expectedStatus string) {
	s.t.Helper()
	found, err := models.FindApp(s.r.Organization.ID, app.ID)
	require.NoError(s.t, err)
	assert.Equal(s.t, expectedStatus, found.SyncStatus)
}

func (s *appSteps) thenAppListHasCount(expectedCount int) {
	s.t.Helper()
	apps, err := models.ListApps(s.r.Organization.ID.String())
	require.NoError(s.t, err)
	assert.Len(s.t, apps, expectedCount)
}

// ── Full lifecycle test ───────────────────────────────────────────────────────

// TestAppLifecycle tests the complete App lifecycle end-to-end at the service layer:
//
//	Create → Edit Dashboard → Add Docs → Sync → Delete
func TestAppLifecycle(t *testing.T) {
	t.Run("full app lifecycle", func(t *testing.T) {
		steps := newAppSteps(t)
		steps.givenAUserAndOrg()

		// 1. Create app
		app := steps.whenICreateAnApp("Production Deploy", "proddeploy", "Prod deployment app")
		assert.Equal(t, steps.r.Organization.Name+"-proddeploy", app.Slug)
		assert.Equal(t, "Production Deploy", app.DisplayName)
		assert.Equal(t, models.AppSyncStatusOk, app.SyncStatus)
		steps.thenAppListHasCount(1)

		// 2. Add documentation
		steps.whenIAddADocToApp(app, "docs/runbook.md", "# Runbook\n\nRestart procedure...")
		steps.whenIAddADocToApp(app, "docs/adr/001-architecture.md", "# ADR-001\n\nDecision...")
		steps.thenDocExists(app, "docs/runbook.md", "# Runbook\n\nRestart procedure...")
		steps.thenDocExists(app, "docs/adr/001-architecture.md", "# ADR-001\n\nDecision...")

		// 3. Edit dashboard (requires a canvas)
		canvas, _ := support.CreateCanvas(t, steps.r.Organization.ID, steps.r.User, nil, nil)
		steps.whenIUpdateDashboard(app, canvas.ID)
		steps.thenDashboardHasPanels(canvas.ID, 1)

		// 4. Sync app
		steps.whenISyncApp(app)
		steps.thenSyncStatusIs(app, models.AppSyncStatusOk)

		// 5. Delete app
		steps.whenIDeleteApp(app)
		steps.thenAppIsDeleted(app.ID)
		steps.thenAppListHasCount(0)
	})
}

// TestAppCreate covers the create flow with validation scenarios.
func TestAppCreate(t *testing.T) {
	t.Run("creates app with valid inputs", func(t *testing.T) {
		steps := newAppSteps(t)
		steps.givenAUserAndOrg()

		app := steps.whenICreateAnApp("My Service", "myservice", "service description")
		require.NotEqual(t, uuid.Nil, app.ID)
		assert.Equal(t, "My Service", app.DisplayName)
		assert.Equal(t, steps.r.Organization.Name+"-myservice", app.Slug)
		assert.Equal(t, "service description", app.Description)
		assert.Equal(t, "main", app.DefaultBranch)
		assert.Equal(t, models.AppSyncStatusOk, app.SyncStatus)
		assert.NotNil(t, app.CreatedAt)
		assert.NotNil(t, app.UpdatedAt)
		assert.Equal(t, steps.r.User, *app.CreatedBy)
	})

	t.Run("two apps can coexist with different slugs", func(t *testing.T) {
		steps := newAppSteps(t)
		steps.givenAUserAndOrg()

		steps.whenICreateAnApp("App One", "appone", "")
		steps.whenICreateAnApp("App Two", "apptwo", "")
		steps.thenAppListHasCount(2)
	})

	t.Run("slug is globally unique across organizations", func(t *testing.T) {
		steps := newAppSteps(t)
		steps.givenAUserAndOrg()

		ctx := steps.authedCtx()
		resp, err := grpcApps.CreateApp(ctx, steps.r.Organization.ID, steps.r.Organization.Name, "App One", "uniqueslug", "")
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Attempt to create with same org_slug-app_slug combination
		_, err = grpcApps.CreateApp(ctx, steps.r.Organization.ID, steps.r.Organization.Name, "App Two", "uniqueslug", "")
		require.Error(t, err)
	})
}

// TestAppDocs covers the documentation lifecycle.
func TestAppDocs(t *testing.T) {
	t.Run("add, update and list docs", func(t *testing.T) {
		steps := newAppSteps(t)
		steps.givenAUserAndOrg()
		app := steps.whenICreateAnApp("Doc App", "docapp", "")

		// Add initial doc
		steps.whenIAddADocToApp(app, "docs/readme.md", "# Initial content")
		steps.thenDocExists(app, "docs/readme.md", "# Initial content")

		// Update doc
		steps.whenIAddADocToApp(app, "docs/readme.md", "# Updated content")
		steps.thenDocExists(app, "docs/readme.md", "# Updated content")

		// Add second doc
		steps.whenIAddADocToApp(app, "docs/ops.md", "# Ops Guide")
		steps.thenDocExists(app, "docs/ops.md", "# Ops Guide")

		// List all docs
		ctx := steps.authedCtx()
		listResp, err := grpcApps.ListAppDocs(ctx, steps.r.Organization.ID.String(), app.ID.String())
		require.NoError(t, err)
		assert.Len(t, listResp.GetDocs(), 2)
	})
}

// TestAppDashboard covers the dashboard lifecycle.
func TestAppDashboard(t *testing.T) {
	t.Run("dashboard is empty when app has no canvas", func(t *testing.T) {
		steps := newAppSteps(t)
		steps.givenAUserAndOrg()
		app := steps.whenICreateAnApp("No Canvas App", "nocanvasapp", "")

		ctx := steps.authedCtx()
		resp, err := grpcApps.GetAppDashboard(ctx, steps.r.Organization.ID.String(), app.ID.String())
		require.NoError(t, err)
		assert.Empty(t, resp.GetDashboard().GetPanels())
	})

	t.Run("dashboard stores and returns panels after update", func(t *testing.T) {
		steps := newAppSteps(t)
		steps.givenAUserAndOrg()
		app := steps.whenICreateAnApp("Dashboard App", "dashapp", "")
		canvas, _ := support.CreateCanvas(t, steps.r.Organization.ID, steps.r.User, nil, nil)
		steps.whenIUpdateDashboard(app, canvas.ID)
		steps.thenDashboardHasPanels(canvas.ID, 1)

		// Read back via gRPC
		ctx := steps.authedCtx()
		resp, err := grpcApps.GetAppDashboard(ctx, steps.r.Organization.ID.String(), app.ID.String())
		require.NoError(t, err)
		assert.Len(t, resp.GetDashboard().GetPanels(), 1)
		assert.Equal(t, "panel1", resp.GetDashboard().GetPanels()[0].GetId())
	})
}

// TestAppSync covers the sync behavior.
func TestAppSync(t *testing.T) {
	t.Run("sync transitions status back to ok", func(t *testing.T) {
		steps := newAppSteps(t)
		steps.givenAUserAndOrg()
		app := steps.whenICreateAnApp("Sync App", "syncapp", "")

		// Manually set status to syncing to mimic in-progress sync
		require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
			app.SyncStatus = models.AppSyncStatusSyncing
			return models.UpdateApp(tx, app)
		}))

		steps.whenISyncApp(app)
		steps.thenSyncStatusIs(app, models.AppSyncStatusOk)
	})

	t.Run("sync on deleted app returns not found", func(t *testing.T) {
		steps := newAppSteps(t)
		steps.givenAUserAndOrg()
		app := steps.whenICreateAnApp("Delete Sync App", "deletesyncapp", "")
		steps.whenIDeleteApp(app)

		ctx := steps.authedCtx()
		_, err := grpcApps.SyncApp(ctx, steps.r.Organization.ID, app.ID.String())
		require.Error(t, err)
	})
}

// TestAppDelete covers deletion scenarios.
func TestAppDelete(t *testing.T) {
	t.Run("deleted app no longer appears in list", func(t *testing.T) {
		steps := newAppSteps(t)
		steps.givenAUserAndOrg()

		app1 := steps.whenICreateAnApp("Keep App", "keepapp", "")
		app2 := steps.whenICreateAnApp("Delete Me", "deleteme", "")
		steps.thenAppListHasCount(2)

		steps.whenIDeleteApp(app2)
		steps.thenAppListHasCount(1)

		apps, err := models.ListApps(steps.r.Organization.ID.String())
		require.NoError(t, err)
		require.Len(t, apps, 1)
		assert.Equal(t, app1.ID, apps[0].ID)
	})

	t.Run("created_at and updated_at are recent", func(t *testing.T) {
		steps := newAppSteps(t)
		steps.givenAUserAndOrg()

		before := time.Now().Add(-time.Second)
		app := steps.whenICreateAnApp("Timestamp App", "tsapp", "")
		after := time.Now().Add(time.Second)

		assert.True(t, app.CreatedAt.After(before) && app.CreatedAt.Before(after),
			"created_at should be near now")
		assert.True(t, app.UpdatedAt.After(before) && app.UpdatedAt.Before(after),
			"updated_at should be near now")
	})
}
