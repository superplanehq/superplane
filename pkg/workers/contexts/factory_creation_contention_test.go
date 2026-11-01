package contexts

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	factorycomp "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/pkg/integrations/productive"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type pausedFileUpload struct {
	blob.Provider
	entered chan<- struct{}
	release <-chan struct{}
	once    sync.Once
}

func (p *pausedFileUpload) Put(ctx context.Context, key string, body io.Reader, options blob.PutOptions) error {
	p.once.Do(func() {
		close(p.entered)
		<-p.release
	})
	return p.Provider.Put(ctx, key, body, options)
}

func TestFactoryContext_CreateWorkOrderRemoteFetchDoesNotLockCounter(t *testing.T) {
	for _, phase := range []string{"remote_read", "remote_upload", "workspace_copy"} {
		t.Run(phase, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			store, err := filesystem.New(t.TempDir())
			require.NoError(t, err)
			blob.SetCurrent(store)
			defer blob.SetCurrent(nil)
			db := database.Conn()
			f, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			canvas, execution, _ := setupFactoryAppExecution(t, r, f.ID)
			otherCanvas, otherExecution, _ := setupFactoryAppExecution(t, r, f.ID)
			entered, release := make(chan struct{}), make(chan struct{})
			description := "![image](https://user-images.githubusercontent.com/1/image.png)"
			if phase == "workspace_copy" {
				file, err := models.CreatePendingFile(db, models.CreateFileParams{
					Scope: blob.ScopeWorkspace, OrganizationID: r.Organization.ID, FactoryID: f.ID,
					Filename: "workspace.png", ContentType: "image/png", CreatedByID: r.User,
				})
				require.NoError(t, err)
				require.NoError(t, storedfiles.CompleteUpload(t.Context(), db, store, file, bytes.NewReader([]byte("image"))))
				description = "![image](" + blob.FileRef(file.ID) + ")"
			}
			if phase != "remote_read" {
				blob.SetCurrent(&pausedFileUpload{Provider: store, entered: entered, release: release})
			}
			var releaseOnce sync.Once
			releaseFetch := func() { releaseOnce.Do(func() { close(release) }) }
			done := make(chan error, 1)
			finished := false
			var first *core.WorkOrder
			go func() {
				var cleanups []FileBindCleanup
				txErr := db.Transaction(func(tx *gorm.DB) error {
					ctx := NewFactoryContext(tx, canvas, execution).
						WithFileBindCleanup(func(job FileBindCleanup) { cleanups = append(cleanups, job) }).
						WithRemoteImageFetch(func(context.Context, *http.Request) (*http.Response, error) {
							if phase == "remote_read" {
								close(entered)
								<-release
							}
							return &http.Response{StatusCode: http.StatusOK,
								Header: http.Header{"Content-Type": []string{"image/png"}},
								Body:   io.NopCloser(bytes.NewReader([]byte("image")))}, nil
						})
					var err error
					first, _, err = ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Slow image", Description: description})
					return err
				})
				ApplyFileBindCleanups(cleanups, txErr)
				done <- txErr
			}()
			defer func() {
				releaseFetch()
				if !finished {
					require.NoError(t, <-done)
				}
			}()
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("creation did not reach remote processing")
			}
			// This independent transaction can acquire a connection and execute SQL.
			// On the original code, only the allocator UPDATE times out.
			var second *core.WorkOrder
			err = db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Exec("SET LOCAL statement_timeout = '500ms'").Error; err != nil {
					return err
				}
				var err error
				second, _, err = NewFactoryContext(tx, otherCanvas, otherExecution).CreateWorkOrder(core.WorkOrderParams{Title: "Independent creation"})
				return err
			})
			require.NoError(t, err, "remote processing must not hold the factory counter lock")
			secondOrder, err := f.FindWorkOrder(db, uuid.MustParse(second.ID))
			require.NoError(t, err)
			assert.Positive(t, secondOrder.Number)
			// Finish the first transaction before checking both persisted results.
			releaseFetch()
			firstErr := <-done
			finished = true
			require.NoError(t, firstErr)
			firstOrder, err := f.FindWorkOrder(db, uuid.MustParse(first.ID))
			require.NoError(t, err)
			assert.Positive(t, firstOrder.Number)
			assert.NotEqual(t, firstOrder.Number, secondOrder.Number)
			assert.Equal(t, &execution.RunID, firstOrder.SourceRunID)
			assert.Equal(t, first.Description, firstOrder.Description)
			files, err := models.ListReadyTaskFiles(db, firstOrder.ID)
			require.NoError(t, err)
			require.Len(t, files, 1)
			assert.Contains(t, firstOrder.Description, blob.FileRef(files[0].ID))
		})
	}
}

func TestFactoryContext_CreateWorkOrderRollbackCleansPreparedObjects(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	defer blob.SetCurrent(nil)
	db := database.Conn()
	f, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, execution, _ := setupFactoryAppExecution(t, r, f.ID)
	listener, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: "on-work-order", Type: models.NodeTypeTrigger,
		Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: factorycomp.OnWorkOrderTriggerName}}),
	}}, nil)
	require.NoError(t, db.Model(listener).Update("factory_id", f.ID).Error)
	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope: blob.ScopeWorkspace, OrganizationID: r.Organization.ID, FactoryID: f.ID,
		Filename: "workspace.png", ContentType: "image/png", CreatedByID: r.User,
	})
	require.NoError(t, err)
	require.NoError(t, storedfiles.CompleteUpload(t.Context(), db, store, file, bytes.NewReader([]byte("workspace image"))))
	sourceKey := file.StorageKey
	var cleanups []FileBindCleanup
	var created *core.WorkOrder
	forced := errors.New("force rollback after creation")
	txErr := db.Transaction(func(tx *gorm.DB) error {
		ctx := NewFactoryContext(tx, canvas, execution).
			WithFileBindCleanup(func(job FileBindCleanup) { cleanups = append(cleanups, job) }).
			WithRemoteImageFetch(func(context.Context, *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK,
					Header: http.Header{"Content-Type": []string{"image/png"}},
					Body:   io.NopCloser(bytes.NewReader([]byte("remote image")))}, nil
			})
		var err error
		created, _, err = ctx.CreateWorkOrder(core.WorkOrderParams{
			Title: "Rollback files", Description: "![workspace](" + blob.FileRef(file.ID) + ")\n![remote](https://user-images.githubusercontent.com/1/image.png)",
		})
		if err != nil {
			return err
		}
		order, err := f.FindWorkOrder(tx, uuid.MustParse(created.ID))
		if err != nil {
			return err
		}
		assert.Equal(t, int64(1), order.Number)
		events, err := models.ListCanvasEvents(tx, listener.ID, "on-work-order", 10, nil)
		require.NoError(t, err)
		require.Len(t, events, 1, "successful creation emits an event inside the transaction")
		return forced
	})
	require.ErrorIs(t, txErr, forced)
	require.Len(t, cleanups, 1)
	require.Len(t, cleanups[0].Result.CopiedKeys, 2)
	for _, key := range cleanups[0].Result.CopiedKeys {
		_, err := store.Head(t.Context(), key)
		require.NoError(t, err, "cleanup must wait for transaction completion")
	}
	ApplyFileBindCleanups(cleanups, txErr)
	for _, key := range cleanups[0].Result.CopiedKeys {
		_, err := store.Head(t.Context(), key)
		assert.ErrorIs(t, err, blob.ErrNotFound)
	}
	_, err = store.Head(t.Context(), sourceKey)
	require.NoError(t, err, "rollback preserves the workspace source")
	restored, err := models.FindFile(db, file.ID)
	require.NoError(t, err)
	assert.Equal(t, blob.ScopeWorkspace, restored.Scope)
	assert.Equal(t, sourceKey, restored.StorageKey)
	_, err = f.FindWorkOrder(db, uuid.MustParse(created.ID))
	assert.ErrorIs(t, err, models.ErrFactoryWorkOrderNotFound)
	files, err := models.ListReadyTaskFiles(db, uuid.MustParse(created.ID))
	require.NoError(t, err)
	assert.Empty(t, files)
	events, err := models.ListCanvasEvents(db, listener.ID, "on-work-order", 10, nil)
	require.NoError(t, err)
	assert.Empty(t, events, "rolled-back creation events must not become visible")
	fresh, err := models.FindFactory(db, f.OrganizationID, f.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), fresh.NextWorkOrderNumber)
	next, err := f.CreateWorkOrder(db, "Next task", "", nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), next.Number)
}

func TestFactoryContext_CreateWorkOrderFailureInsideCommittingCaller(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	defer blob.SetCurrent(nil)
	db := database.Conn()
	f, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, execution, _ := setupFactoryAppExecution(t, r, f.ID)
	forced := errors.New("fail prepared file persistence")
	const callback = "test:fail_prepared_file_insert"
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "files" {
			tx.AddError(forced)
		}
	}))
	defer db.Callback().Create().Remove(callback)
	var cleanups []FileBindCleanup
	err = db.Transaction(func(tx *gorm.DB) error {
		ctx := NewFactoryContext(tx, canvas, execution).
			WithFileBindCleanup(func(job FileBindCleanup) { cleanups = append(cleanups, job) }).
			WithRemoteImageFetch(func(context.Context, *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK,
					Header: http.Header{"Content-Type": []string{"image/png"}},
					Body:   io.NopCloser(bytes.NewReader([]byte("remote image")))}, nil
			})
		order, created, createErr := ctx.CreateWorkOrder(core.WorkOrderParams{
			Title: "Failed creation", Description: "![image](https://user-images.githubusercontent.com/1/image.png)",
		})
		require.ErrorIs(t, createErr, forced)
		assert.Nil(t, order)
		assert.False(t, created)
		// The caller records the action failure and commits, as the executor can.
		return nil
	})
	require.NoError(t, err)
	require.Len(t, cleanups, 1)
	require.ErrorIs(t, cleanups[0].BindErr, forced)
	require.Len(t, cleanups[0].Result.CopiedKeys, 1)
	ApplyFileBindCleanups(cleanups, nil)
	_, err = store.Head(t.Context(), cleanups[0].Result.CopiedKeys[0])
	assert.ErrorIs(t, err, blob.ErrNotFound)
	orders, err := f.ListWorkOrders(db, models.ListFactoryWorkOrdersFilters{})
	require.NoError(t, err)
	assert.Empty(t, orders)
	fresh, err := models.FindFactory(db, f.OrganizationID, f.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), fresh.NextWorkOrderNumber)
}

func TestFactoryContext_CreateWorkOrderRejectsClaimedWorkspaceFile(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	defer blob.SetCurrent(nil)
	db := database.Conn()
	f, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, execution, _ := setupFactoryAppExecution(t, r, f.ID)
	otherCanvas, otherExecution, _ := setupFactoryAppExecution(t, r, f.ID)
	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope: blob.ScopeWorkspace, OrganizationID: r.Organization.ID, FactoryID: f.ID,
		Filename: "workspace.png", ContentType: "image/png", CreatedByID: r.User,
	})
	require.NoError(t, err)
	require.NoError(t, storedfiles.CompleteUpload(t.Context(), db, store, file, bytes.NewBufferString("image")))
	description := "![image](" + blob.FileRef(file.ID) + ")"
	entered, release := make(chan struct{}), make(chan struct{})
	blob.SetCurrent(&pausedFileUpload{Provider: store, entered: entered, release: release})
	var releaseOnce sync.Once
	releaseUpload := func() { releaseOnce.Do(func() { close(release) }) }
	done := make(chan error, 1)
	var cleanups []FileBindCleanup
	go func() {
		done <- db.Transaction(func(tx *gorm.DB) error {
			_, _, err := NewFactoryContext(tx, canvas, execution).
				WithFileBindCleanup(func(job FileBindCleanup) { cleanups = append(cleanups, job) }).
				CreateWorkOrder(core.WorkOrderParams{Title: "Delayed claim", Description: description})
			return err
		})
	}()
	finished := false
	defer func() {
		releaseUpload()
		if !finished {
			<-done
		}
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("creation did not reach workspace copy")
	}
	// Both creations read the workspace owner before the first claim commits.
	blob.SetCurrent(store)
	var winner *core.WorkOrder
	err = db.Transaction(func(tx *gorm.DB) error {
		var err error
		winner, _, err = NewFactoryContext(tx, otherCanvas, otherExecution).
			CreateWorkOrder(core.WorkOrderParams{Title: "First claim", Description: description})
		return err
	})
	require.NoError(t, err)
	releaseUpload()
	loserErr := <-done
	finished = true
	require.ErrorIs(t, loserErr, models.ErrFileForeignReference)
	ApplyFileBindCleanups(cleanups, loserErr)
	loaded, err := models.FindFile(db, file.ID)
	require.NoError(t, err)
	assert.Equal(t, uuid.MustParse(winner.ID), *loaded.WorkOrderID)
	_, err = store.Head(t.Context(), loaded.StorageKey)
	require.NoError(t, err, "losing creation must preserve the winning attachment")
	require.Len(t, cleanups, 1)
	for _, key := range cleanups[0].Result.CopiedKeys {
		_, err = store.Head(t.Context(), key)
		assert.ErrorIs(t, err, blob.ErrNotFound)
	}
	orders, err := f.ListWorkOrders(db, models.ListFactoryWorkOrdersFilters{})
	require.NoError(t, err)
	require.Len(t, orders, 1)
	assert.Equal(t, winner.ID, orders[0].ID.String())
	fresh, err := models.FindFactory(db, f.OrganizationID, f.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), fresh.NextWorkOrderNumber)
}

func TestFactoryContext_CreateWorkOrderPreservesLinearLinkAttachments(t *testing.T) {
	for _, scenario := range []string{"links_only", "mixed_attachments", "file_quota"} {
		t.Run(scenario, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			store, err := filesystem.New(t.TempDir())
			require.NoError(t, err)
			blob.SetCurrent(store)
			defer blob.SetCurrent(nil)
			t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
			t.Setenv("BASE_URL", "http://files.test")
			db := database.Conn()
			f, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			canvas, execution, _ := setupFactoryAppExecutionWithPayload(t, r, f.ID, map[string]any{
				"type": linear.IssuePayloadType,
				"data": map[string]any{
					"url":  "https://linear.app/acme/issue/ENG-142/bug",
					"data": map[string]any{"id": uuid.NewString(), "identifier": "ENG-142"},
				},
			})
			listener, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
				NodeID: "on-work-order", Type: models.NodeTypeTrigger,
				Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: factorycomp.OnWorkOrderTriggerName}}),
			}}, nil)
			require.NoError(t, db.Model(listener).Update("factory_id", f.ID).Error)
			if scenario == "file_quota" {
				file, err := models.CreatePendingFile(db, models.CreateFileParams{
					Scope: blob.ScopeWorkspace, OrganizationID: r.Organization.ID, FactoryID: f.ID,
					Filename: "existing.png", ContentType: "image/png", CreatedByID: r.User,
				})
				require.NoError(t, err)
				// Represent a full storage quota without uploading ten gigabytes.
				require.NoError(t, db.Model(file).Updates(map[string]any{
					"state": models.FileStateReady, "size_bytes": models.MaxOrganizationFileBytes,
				}).Error)
			}
			inlineURL := "https://uploads.linear.app/image.png"
			description := "Investigate the issue."
			var issueFiles []linear.IssueFile
			if scenario != "links_only" {
				description += "\n![image](" + inlineURL + ")"
				issueFiles = []linear.IssueFile{
					{Name: "image.png", ContentType: "image/png", Body: []byte("image"), ReplaceURLs: []string{inlineURL}},
					{Name: "notes.pdf", ContentType: "application/pdf", Body: []byte("notes")},
				}
			}
			var cleanups []FileBindCleanup
			ctx := NewFactoryContext(db, canvas, execution).
				WithFileBindCleanup(func(job FileBindCleanup) { cleanups = append(cleanups, job) }).
				WithLinearIssueFiles(func(context.Context, string, string) ([]linear.IssueFile, []linear.IssueLink, error) {
					return issueFiles, []linear.IssueLink{{Title: "Design", URL: "https://example.com/design"}}, nil
				})
			created, inserted, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Linear attachments", Description: description})
			require.NoError(t, err)
			require.True(t, inserted)
			ApplyFileBindCleanups(cleanups, nil)
			order, err := f.FindWorkOrder(db, uuid.MustParse(created.ID))
			require.NoError(t, err)
			assert.Contains(t, order.Description, "- [Design](https://example.com/design)")
			assert.Equal(t, order.Description, created.Description)
			files, err := models.ListReadyTaskFiles(db, order.ID)
			require.NoError(t, err)
			if scenario == "mixed_attachments" {
				require.Len(t, files, 2)
				assert.NotContains(t, order.Description, inlineURL)
				for _, file := range files {
					assert.Contains(t, order.Description, blob.FileRef(file.ID))
				}
			} else {
				assert.Empty(t, files)
				assert.NotContains(t, order.Description, blob.FileRefScheme+"://")
				if scenario == "file_quota" {
					assert.Contains(t, order.Description, inlineURL)
					assert.NotContains(t, order.Description, "notes.pdf")
				}
			}
			events, err := models.ListCanvasEvents(db, listener.ID, "on-work-order", 10, nil)
			require.NoError(t, err)
			require.Len(t, events, 1)
			assert.Equal(t, order.Description, onWorkOrderEventWorkOrder(t, events[0])["description"])
		})
	}
}

func TestFactoryContext_CreateWorkOrderKeepsRejectedRemoteLinks(t *testing.T) {
	for _, reason := range []string{"organization_quota", "empty_file", "appended_attachment_quota"} {
		t.Run(reason, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			store, err := filesystem.New(t.TempDir())
			require.NoError(t, err)
			blob.SetCurrent(store)
			defer blob.SetCurrent(nil)
			db := database.Conn()
			t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
			t.Setenv("BASE_URL", "http://files.test")
			f, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			canvas, execution, _ := setupFactoryAppExecution(t, r, f.ID)
			listener, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
				NodeID: "on-work-order", Type: models.NodeTypeTrigger,
				Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: factorycomp.OnWorkOrderTriggerName}}),
			}}, nil)
			require.NoError(t, db.Model(listener).Update("factory_id", f.ID).Error)
			if reason != "empty_file" {
				file, err := models.CreatePendingFile(db, models.CreateFileParams{
					Scope: blob.ScopeWorkspace, OrganizationID: r.Organization.ID, FactoryID: f.ID,
					Filename: "existing.png", ContentType: "image/png", CreatedByID: r.User,
				})
				require.NoError(t, err)
				// Represent existing storage without uploading ten gigabytes.
				require.NoError(t, db.Model(file).Updates(map[string]any{
					"state": models.FileStateReady, "size_bytes": models.MaxOrganizationFileBytes - 4,
				}).Error)
			}
			rejectedURL := "https://user-images.githubusercontent.com/1/rejected.png"
			acceptedURL := "https://user-images.githubusercontent.com/1/accepted.png"
			description := "![rejected](" + rejectedURL + ")\n![accepted](" + acceptedURL + ")"
			if reason == "appended_attachment_quota" {
				canvas, execution, _ = setupFactoryAppExecutionWithPayload(t, r, f.ID, map[string]any{
					"type": productive.TaskPayloadType,
					"data": map[string]any{
						"meta": map[string]any{"event": productive.TaskCreatedEvent},
						"data": map[string]any{"id": "20305431", "type": "tasks"},
					},
				})
			}
			var cleanups []FileBindCleanup
			var created *core.WorkOrder
			err = db.Transaction(func(tx *gorm.DB) error {
				ctx := NewFactoryContext(tx, canvas, execution).
					WithFileBindCleanup(func(job FileBindCleanup) { cleanups = append(cleanups, job) }).
					WithRemoteImageFetch(func(_ context.Context, req *http.Request) (*http.Response, error) {
						body := "ok"
						if req.URL.String() == rejectedURL {
							body = "too large"
							if reason == "empty_file" {
								body = ""
							}
						}
						return &http.Response{StatusCode: http.StatusOK,
							Header: http.Header{"Content-Type": []string{"image/png"}},
							Body:   io.NopCloser(bytes.NewBufferString(body))}, nil
					})
				if reason == "appended_attachment_quota" {
					ctx.remoteImageFetch = nil
					ctx.WithProductiveTaskFiles(func(context.Context, string, string) ([]productive.TaskFile, error) {
						return []productive.TaskFile{
							{Name: "rejected.pdf", ContentType: "application/pdf", Body: []byte("too large")},
							{Name: "accepted.png", ContentType: "image/png", Body: []byte("ok"), ReplaceURLs: []string{acceptedURL}},
						}, nil
					})
				}
				var err error
				created, _, err = ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Optional images", Description: description})
				return err
			})
			require.NoError(t, err)
			ApplyFileBindCleanups(cleanups, nil)
			order, err := f.FindWorkOrder(db, uuid.MustParse(created.ID))
			require.NoError(t, err)
			files, err := models.ListReadyTaskFiles(db, order.ID)
			require.NoError(t, err)
			require.Len(t, files, 1)
			assert.Contains(t, order.Description, rejectedURL)
			assert.NotContains(t, order.Description, acceptedURL)
			assert.Contains(t, order.Description, blob.FileRef(files[0].ID))
			assert.Equal(t, order.Description, created.Description)
			assert.NotContains(t, order.Description, "rejected.pdf")
			events, err := models.ListCanvasEvents(db, listener.ID, "on-work-order", 10, nil)
			require.NoError(t, err)
			require.Len(t, events, 1)
			payload := onWorkOrderEventWorkOrder(t, events[0])
			assert.Equal(t, order.Description, payload["description"])
			listed, ok := payload["files"].([]any)
			require.True(t, ok)
			require.Len(t, listed, 1)
			_, err = store.Head(t.Context(), files[0].StorageKey)
			require.NoError(t, err)
			require.Len(t, cleanups, 1)
			require.NotEmpty(t, cleanups[0].Result.StaleKeys)
			for _, key := range cleanups[0].Result.StaleKeys {
				_, err = store.Head(t.Context(), key)
				assert.ErrorIs(t, err, blob.ErrNotFound, "rejected uploads must be discarded")
			}
		})
	}
}
