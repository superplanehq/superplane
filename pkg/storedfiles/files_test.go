package storedfiles

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func setupFileStore(t *testing.T) blob.Provider {
	t.Helper()
	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })
	return store
}

func TestCompleteUploadAndBindDescriptionFiles(t *testing.T) {
	r := support.Setup(t)
	provider := setupFileStore(t)
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Attach", "", &r.User, nil, nil)
	require.NoError(t, err)

	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "bug.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, CompleteUpload(t.Context(), db, provider, file, bytes.NewReader([]byte("png-bytes"))))

	loaded, err := models.FindFile(db, file.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FileStateReady, loaded.State)
	assert.Equal(t, int64(9), loaded.SizeBytes)

	description := "See ![bug](" + blob.FileRef(file.ID) + ")"
	sourceKey := loaded.StorageKey
	bound, err := BindDescriptionFiles(t.Context(), db, provider, r.Organization.ID, factoryModel.ID, order.ID, description)
	require.NoError(t, err)
	require.Equal(t, []string{sourceKey}, bound.StaleKeys)
	require.Len(t, bound.CopiedKeys, 1)
	_, err = provider.Head(t.Context(), sourceKey)
	require.NoError(t, err)
	require.NoError(t, ApplyBindResult(t.Context(), provider, bound, nil))
	_, err = provider.Head(t.Context(), sourceKey)
	assert.ErrorIs(t, err, blob.ErrNotFound)

	reparented, err := models.FindFile(db, file.ID)
	require.NoError(t, err)
	assert.Equal(t, blob.ScopeTask, reparented.Scope)
	assert.Equal(t, order.ID, *reparented.WorkOrderID)
	assert.Contains(t, reparented.StorageKey, "/tasks/"+order.ID.String()+"/")
	_, err = provider.Head(t.Context(), reparented.StorageKey)
	require.NoError(t, err)
}

func TestBindDescriptionFilesRejectsForeignWorkOrder(t *testing.T) {
	r := support.Setup(t)
	provider := setupFileStore(t)
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	first, err := factoryModel.CreateWorkOrder(db, "First", "", &r.User, nil, nil)
	require.NoError(t, err)
	second, err := factoryModel.CreateWorkOrder(db, "Second", "", &r.User, nil, nil)
	require.NoError(t, err)

	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    first.ID,
		Filename:       "bug.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, CompleteUpload(t.Context(), db, provider, file, bytes.NewReader([]byte("png-bytes"))))

	_, err = BindDescriptionFiles(
		t.Context(),
		db,
		provider,
		r.Organization.ID,
		factoryModel.ID,
		second.ID,
		"![bug]("+blob.FileRef(file.ID)+")",
	)
	assert.ErrorIs(t, err, models.ErrFileForeignReference)
}

func TestBindDescriptionFilesDeletesCopiedObjectWhenLaterFileFails(t *testing.T) {
	r := support.Setup(t)
	provider := setupFileStore(t)
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Attach", "", &r.User, nil, nil)
	require.NoError(t, err)

	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "bug.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, CompleteUpload(t.Context(), db, provider, file, bytes.NewReader([]byte("png-bytes"))))

	sourceKey := file.StorageKey
	nextKey, err := blob.ObjectKey(file.InstallationID, blob.ScopeTask, r.Organization.ID, factoryModel.ID, order.ID, file.ID)
	require.NoError(t, err)

	description := "![ok](" + blob.FileRef(file.ID) + ") ![missing](" + blob.FileRef(uuid.New()) + ")"
	err = db.Transaction(func(tx *gorm.DB) error {
		_, bindErr := BindDescriptionFiles(t.Context(), tx, provider, r.Organization.ID, factoryModel.ID, order.ID, description)
		return bindErr
	})
	assert.ErrorIs(t, err, models.ErrFileNotFound)

	loaded, err := models.FindFile(db, file.ID)
	require.NoError(t, err)
	assert.Equal(t, blob.ScopeWorkspace, loaded.Scope)
	assert.Equal(t, sourceKey, loaded.StorageKey)
	_, err = provider.Head(t.Context(), sourceKey)
	require.NoError(t, err)
	_, err = provider.Head(t.Context(), nextKey)
	assert.ErrorIs(t, err, blob.ErrNotFound)
}

func TestBindDescriptionFilesSerializesConcurrentTaskQuota(t *testing.T) {
	r := support.Setup(t)
	provider := setupFileStore(t)
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Quota", "", &r.User, nil, nil)
	require.NoError(t, err)

	for i := 0; i < models.MaxFilesPerWorkOrder-1; i++ {
		file, createErr := models.CreatePendingFile(db, models.CreateFileParams{
			Scope:          blob.ScopeTask,
			OrganizationID: r.Organization.ID,
			FactoryID:      factoryModel.ID,
			WorkOrderID:    order.ID,
			Filename:       "shot.png",
			ContentType:    "image/png",
			CreatedByID:    r.User,
		})
		require.NoError(t, createErr)
		require.NoError(t, CompleteUpload(t.Context(), db, provider, file, bytes.NewReader([]byte("png-bytes"))))
	}

	workspaceFiles := make([]*models.File, 2)
	for i := range workspaceFiles {
		file, createErr := models.CreatePendingFile(db, models.CreateFileParams{
			Scope:          blob.ScopeWorkspace,
			OrganizationID: r.Organization.ID,
			FactoryID:      factoryModel.ID,
			Filename:       "extra.png",
			ContentType:    "image/png",
			CreatedByID:    r.User,
		})
		require.NoError(t, createErr)
		require.NoError(t, CompleteUpload(t.Context(), db, provider, file, bytes.NewReader([]byte("png-bytes"))))
		workspaceFiles[i] = file
	}

	errCh := make(chan error, 2)
	var start sync.WaitGroup
	start.Add(2)
	for _, file := range workspaceFiles {
		go func(file *models.File) {
			start.Done()
			start.Wait()
			errCh <- db.Transaction(func(tx *gorm.DB) error {
				_, bindErr := BindDescriptionFiles(
					t.Context(),
					tx,
					provider,
					r.Organization.ID,
					factoryModel.ID,
					order.ID,
					"![extra]("+blob.FileRef(file.ID)+")",
				)
				return bindErr
			})
		}(file)
	}

	firstErr := <-errCh
	secondErr := <-errCh
	quotaFailures := 0
	for _, bindErr := range []error{firstErr, secondErr} {
		if bindErr == nil {
			continue
		}
		assert.ErrorIs(t, bindErr, models.ErrFileQuotaExceeded)
		quotaFailures++
	}
	assert.Equal(t, 1, quotaFailures)

	count, err := models.CountOpenTaskFiles(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(models.MaxFilesPerWorkOrder), count)
}

func TestIngestRemoteImagesRewritesGitHubURL(t *testing.T) {
	r := support.Setup(t)
	provider := setupFileStore(t)
	db := database.Conn()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/fail.png" {
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png-bytes"))
	}))
	t.Cleanup(server.Close)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Import", "", &r.User, nil, nil)
	require.NoError(t, err)

	okURL := server.URL + "/ok.png"
	failURL := server.URL + "/fail.png"
	markdown := "![ok](" + okURL + ") ![fail](" + failURL + ")"

	next, err := IngestRemoteImages(
		t.Context(),
		db,
		provider,
		func(ctx context.Context, req *http.Request) (*http.Response, error) {
			return http.DefaultClient.Do(req.WithContext(ctx))
		},
		func(string) bool { return true },
		r.Organization.ID,
		factoryModel.ID,
		order.ID,
		&r.User,
		markdown,
	)
	require.NoError(t, err)
	assert.Contains(t, next, blob.FileRefScheme+"://")
	assert.NotContains(t, next, okURL)
	assert.Contains(t, next, failURL)

	files, err := models.ListReadyTaskFiles(db, order.ID)
	require.NoError(t, err)
	require.Len(t, files, 1)
}

func TestCompleteUploadRejectsOversizedBody(t *testing.T) {
	r := support.Setup(t)
	provider := setupFileStore(t)
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "big.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)

	err = CompleteUpload(t.Context(), db, provider, file, io.LimitReader(strings.NewReader(strings.Repeat("a", models.MaxFileBytes+8)), int64(models.MaxFileBytes+8)))
	assert.ErrorIs(t, err, models.ErrFileQuotaExceeded)

	loaded, err := models.FindFile(db, file.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FileStateFailed, loaded.State)
}

func TestCompleteUploadDeletesObjectWhenReadyQuotaExceeded(t *testing.T) {
	r := support.Setup(t)
	provider := setupFileStore(t)
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	filler, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "filler.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)

	pending, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "next.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)

	require.NoError(t, CompleteUpload(t.Context(), db, provider, filler, bytes.NewReader([]byte("png-bytes"))))
	require.NoError(t, db.Model(filler).Update("size_bytes", models.MaxOrganizationFileBytes).Error)

	err = CompleteUpload(t.Context(), db, provider, pending, bytes.NewReader([]byte("png-bytes")))
	assert.ErrorIs(t, err, models.ErrFileQuotaExceeded)

	loaded, err := models.FindFile(db, pending.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FileStateFailed, loaded.State)

	_, err = provider.Head(t.Context(), pending.StorageKey)
	assert.ErrorIs(t, err, blob.ErrNotFound)
}
