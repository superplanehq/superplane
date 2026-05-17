package apps

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pbCanvases "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func authedCtx(userID uuid.UUID) context.Context {
	return authentication.SetUserIdInMetadata(context.Background(), userID.String())
}

func createApp(t *testing.T, r *support.ResourceRegistry, slug string) *models.App {
	t.Helper()
	now := time.Now()
	app := &models.App{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		DisplayName:    "App " + slug,
		Slug:           r.Organization.Name + "-" + slug,
		Description:    "",
		DefaultBranch:  "main",
		SyncStatus:     models.AppSyncStatusOk,
		CreatedBy:      &r.User,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return models.CreateApp(tx, app)
	}))
	return app
}

// ── CreateApp ─────────────────────────────────────────────────────────────────

func Test__CreateApp(t *testing.T) {
	r := support.Setup(t)
	orgID := r.Organization.ID
	orgSlug := r.Organization.Name

	t.Run("unauthenticated -> error", func(t *testing.T) {
		_, err := CreateApp(context.Background(), orgID, orgSlug, "My App", "slug1", "")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
	})

	t.Run("empty display_name -> error", func(t *testing.T) {
		ctx := authedCtx(r.User)
		_, err := CreateApp(ctx, orgID, orgSlug, "   ", "slug2", "")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("empty app_slug -> error", func(t *testing.T) {
		ctx := authedCtx(r.User)
		_, err := CreateApp(ctx, orgID, orgSlug, "My App", "", "")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid app_slug characters -> error", func(t *testing.T) {
		ctx := authedCtx(r.User)
		_, err := CreateApp(ctx, orgID, orgSlug, "My App", "Bad-Slug!", "")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("creates app and returns it", func(t *testing.T) {
		ctx := authedCtx(r.User)
		resp, err := CreateApp(ctx, orgID, orgSlug, "My App", "myapp", "some description")
		require.NoError(t, err)
		require.NotNil(t, resp.GetApp())
		assert.Equal(t, "My App", resp.GetApp().GetMetadata().GetDisplayName())
		assert.Equal(t, orgSlug+"-myapp", resp.GetApp().GetMetadata().GetSlug())
		assert.Equal(t, "some description", resp.GetApp().GetMetadata().GetDescription())
		assert.Equal(t, orgID.String(), resp.GetApp().GetMetadata().GetOrganizationId())
		assert.NotEmpty(t, resp.GetApp().GetMetadata().GetId())
		assert.Equal(t, models.AppSyncStatusOk, resp.GetApp().GetSyncState().GetStatus())
	})

	t.Run("duplicate slug -> AlreadyExists error", func(t *testing.T) {
		ctx := authedCtx(r.User)
		_, err := CreateApp(ctx, orgID, orgSlug, "Dup App A", "dupslug", "")
		require.NoError(t, err)

		_, err = CreateApp(ctx, orgID, orgSlug, "Dup App B", "dupslug", "")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, s.Code())
	})
}

// ── DescribeApp ───────────────────────────────────────────────────────────────

func Test__DescribeApp(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	t.Run("invalid organization_id -> error", func(t *testing.T) {
		_, err := DescribeApp(ctx, "not-a-uuid", uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid app_id -> error", func(t *testing.T) {
		_, err := DescribeApp(ctx, r.Organization.ID.String(), "bad-id")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("app not found -> error", func(t *testing.T) {
		_, err := DescribeApp(ctx, r.Organization.ID.String(), uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("returns app when found", func(t *testing.T) {
		app := createApp(t, r, "describe")
		resp, err := DescribeApp(ctx, r.Organization.ID.String(), app.ID.String())
		require.NoError(t, err)
		assert.Equal(t, app.ID.String(), resp.GetApp().GetMetadata().GetId())
		assert.Equal(t, app.DisplayName, resp.GetApp().GetMetadata().GetDisplayName())
		assert.Equal(t, app.Slug, resp.GetApp().GetMetadata().GetSlug())
	})
}

// ── ListApps ──────────────────────────────────────────────────────────────────

func Test__ListApps(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	t.Run("returns empty list when no apps exist", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		resp, err := ListApps(ctx, r.Organization.ID.String())
		require.NoError(t, err)
		assert.Empty(t, resp.GetApps())
	})

	t.Run("returns all apps for organization", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		createApp(t, r, "list-1")
		createApp(t, r, "list-2")

		resp, err := ListApps(ctx, r.Organization.ID.String())
		require.NoError(t, err)
		assert.Len(t, resp.GetApps(), 2)
	})

	t.Run("serializes app metadata correctly", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		app := createApp(t, r, "list-meta")

		resp, err := ListApps(ctx, r.Organization.ID.String())
		require.NoError(t, err)
		require.Len(t, resp.GetApps(), 1)
		assert.Equal(t, app.ID.String(), resp.GetApps()[0].GetMetadata().GetId())
		assert.Equal(t, app.DisplayName, resp.GetApps()[0].GetMetadata().GetDisplayName())
	})
}

// ── DeleteApp ─────────────────────────────────────────────────────────────────

func Test__DeleteApp(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	t.Run("invalid app_id -> error", func(t *testing.T) {
		_, err := DeleteApp(ctx, r.Organization.ID, "not-a-uuid")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("app not found -> error", func(t *testing.T) {
		_, err := DeleteApp(ctx, r.Organization.ID, uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("soft-deletes the app", func(t *testing.T) {
		app := createApp(t, r, "to-delete")
		_, err := DeleteApp(ctx, r.Organization.ID, app.ID.String())
		require.NoError(t, err)

		_, err = models.FindApp(r.Organization.ID, app.ID)
		assert.Error(t, err, "app should be soft-deleted and not findable")
	})

	t.Run("cannot delete app from different org", func(t *testing.T) {
		otherOrg := support.CreateOrganization(t, r, r.User)
		app := createApp(t, r, "cross-org-del")

		_, err := DeleteApp(ctx, otherOrg.ID, app.ID.String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})
}

// ── SyncApp ───────────────────────────────────────────────────────────────────

func Test__SyncApp(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	t.Run("invalid app_id -> error", func(t *testing.T) {
		_, err := SyncApp(ctx, r.Organization.ID, "bad-id")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("app not found -> error", func(t *testing.T) {
		_, err := SyncApp(ctx, r.Organization.ID, uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("sync stub sets status to ok", func(t *testing.T) {
		app := createApp(t, r, "sync-ok")
		resp, err := SyncApp(ctx, r.Organization.ID, app.ID.String())
		require.NoError(t, err)
		assert.Equal(t, models.AppSyncStatusOk, resp.GetApp().GetSyncState().GetStatus())
	})
}

// ── GetAppDashboard ───────────────────────────────────────────────────────────

func Test__GetAppDashboard(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	t.Run("invalid organization_id -> error", func(t *testing.T) {
		_, err := GetAppDashboard(ctx, "bad", uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid app_id -> error", func(t *testing.T) {
		_, err := GetAppDashboard(ctx, r.Organization.ID.String(), "bad")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("app not found -> error", func(t *testing.T) {
		_, err := GetAppDashboard(ctx, r.Organization.ID.String(), uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("returns empty dashboard when app has no canvas", func(t *testing.T) {
		app := createApp(t, r, "dash-nocanvas")
		resp, err := GetAppDashboard(ctx, r.Organization.ID.String(), app.ID.String())
		require.NoError(t, err)
		d := resp.GetDashboard()
		require.NotNil(t, d)
		assert.Empty(t, d.GetPanels())
		assert.Empty(t, d.GetLayout())
	})

	t.Run("returns stored dashboard when canvas exists", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		app := createApp(t, r, "dash-withcanvas")

		// Associate canvas with app
		canvasID := canvas.ID
		require.NoError(t, database.Conn().Model(&models.App{}).
			Where("id = ?", app.ID).
			Update("canvas_id", canvasID).Error)

		// Upsert dashboard panels
		_, err := models.UpsertCanvasDashboard(canvas.ID, []models.DashboardPanel{
			{ID: "p1", Type: "markdown", Content: map[string]any{"body": "hello"}},
		}, []models.DashboardLayoutItem{
			{I: "p1", X: 0, Y: 0, W: 4, H: 2},
		})
		require.NoError(t, err)

		resp, err := GetAppDashboard(ctx, r.Organization.ID.String(), app.ID.String())
		require.NoError(t, err)
		d := resp.GetDashboard()
		require.Len(t, d.GetPanels(), 1)
		assert.Equal(t, "p1", d.GetPanels()[0].GetId())
		assert.Equal(t, "markdown", d.GetPanels()[0].GetType())
	})
}

// ── UpdateAppDashboard ────────────────────────────────────────────────────────

func Test__UpdateAppDashboard(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	t.Run("invalid organization_id -> error", func(t *testing.T) {
		_, err := UpdateAppDashboard(ctx, "bad", uuid.New().String(), nil, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid app_id -> error", func(t *testing.T) {
		_, err := UpdateAppDashboard(ctx, orgID, "bad", nil, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("app not found -> error", func(t *testing.T) {
		_, err := UpdateAppDashboard(ctx, orgID, uuid.New().String(), nil, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("app with no canvas -> FailedPrecondition error", func(t *testing.T) {
		app := createApp(t, r, "updash-nocanvas")
		_, err := UpdateAppDashboard(ctx, orgID, app.ID.String(), []*pbCanvases.DashboardPanel{
			{Id: "p1", Type: "markdown"},
		}, []*pbCanvases.DashboardLayoutItem{{I: "p1", X: 0, Y: 0, W: 1, H: 1}})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, s.Code())
	})

	t.Run("updates dashboard on app with canvas", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		app := createApp(t, r, "updash-canvas")
		canvasID := canvas.ID
		require.NoError(t, database.Conn().Model(&models.App{}).
			Where("id = ?", app.ID).
			Update("canvas_id", canvasID).Error)

		minW := int32(2)
		resp, err := UpdateAppDashboard(ctx, orgID, app.ID.String(),
			[]*pbCanvases.DashboardPanel{
				{Id: "panel1", Type: "markdown"},
			},
			[]*pbCanvases.DashboardLayoutItem{
				{I: "panel1", X: 0, Y: 0, W: 6, H: 3, MinW: &minW},
			},
		)
		require.NoError(t, err)
		d := resp.GetDashboard()
		require.Len(t, d.GetPanels(), 1)
		assert.Equal(t, "panel1", d.GetPanels()[0].GetId())
		require.Len(t, d.GetLayout(), 1)
		assert.EqualValues(t, 6, d.GetLayout()[0].GetW())
		require.NotNil(t, d.GetLayout()[0].MinW)
		assert.EqualValues(t, 2, *d.GetLayout()[0].MinW)
	})
}

// ── GetAppDoc ─────────────────────────────────────────────────────────────────

func Test__GetAppDoc(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	t.Run("invalid organization_id -> error", func(t *testing.T) {
		_, err := GetAppDoc(ctx, "bad", uuid.New().String(), "docs/foo.md")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid app_id -> error", func(t *testing.T) {
		_, err := GetAppDoc(ctx, orgID, "bad", "docs/foo.md")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("empty path -> error", func(t *testing.T) {
		app := createApp(t, r, "getdoc-empty-path")
		_, err := GetAppDoc(ctx, orgID, app.ID.String(), "")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("app not found -> error", func(t *testing.T) {
		_, err := GetAppDoc(ctx, orgID, uuid.New().String(), "docs/foo.md")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("doc not found -> error", func(t *testing.T) {
		app := createApp(t, r, "getdoc-nodoc")
		_, err := GetAppDoc(ctx, orgID, app.ID.String(), "docs/missing.md")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("returns doc when found", func(t *testing.T) {
		app := createApp(t, r, "getdoc-found")
		now := time.Now()
		_, err := models.UpsertAppDoc(database.Conn(), &models.AppDoc{
			ID:        uuid.New(),
			AppID:     app.ID,
			Path:      "docs/readme.md",
			Content:   "# Hello World",
			Sha:       "deadbeef",
			UpdatedAt: &now,
		})
		require.NoError(t, err)

		resp, err := GetAppDoc(ctx, orgID, app.ID.String(), "docs/readme.md")
		require.NoError(t, err)
		assert.Equal(t, "docs/readme.md", resp.GetDoc().GetPath())
		assert.Equal(t, "# Hello World", resp.GetDoc().GetContent())
		assert.Equal(t, "deadbeef", resp.GetDoc().GetSha())
		assert.Equal(t, app.ID.String(), resp.GetDoc().GetAppId())
	})
}

// ── ListAppDocs ───────────────────────────────────────────────────────────────

func Test__ListAppDocs(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	t.Run("invalid organization_id -> error", func(t *testing.T) {
		_, err := ListAppDocs(ctx, "bad", uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid app_id -> error", func(t *testing.T) {
		_, err := ListAppDocs(ctx, orgID, "bad")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("app not found -> error", func(t *testing.T) {
		_, err := ListAppDocs(ctx, orgID, uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("returns empty list when no docs", func(t *testing.T) {
		app := createApp(t, r, "listdocs-empty")
		resp, err := ListAppDocs(ctx, orgID, app.ID.String())
		require.NoError(t, err)
		assert.Empty(t, resp.GetDocs())
	})

	t.Run("returns all docs for app", func(t *testing.T) {
		app := createApp(t, r, "listdocs-multi")
		now := time.Now()
		for _, path := range []string{"docs/a.md", "docs/b.md"} {
			p := path
			_, err := models.UpsertAppDoc(database.Conn(), &models.AppDoc{
				ID:        uuid.New(),
				AppID:     app.ID,
				Path:      p,
				Content:   "content of " + p,
				UpdatedAt: &now,
			})
			require.NoError(t, err)
		}

		resp, err := ListAppDocs(ctx, orgID, app.ID.String())
		require.NoError(t, err)
		assert.Len(t, resp.GetDocs(), 2)
	})
}

// ── UpdateAppDoc ──────────────────────────────────────────────────────────────

func Test__UpdateAppDoc(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	orgID := r.Organization.ID.String()

	t.Run("invalid organization_id -> error", func(t *testing.T) {
		_, err := UpdateAppDoc(ctx, "bad", uuid.New().String(), "docs/x.md", "content")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid app_id -> error", func(t *testing.T) {
		_, err := UpdateAppDoc(ctx, orgID, "bad", "docs/x.md", "content")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("empty path -> error", func(t *testing.T) {
		app := createApp(t, r, "updoc-emptypath")
		_, err := UpdateAppDoc(ctx, orgID, app.ID.String(), "", "content")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("app not found -> error", func(t *testing.T) {
		_, err := UpdateAppDoc(ctx, orgID, uuid.New().String(), "docs/x.md", "content")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("creates new doc when not present", func(t *testing.T) {
		app := createApp(t, r, "updoc-create")
		resp, err := UpdateAppDoc(ctx, orgID, app.ID.String(), "docs/new.md", "# New Doc")
		require.NoError(t, err)
		assert.Equal(t, "docs/new.md", resp.GetDoc().GetPath())
		assert.Equal(t, "# New Doc", resp.GetDoc().GetContent())
		assert.Equal(t, app.ID.String(), resp.GetDoc().GetAppId())
	})

	t.Run("updates existing doc content", func(t *testing.T) {
		app := createApp(t, r, "updoc-update")

		_, err := UpdateAppDoc(ctx, orgID, app.ID.String(), "docs/existing.md", "original")
		require.NoError(t, err)

		resp, err := UpdateAppDoc(ctx, orgID, app.ID.String(), "docs/existing.md", "updated content")
		require.NoError(t, err)
		assert.Equal(t, "updated content", resp.GetDoc().GetContent())

		// Only one doc at this path
		docs, err := models.FindAppDocsByAppID(app.ID)
		require.NoError(t, err)
		assert.Len(t, docs, 1)
	})

	t.Run("sets updated_at timestamp", func(t *testing.T) {
		app := createApp(t, r, "updoc-ts")
		resp, err := UpdateAppDoc(ctx, orgID, app.ID.String(), "docs/ts.md", "content")
		require.NoError(t, err)
		assert.NotNil(t, resp.GetDoc().GetUpdatedAt())
	})
}
