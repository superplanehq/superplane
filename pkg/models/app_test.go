package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

// helpers

func createTestApp(t *testing.T, orgID uuid.UUID, slug string) *models.App {
	t.Helper()
	now := time.Now()
	userID := uuid.New()
	app := &models.App{
		ID:             uuid.New(),
		OrganizationID: orgID,
		DisplayName:    "Test App " + slug,
		Slug:           slug,
		Description:    "a test app",
		DefaultBranch:  "main",
		SyncStatus:     models.AppSyncStatusOk,
		CreatedBy:      &userID,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, models.CreateApp(database.Conn(), app))
	return app
}

// ─── CreateApp ───────────────────────────────────────────────────────────────

func Test__App__CreateApp(t *testing.T) {
	r := support.Setup(t)

	t.Run("creates app with all fields populated", func(t *testing.T) {
		now := time.Now()
		userID := r.User
		slug := "testorg-myapp"
		app := &models.App{
			ID:             uuid.New(),
			OrganizationID: r.Organization.ID,
			DisplayName:    "My App",
			Slug:           slug,
			Description:    "desc",
			DefaultBranch:  "main",
			SyncStatus:     models.AppSyncStatusOk,
			CreatedBy:      &userID,
			CreatedAt:      &now,
			UpdatedAt:      &now,
		}

		err := models.CreateApp(database.Conn(), app)
		require.NoError(t, err)
		assert.False(t, app.ID == uuid.Nil)

		// Verify it round-trips from DB
		found, err := models.FindApp(r.Organization.ID, app.ID)
		require.NoError(t, err)
		assert.Equal(t, app.ID, found.ID)
		assert.Equal(t, "My App", found.DisplayName)
		assert.Equal(t, slug, found.Slug)
		assert.Equal(t, "desc", found.Description)
		assert.Equal(t, "main", found.DefaultBranch)
		assert.Equal(t, models.AppSyncStatusOk, found.SyncStatus)
		assert.Equal(t, r.User, *found.CreatedBy)
	})

	t.Run("slug uniqueness constraint prevents duplicate slugs", func(t *testing.T) {
		slug := "testorg-dupslug"
		app1 := createTestApp(t, r.Organization.ID, slug)
		require.NotNil(t, app1)

		now := time.Now()
		app2 := &models.App{
			ID:             uuid.New(),
			OrganizationID: r.Organization.ID,
			DisplayName:    "Dup App",
			Slug:           slug,
			DefaultBranch:  "main",
			SyncStatus:     models.AppSyncStatusOk,
			CreatedAt:      &now,
			UpdatedAt:      &now,
		}
		err := models.CreateApp(database.Conn(), app2)
		assert.Error(t, err, "expected duplicate slug error")
	})
}

// ─── FindApp ─────────────────────────────────────────────────────────────────

func Test__App__FindApp(t *testing.T) {
	r := support.Setup(t)

	t.Run("returns app when found", func(t *testing.T) {
		app := createTestApp(t, r.Organization.ID, "testorg-findme")

		found, err := models.FindApp(r.Organization.ID, app.ID)
		require.NoError(t, err)
		assert.Equal(t, app.ID, found.ID)
	})

	t.Run("returns error when app not found", func(t *testing.T) {
		_, err := models.FindApp(r.Organization.ID, uuid.New())
		assert.Error(t, err)
	})

	t.Run("does not return app from different organization", func(t *testing.T) {
		otherOrg := support.CreateOrganization(t, r, r.User)
		app := createTestApp(t, r.Organization.ID, "testorg-wrongorg")

		_, err := models.FindApp(otherOrg.ID, app.ID)
		assert.Error(t, err)
	})

	t.Run("does not return soft-deleted app", func(t *testing.T) {
		app := createTestApp(t, r.Organization.ID, "testorg-softdel")
		require.NoError(t, app.SoftDelete())

		_, err := models.FindApp(r.Organization.ID, app.ID)
		assert.Error(t, err)
	})
}

// ─── FindAppInTransaction ────────────────────────────────────────────────────

func Test__App__FindAppInTransaction(t *testing.T) {
	r := support.Setup(t)

	t.Run("finds app within a transaction", func(t *testing.T) {
		app := createTestApp(t, r.Organization.ID, "testorg-txfind")

		var found *models.App
		err := database.Conn().Transaction(func(tx *gorm.DB) error {
			var txErr error
			found, txErr = models.FindAppInTransaction(tx, r.Organization.ID, app.ID)
			return txErr
		})
		require.NoError(t, err)
		assert.Equal(t, app.ID, found.ID)
	})
}

// ─── FindAppBySlug ────────────────────────────────────────────────────────────

func Test__App__FindAppBySlug(t *testing.T) {
	r := support.Setup(t)

	t.Run("returns app by slug", func(t *testing.T) {
		slug := "testorg-byslug"
		app := createTestApp(t, r.Organization.ID, slug)

		found, err := models.FindAppBySlug(r.Organization.ID, slug)
		require.NoError(t, err)
		assert.Equal(t, app.ID, found.ID)
	})

	t.Run("returns error when slug not found", func(t *testing.T) {
		_, err := models.FindAppBySlug(r.Organization.ID, "testorg-doesnotexist")
		assert.Error(t, err)
	})

	t.Run("returns error for slug in different organization", func(t *testing.T) {
		slug := "testorg-slugdifforg"
		createTestApp(t, r.Organization.ID, slug)
		otherOrg := support.CreateOrganization(t, r, r.User)

		_, err := models.FindAppBySlug(otherOrg.ID, slug)
		assert.Error(t, err)
	})
}

// ─── ListApps ─────────────────────────────────────────────────────────────────

func Test__App__ListApps(t *testing.T) {
	r := support.Setup(t)

	t.Run("returns empty list when no apps exist", func(t *testing.T) {
		apps, err := models.ListApps(r.Organization.ID.String())
		require.NoError(t, err)
		assert.Empty(t, apps)
	})

	t.Run("returns all apps for organization", func(t *testing.T) {
		createTestApp(t, r.Organization.ID, "testorg-list1")
		createTestApp(t, r.Organization.ID, "testorg-list2")
		createTestApp(t, r.Organization.ID, "testorg-list3")

		apps, err := models.ListApps(r.Organization.ID.String())
		require.NoError(t, err)
		assert.Len(t, apps, 3)
	})

	t.Run("does not return apps from other organizations", func(t *testing.T) {
		otherOrg := support.CreateOrganization(t, r, r.User)
		createTestApp(t, otherOrg.ID, "otherorg-app")

		apps, err := models.ListApps(r.Organization.ID.String())
		require.NoError(t, err)
		// still only the 3 from previous sub-test (TruncateTables was not called here)
		for _, a := range apps {
			assert.Equal(t, r.Organization.ID, a.OrganizationID)
		}
	})

	t.Run("does not return soft-deleted apps", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		app1 := createTestApp(t, r.Organization.ID, "testorg-nodel")
		app2 := createTestApp(t, r.Organization.ID, "testorg-deleted")
		require.NoError(t, app2.SoftDelete())

		apps, err := models.ListApps(r.Organization.ID.String())
		require.NoError(t, err)
		require.Len(t, apps, 1)
		assert.Equal(t, app1.ID, apps[0].ID)
	})

	t.Run("orders apps by created_at ascending", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		a := createTestApp(t, r.Organization.ID, "testorg-order-a")
		b := createTestApp(t, r.Organization.ID, "testorg-order-b")

		apps, err := models.ListApps(r.Organization.ID.String())
		require.NoError(t, err)
		require.Len(t, apps, 2)
		assert.Equal(t, a.ID, apps[0].ID)
		assert.Equal(t, b.ID, apps[1].ID)
	})
}

// ─── UpdateApp ────────────────────────────────────────────────────────────────

func Test__App__UpdateApp(t *testing.T) {
	r := support.Setup(t)

	t.Run("persists changes to app", func(t *testing.T) {
		app := createTestApp(t, r.Organization.ID, "testorg-update")

		app.DisplayName = "Updated Name"
		app.Description = "Updated desc"
		app.SyncStatus = models.AppSyncStatusFailed
		errMsg := "some error"
		app.SyncError = &errMsg

		err := models.UpdateApp(database.Conn(), app)
		require.NoError(t, err)

		found, err := models.FindApp(r.Organization.ID, app.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", found.DisplayName)
		assert.Equal(t, "Updated desc", found.Description)
		assert.Equal(t, models.AppSyncStatusFailed, found.SyncStatus)
		require.NotNil(t, found.SyncError)
		assert.Equal(t, "some error", *found.SyncError)
	})

	t.Run("sets updated_at timestamp", func(t *testing.T) {
		app := createTestApp(t, r.Organization.ID, "testorg-updatedts")
		originalUpdated := app.UpdatedAt

		// Ensure at least 1ms passes
		time.Sleep(2 * time.Millisecond)

		app.DisplayName = "Changed"
		require.NoError(t, models.UpdateApp(database.Conn(), app))

		found, err := models.FindApp(r.Organization.ID, app.ID)
		require.NoError(t, err)
		require.NotNil(t, found.UpdatedAt)
		assert.True(t, found.UpdatedAt.After(*originalUpdated))
	})
}

// ─── SoftDelete ───────────────────────────────────────────────────────────────

func Test__App__SoftDelete(t *testing.T) {
	r := support.Setup(t)

	t.Run("soft-deleted app is not returned by FindApp", func(t *testing.T) {
		app := createTestApp(t, r.Organization.ID, "testorg-softdelete")
		require.NoError(t, app.SoftDelete())

		_, err := models.FindApp(r.Organization.ID, app.ID)
		assert.Error(t, err)
	})

	t.Run("soft-deleted app is not returned by ListApps", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		app := createTestApp(t, r.Organization.ID, "testorg-listdel")
		require.NoError(t, app.SoftDelete())

		apps, err := models.ListApps(r.Organization.ID.String())
		require.NoError(t, err)
		assert.Empty(t, apps)
	})
}

// ─── IsAppSlugTaken ───────────────────────────────────────────────────────────

func Test__App__IsAppSlugTaken(t *testing.T) {
	r := support.Setup(t)

	t.Run("returns false when slug is not taken", func(t *testing.T) {
		taken, err := models.IsAppSlugTaken("totally-unique-slug-xyz")
		require.NoError(t, err)
		assert.False(t, taken)
	})

	t.Run("returns true when slug exists", func(t *testing.T) {
		slug := "testorg-takenslug"
		createTestApp(t, r.Organization.ID, slug)

		taken, err := models.IsAppSlugTaken(slug)
		require.NoError(t, err)
		assert.True(t, taken)
	})

	t.Run("soft-deleted apps still hold the slug", func(t *testing.T) {
		slug := "testorg-delslug"
		app := createTestApp(t, r.Organization.ID, slug)
		require.NoError(t, app.SoftDelete())

		taken, err := models.IsAppSlugTaken(slug)
		require.NoError(t, err)
		// GORM soft-delete keeps the row, so the slug is still considered taken
		assert.True(t, taken)
	})
}

// ─── AppDoc CRUD ──────────────────────────────────────────────────────────────

func Test__AppDoc__UpsertAndFind(t *testing.T) {
	r := support.Setup(t)

	t.Run("creates a new doc", func(t *testing.T) {
		app := createTestApp(t, r.Organization.ID, "testorg-doc-create")
		now := time.Now()
		doc := &models.AppDoc{
			ID:        uuid.New(),
			AppID:     app.ID,
			Path:      "docs/readme.md",
			Content:   "# Hello",
			Sha:       "abc123",
			UpdatedAt: &now,
		}

		saved, err := models.UpsertAppDoc(database.Conn(), doc)
		require.NoError(t, err)
		assert.Equal(t, doc.AppID, saved.AppID)
		assert.Equal(t, "docs/readme.md", saved.Path)
		assert.Equal(t, "# Hello", saved.Content)
		assert.Equal(t, "abc123", saved.Sha)
	})

	t.Run("updates existing doc on conflict", func(t *testing.T) {
		app := createTestApp(t, r.Organization.ID, "testorg-doc-update")
		now := time.Now()
		doc := &models.AppDoc{
			ID:        uuid.New(),
			AppID:     app.ID,
			Path:      "docs/guide.md",
			Content:   "original",
			Sha:       "sha1",
			UpdatedAt: &now,
		}
		_, err := models.UpsertAppDoc(database.Conn(), doc)
		require.NoError(t, err)

		// Upsert again with new content
		updated := &models.AppDoc{
			ID:        uuid.New(), // new ID, same (app_id, path) conflict key
			AppID:     app.ID,
			Path:      "docs/guide.md",
			Content:   "updated content",
			Sha:       "sha2",
			UpdatedAt: &now,
		}
		saved, err := models.UpsertAppDoc(database.Conn(), updated)
		require.NoError(t, err)
		assert.Equal(t, "updated content", saved.Content)
		assert.Equal(t, "sha2", saved.Sha)

		// Confirm only one doc at that path
		all, err := models.FindAppDocsByAppID(app.ID)
		require.NoError(t, err)
		assert.Len(t, all, 1)
	})

	t.Run("FindAppDocByPath returns doc", func(t *testing.T) {
		app := createTestApp(t, r.Organization.ID, "testorg-doc-bypath")
		now := time.Now()
		doc := &models.AppDoc{
			ID:        uuid.New(),
			AppID:     app.ID,
			Path:      "docs/ops.md",
			Content:   "ops runbook",
			UpdatedAt: &now,
		}
		_, err := models.UpsertAppDoc(database.Conn(), doc)
		require.NoError(t, err)

		found, err := models.FindAppDocByPath(app.ID, "docs/ops.md")
		require.NoError(t, err)
		assert.Equal(t, "ops runbook", found.Content)
	})

	t.Run("FindAppDocByPath returns error when not found", func(t *testing.T) {
		app := createTestApp(t, r.Organization.ID, "testorg-doc-notfound")
		_, err := models.FindAppDocByPath(app.ID, "docs/missing.md")
		assert.Error(t, err)
	})
}

func Test__AppDoc__FindAppDocsByAppID(t *testing.T) {
	r := support.Setup(t)

	t.Run("returns empty list when no docs exist", func(t *testing.T) {
		app := createTestApp(t, r.Organization.ID, "testorg-docs-empty")
		docs, err := models.FindAppDocsByAppID(app.ID)
		require.NoError(t, err)
		assert.Empty(t, docs)
	})

	t.Run("returns all docs for app ordered by path", func(t *testing.T) {
		app := createTestApp(t, r.Organization.ID, "testorg-docs-list")
		now := time.Now()
		for _, path := range []string{"docs/z.md", "docs/a.md", "docs/m.md"} {
			p := path
			_, err := models.UpsertAppDoc(database.Conn(), &models.AppDoc{
				ID:        uuid.New(),
				AppID:     app.ID,
				Path:      p,
				Content:   "content for " + p,
				UpdatedAt: &now,
			})
			require.NoError(t, err)
		}

		docs, err := models.FindAppDocsByAppID(app.ID)
		require.NoError(t, err)
		require.Len(t, docs, 3)
		assert.Equal(t, "docs/a.md", docs[0].Path)
		assert.Equal(t, "docs/m.md", docs[1].Path)
		assert.Equal(t, "docs/z.md", docs[2].Path)
	})

	t.Run("only returns docs for the given app", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		app1 := createTestApp(t, r.Organization.ID, "testorg-docs-app1")
		app2 := createTestApp(t, r.Organization.ID, "testorg-docs-app2")
		now := time.Now()

		_, err := models.UpsertAppDoc(database.Conn(), &models.AppDoc{
			ID:        uuid.New(),
			AppID:     app1.ID,
			Path:      "docs/app1.md",
			UpdatedAt: &now,
		})
		require.NoError(t, err)

		_, err = models.UpsertAppDoc(database.Conn(), &models.AppDoc{
			ID:        uuid.New(),
			AppID:     app2.ID,
			Path:      "docs/app2.md",
			UpdatedAt: &now,
		})
		require.NoError(t, err)

		docs, err := models.FindAppDocsByAppID(app1.ID)
		require.NoError(t, err)
		require.Len(t, docs, 1)
		assert.Equal(t, "docs/app1.md", docs[0].Path)
	})
}
