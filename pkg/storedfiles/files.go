package storedfiles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	maxIngestBytes     = models.MaxFileBytes
	ingestFetchTimeout = 30 * time.Second
)

func BindDescriptionFiles(
	ctx context.Context,
	tx *gorm.DB,
	provider blob.Provider,
	organizationID, factoryID, workOrderID uuid.UUID,
	markdown string,
) error {
	ids := blob.FileIDsInMarkdown(markdown)
	if len(ids) == 0 {
		return nil
	}

	files, err := models.ListFilesByIDs(tx, ids)
	if err != nil {
		return err
	}
	byID := map[uuid.UUID]models.File{}
	for _, file := range files {
		byID[file.ID] = file
	}

	openCount, err := models.CountOpenTaskFiles(tx, workOrderID)
	if err != nil {
		return err
	}

	for _, id := range ids {
		file, ok := byID[id]
		if !ok {
			return fmt.Errorf("%w: %s", models.ErrFileNotFound, id)
		}
		if err := bindFileToWorkOrder(ctx, tx, provider, organizationID, factoryID, workOrderID, &file, &openCount); err != nil {
			return err
		}
	}
	return nil
}

func bindFileToWorkOrder(
	ctx context.Context,
	tx *gorm.DB,
	provider blob.Provider,
	organizationID, factoryID, workOrderID uuid.UUID,
	file *models.File,
	openCount *int64,
) error {
	if file.State != models.FileStateReady {
		return fmt.Errorf("%w: %s", models.ErrFileNotReady, file.ID)
	}
	if file.OrganizationID == nil || *file.OrganizationID != organizationID {
		return models.ErrFileForeignReference
	}
	if file.FactoryID == nil || *file.FactoryID != factoryID {
		return models.ErrFileForeignReference
	}

	switch file.Scope {
	case blob.ScopeTask:
		if file.WorkOrderID == nil || *file.WorkOrderID != workOrderID {
			return models.ErrFileForeignReference
		}
		return nil
	case blob.ScopeWorkspace:
		if *openCount >= models.MaxFilesPerWorkOrder {
			return fmt.Errorf("%w: task file limit is %d", models.ErrFileQuotaExceeded, models.MaxFilesPerWorkOrder)
		}
		if err := reparentWorkspaceFile(ctx, tx, provider, workOrderID, file); err != nil {
			return err
		}
		*openCount++
		return nil
	default:
		return models.ErrFileForeignReference
	}
}

func reparentWorkspaceFile(
	ctx context.Context,
	tx *gorm.DB,
	provider blob.Provider,
	workOrderID uuid.UUID,
	file *models.File,
) error {
	if provider == nil {
		return blob.ErrProviderNotConfigured
	}
	orgID := uuid.Nil
	if file.OrganizationID != nil {
		orgID = *file.OrganizationID
	}
	factoryID := uuid.Nil
	if file.FactoryID != nil {
		factoryID = *file.FactoryID
	}
	nextKey, err := blob.ObjectKey(file.InstallationID, blob.ScopeTask, orgID, factoryID, workOrderID, file.ID)
	if err != nil {
		return err
	}
	if nextKey != file.StorageKey {
		reader, err := provider.Get(ctx, file.StorageKey)
		if err != nil {
			return err
		}
		putErr := provider.Put(ctx, nextKey, reader, blob.PutOptions{ContentType: file.ContentType})
		_ = reader.Close()
		if putErr != nil {
			return putErr
		}
		if err := provider.Delete(ctx, file.StorageKey); err != nil {
			return err
		}
	}
	return file.ReparentToTask(tx, workOrderID, nextKey)
}

func CompleteUpload(ctx context.Context, tx *gorm.DB, provider blob.Provider, file *models.File, body io.Reader) error {
	if provider == nil {
		return blob.ErrProviderNotConfigured
	}
	if file.State != models.FileStatePending {
		return fmt.Errorf("%w: file is not pending", models.ErrFileInvalid)
	}

	hasher := sha256.New()
	limited := &limitedReader{r: io.TeeReader(body, hasher), n: models.MaxFileBytes}
	if err := provider.Put(ctx, file.StorageKey, limited, blob.PutOptions{ContentType: file.ContentType}); err != nil {
		_ = provider.Delete(ctx, file.StorageKey)
		_ = file.MarkFailed(tx)
		if errors.Is(err, errFileTooLarge) {
			return fmt.Errorf("%w: file exceeds %d bytes", models.ErrFileQuotaExceeded, models.MaxFileBytes)
		}
		return err
	}
	if limited.exceeded {
		_ = provider.Delete(ctx, file.StorageKey)
		_ = file.MarkFailed(tx)
		return fmt.Errorf("%w: file exceeds %d bytes", models.ErrFileQuotaExceeded, models.MaxFileBytes)
	}

	info, err := provider.Head(ctx, file.StorageKey)
	if err != nil {
		_ = file.MarkFailed(tx)
		return err
	}
	size := info.Size
	if size <= 0 {
		size = limited.read
	}
	if err := file.MarkReady(tx, size, hex.EncodeToString(hasher.Sum(nil))); err != nil {
		_ = provider.Delete(ctx, file.StorageKey)
		_ = file.MarkFailed(tx)
		return err
	}
	return nil
}

func DownloadURL(ctx context.Context, provider blob.Provider, file *models.File, ttl time.Duration) (string, error) {
	if file.State != models.FileStateReady {
		return "", models.ErrFileNotReady
	}
	return blob.ResolveDownloadURL(ctx, provider, file.StorageKey, file.ID, ttl)
}

func ContentUploadURL(fileID uuid.UUID) string {
	return blob.PublicBaseURL() + "/api/v1/files/" + fileID.String() + "/content"
}

func DeleteObjectAndRow(ctx context.Context, tx *gorm.DB, provider blob.Provider, file *models.File) error {
	if provider != nil && strings.TrimSpace(file.StorageKey) != "" {
		if err := provider.Delete(ctx, file.StorageKey); err != nil && !errors.Is(err, blob.ErrNotFound) {
			return err
		}
	}
	return file.Delete(tx)
}

type FetchFunc func(ctx context.Context, req *http.Request) (*http.Response, error)

func FetcherFromHTTP(httpCtx core.HTTPContext) FetchFunc {
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		return httpCtx.Do(req.WithContext(ctx))
	}
}

func IngestRemoteImages(
	ctx context.Context,
	tx *gorm.DB,
	provider blob.Provider,
	fetch FetchFunc,
	allow func(string) bool,
	organizationID, factoryID, workOrderID uuid.UUID,
	createdBy *uuid.UUID,
	markdown string,
) (string, error) {
	if fetch == nil || strings.TrimSpace(markdown) == "" {
		return markdown, nil
	}

	next := markdown
	openCount, err := models.CountOpenTaskFiles(tx, workOrderID)
	if err != nil {
		return markdown, err
	}

	for _, rawURL := range blob.HTTPImageURLs(markdown) {
		if allow != nil && !allow(rawURL) {
			continue
		}
		if openCount >= models.MaxFilesPerWorkOrder {
			break
		}
		file, ingested, err := ingestOneImage(
			ctx,
			tx,
			provider,
			fetch,
			organizationID,
			factoryID,
			workOrderID,
			createdBy,
			rawURL,
		)
		if err != nil || !ingested {
			continue
		}
		next = blob.ReplaceURL(next, rawURL, blob.FileRef(file.ID))
		openCount++
	}
	return next, nil
}

func ingestOneImage(
	ctx context.Context,
	tx *gorm.DB,
	provider blob.Provider,
	fetch FetchFunc,
	organizationID, factoryID, workOrderID uuid.UUID,
	createdBy *uuid.UUID,
	rawURL string,
) (*models.File, bool, error) {
	if provider == nil {
		return nil, false, blob.ErrProviderNotConfigured
	}

	fetchCtx, cancel := context.WithTimeout(ctx, ingestFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, nil
	}
	resp, err := fetch(fetchCtx, req)
	if err != nil {
		return nil, false, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false, nil
	}

	contentType := normalizeFetchedContentType(resp.Header.Get("Content-Type"), rawURL)
	if !models.IsAllowedFileContentType(contentType) {
		return nil, false, nil
	}

	createdByID := uuid.Nil
	if createdBy != nil {
		createdByID = *createdBy
	}
	file, err := models.CreatePendingFile(tx, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: organizationID,
		FactoryID:      factoryID,
		WorkOrderID:    workOrderID,
		Filename:       filenameFromURL(rawURL, contentType),
		ContentType:    contentType,
		CreatedByID:    createdByID,
	})
	if err != nil {
		return nil, false, nil
	}

	if err := CompleteUpload(fetchCtx, tx, provider, file, io.LimitReader(resp.Body, maxIngestBytes+1)); err != nil {
		_ = DeleteObjectAndRow(fetchCtx, tx, provider, file)
		return nil, false, nil
	}
	return file, true, nil
}

func filenameFromURL(rawURL, contentType string) string {
	base := path.Base(strings.TrimSpace(rawURL))
	if query := strings.Index(base, "?"); query >= 0 {
		base = base[:query]
	}
	if base == "" || base == "." || base == "/" {
		base = "image"
	}
	if filepathExt := path.Ext(base); filepathExt == "" {
		switch contentType {
		case "image/png":
			base += ".png"
		case "image/jpeg":
			base += ".jpg"
		case "image/gif":
			base += ".gif"
		case "image/webp":
			base += ".webp"
		case "application/pdf":
			base += ".pdf"
		}
	}
	return base
}

func normalizeFetchedContentType(header, rawURL string) string {
	value := strings.ToLower(strings.TrimSpace(header))
	if idx := strings.Index(value, ";"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	if models.IsAllowedFileContentType(value) {
		return value
	}
	switch strings.ToLower(path.Ext(strings.TrimSpace(rawURL))) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	default:
		return value
	}
}

var errFileTooLarge = errors.New("file exceeds the maximum size")

type limitedReader struct {
	r        io.Reader
	n        int64
	read     int64
	exceeded bool
}

func (l *limitedReader) Read(p []byte) (int, error) {
	if l.read >= l.n {
		buf := make([]byte, 1)
		n, err := l.r.Read(buf)
		if n > 0 {
			l.exceeded = true
			return 0, errFileTooLarge
		}
		return 0, err
	}
	remaining := l.n - l.read
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err := l.r.Read(p)
	l.read += int64(n)
	return n, err
}
