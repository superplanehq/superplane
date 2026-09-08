package storedfiles

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

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
	require.NoError(t, ApplyBindResult(t.Context(), db, provider, r.Organization.ID, factoryModel.ID, bound, nil))
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

	ingested, err := IngestRemoteImages(
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
	assert.Contains(t, ingested.Markdown, blob.FileRefScheme+"://")
	assert.NotContains(t, ingested.Markdown, okURL)
	assert.Contains(t, ingested.Markdown, failURL)
	require.Len(t, ingested.ObjectKeys, 1)

	files, err := models.ListReadyTaskFiles(db, order.ID)
	require.NoError(t, err)
	require.Len(t, files, 1)
}

func TestIngestRemoteImagesDeletesObjectsWhenTransactionRollsBack(t *testing.T) {
	r := support.Setup(t)
	provider := setupFileStore(t)
	db := database.Conn()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png-bytes"))
	}))
	t.Cleanup(server.Close)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Import", "", &r.User, nil, nil)
	require.NoError(t, err)

	okURL := server.URL + "/ok.png"
	var ingested IngestResult
	err = db.Transaction(func(tx *gorm.DB) error {
		result, ingestErr := IngestRemoteImages(
			t.Context(),
			tx,
			provider,
			func(ctx context.Context, req *http.Request) (*http.Response, error) {
				return http.DefaultClient.Do(req.WithContext(ctx))
			},
			func(string) bool { return true },
			r.Organization.ID,
			factoryModel.ID,
			order.ID,
			&r.User,
			"![ok]("+okURL+")",
		)
		if ingestErr != nil {
			return ingestErr
		}
		ingested = result
		return errors.New("force rollback")
	})
	require.Error(t, err)
	require.Len(t, ingested.ObjectKeys, 1)
	require.NoError(t, ApplyBindResult(t.Context(), db, provider, r.Organization.ID, factoryModel.ID, BindResult{CopiedKeys: ingested.ObjectKeys}, err))

	_, headErr := provider.Head(t.Context(), ingested.ObjectKeys[0])
	assert.ErrorIs(t, headErr, blob.ErrNotFound)
	files, err := models.ListReadyTaskFiles(db, order.ID)
	require.NoError(t, err)
	assert.Empty(t, files)
}

type failDeleteProvider struct {
	blob.Provider
}

func (p failDeleteProvider) Delete(ctx context.Context, key string) error {
	return errors.New("delete denied")
}

func TestApplyBindResultRecordsAbandonedObjectsWhenDeleteFails(t *testing.T) {
	r := support.Setup(t)
	provider := setupFileStore(t)
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	key := "abandoned/" + uuid.NewString()
	require.NoError(t, provider.Put(t.Context(), key, strings.NewReader("png-bytes"), blob.PutOptions{ContentType: "image/png"}))

	err = ApplyBindResult(
		t.Context(),
		db,
		failDeleteProvider{Provider: provider},
		r.Organization.ID,
		factoryModel.ID,
		BindResult{CopiedKeys: []string{key}},
		errors.New("force rollback"),
	)
	require.Error(t, err)

	stale, err := models.ListStalePendingFiles(db, time.Now(), 10)
	require.NoError(t, err)
	require.Len(t, stale, 1)
	assert.Equal(t, key, stale[0].StorageKey)
	assert.Equal(t, models.FileStateFailed, stale[0].State)

	require.NoError(t, DeleteObjectAndRow(t.Context(), db, provider, &stale[0]))
	_, headErr := provider.Head(t.Context(), key)
	assert.ErrorIs(t, headErr, blob.ErrNotFound)
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

func TestDescriptionForDispatchRewritesReadyFileRefs(t *testing.T) {
	r := support.Setup(t)
	provider := setupFileStore(t)
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Dispatch", "", &r.User, nil, nil)
	require.NoError(t, err)

	ready, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "ready.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, CompleteUpload(t.Context(), db, provider, ready, bytes.NewReader([]byte("png-bytes"))))

	pending, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "pending.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)

	otherFactory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	foreign, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      otherFactory.ID,
		Filename:       "foreign.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, CompleteUpload(t.Context(), db, provider, foreign, bytes.NewReader([]byte("png-bytes"))))

	markdown := fmt.Sprintf(
		"See ![ready](%s) ![pending](%s) ![foreign](%s)",
		blob.FileRef(ready.ID),
		blob.FileRef(pending.ID),
		blob.FileRef(foreign.ID),
	)
	rewritten, files, err := DescriptionForDispatch(
		t.Context(),
		db,
		provider,
		r.Organization.ID,
		factoryModel.ID,
		order.ID,
		markdown,
		time.Hour,
	)
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, ready.ID, files[0].ID)
	assert.Equal(t, "ready.png", files[0].Filename)
	assert.Contains(t, files[0].URL, "/api/v1/public/files/"+ready.ID.String())
	assert.Contains(t, rewritten, files[0].URL)
	assert.NotContains(t, rewritten, blob.FileRef(ready.ID))
	assert.Contains(t, rewritten, blob.FileRef(pending.ID))
	assert.Contains(t, rewritten, blob.FileRef(foreign.ID))
}

func TestDescriptionForDispatchSkipsOtherTaskFiles(t *testing.T) {
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
		Filename:       "other.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, CompleteUpload(t.Context(), db, provider, file, bytes.NewReader([]byte("png-bytes"))))

	markdown := "![other](" + blob.FileRef(file.ID) + ")"
	rewritten, files, err := DescriptionForDispatch(
		t.Context(),
		db,
		provider,
		r.Organization.ID,
		factoryModel.ID,
		second.ID,
		markdown,
		time.Hour,
	)
	require.NoError(t, err)
	assert.Empty(t, files)
	assert.Equal(t, markdown, rewritten)
}

func TestCloneDescriptionFilesCopiesReadyFilesAndRewritesMarkdown(t *testing.T) {
	r := support.Setup(t)
	provider := setupFileStore(t)
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	source, err := factoryModel.CreateWorkOrder(db, "Source", "", &r.User, nil, nil)
	require.NoError(t, err)
	copyTo, err := factoryModel.CreateWorkOrder(db, "Copy", "", &r.User, nil, nil)
	require.NoError(t, err)

	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    source.ID,
		Filename:       "bug.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, CompleteUpload(t.Context(), db, provider, file, bytes.NewReader([]byte("png-bytes"))))

	markdown := "See ![bug](" + blob.FileRef(file.ID) + ")"
	cloned, err := CloneDescriptionFiles(
		t.Context(),
		db,
		provider,
		r.Organization.ID,
		factoryModel.ID,
		copyTo.ID,
		r.User,
		markdown,
	)
	require.NoError(t, err)
	assert.NotContains(t, cloned.Markdown, blob.FileRef(file.ID))
	assert.Contains(t, cloned.Markdown, blob.FileRefScheme+"://")
	require.Len(t, cloned.CopiedKeys, 1)

	sourceFiles, err := models.ListReadyTaskFiles(db, source.ID)
	require.NoError(t, err)
	require.Len(t, sourceFiles, 1)
	assert.Equal(t, file.ID, sourceFiles[0].ID)
	assert.Equal(t, file.StorageKey, sourceFiles[0].StorageKey)

	copiedFiles, err := models.ListReadyTaskFiles(db, copyTo.ID)
	require.NoError(t, err)
	require.Len(t, copiedFiles, 1)
	assert.NotEqual(t, file.ID, copiedFiles[0].ID)
	assert.Equal(t, "bug.png", copiedFiles[0].Filename)
	assert.Equal(t, int64(9), copiedFiles[0].SizeBytes)
	assert.Contains(t, cloned.Markdown, blob.FileRef(copiedFiles[0].ID))
	_, err = provider.Head(t.Context(), copiedFiles[0].StorageKey)
	require.NoError(t, err)
	_, err = provider.Head(t.Context(), file.StorageKey)
	require.NoError(t, err)
}

func TestCloneDescriptionFilesLeavesEmptyMarkdown(t *testing.T) {
	cloned, err := CloneDescriptionFiles(
		t.Context(),
		nil,
		nil,
		uuid.Nil,
		uuid.Nil,
		uuid.Nil,
		uuid.Nil,
		"",
	)
	require.NoError(t, err)
	assert.Empty(t, cloned.Markdown)
	assert.Empty(t, cloned.CopiedKeys)
}

func TestCloneDescriptionFilesSkipsForeignAndUnreadFiles(t *testing.T) {
	r := support.Setup(t)
	provider := setupFileStore(t)
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Copy", "", &r.User, nil, nil)
	require.NoError(t, err)

	otherFactory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	foreign, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      otherFactory.ID,
		Filename:       "foreign.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, CompleteUpload(t.Context(), db, provider, foreign, bytes.NewReader([]byte("png-bytes"))))

	pending, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "pending.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)

	markdown := fmt.Sprintf("![foreign](%s) ![pending](%s)", blob.FileRef(foreign.ID), blob.FileRef(pending.ID))
	cloned, err := CloneDescriptionFiles(
		t.Context(),
		db,
		provider,
		r.Organization.ID,
		factoryModel.ID,
		order.ID,
		r.User,
		markdown,
	)
	require.NoError(t, err)
	assert.Equal(t, markdown, cloned.Markdown)
	assert.Empty(t, cloned.CopiedKeys)
	copiedFiles, err := models.ListReadyTaskFiles(db, order.ID)
	require.NoError(t, err)
	assert.Empty(t, copiedFiles)
}

func TestCloneDescriptionFilesDeletesCopiedObjectWhenTransactionRollsBack(t *testing.T) {
	r := support.Setup(t)
	provider := setupFileStore(t)
	db := database.Conn()

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	source, err := factoryModel.CreateWorkOrder(db, "Source", "", &r.User, nil, nil)
	require.NoError(t, err)
	copyTo, err := factoryModel.CreateWorkOrder(db, "Copy", "", &r.User, nil, nil)
	require.NoError(t, err)

	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    source.ID,
		Filename:       "bug.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, CompleteUpload(t.Context(), db, provider, file, bytes.NewReader([]byte("png-bytes"))))

	var cloned CloneResult
	err = db.Transaction(func(tx *gorm.DB) error {
		result, cloneErr := CloneDescriptionFiles(
			t.Context(),
			tx,
			provider,
			r.Organization.ID,
			factoryModel.ID,
			copyTo.ID,
			r.User,
			"![bug]("+blob.FileRef(file.ID)+")",
		)
		if cloneErr != nil {
			return cloneErr
		}
		cloned = result
		return errors.New("force rollback")
	})
	require.Error(t, err)
	require.Len(t, cloned.CopiedKeys, 1)
	require.NoError(t, ApplyBindResult(t.Context(), db, provider, r.Organization.ID, factoryModel.ID, BindResult{CopiedKeys: cloned.CopiedKeys}, err))

	_, headErr := provider.Head(t.Context(), cloned.CopiedKeys[0])
	assert.ErrorIs(t, headErr, blob.ErrNotFound)
	copiedFiles, err := models.ListReadyTaskFiles(db, copyTo.ID)
	require.NoError(t, err)
	assert.Empty(t, copiedFiles)
}

func TestDescriptionForDispatchLeavesPlainMarkdown(t *testing.T) {
	markdown := "No files here"
	rewritten, files, err := DescriptionForDispatch(
		t.Context(),
		nil,
		nil,
		uuid.Nil,
		uuid.Nil,
		uuid.Nil,
		markdown,
		time.Hour,
	)
	require.NoError(t, err)
	assert.Equal(t, markdown, rewritten)
	assert.Empty(t, files)
}
