package models

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/blob"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FileStatePending      = "pending"
	FileStateReady        = "ready"
	FileStateFailed       = "failed"
	FilePurposeAttachment = "attachment"
	FilePurposeArtifact   = "artifact"

	MaxFileBytes             = 50 << 20
	MaxArtifactFileBytes     = 100 << 20
	MaxFilesPerWorkOrder     = 20
	MaxOrganizationFileBytes = 10 << 30
	MaxFileNameBytes         = 255
	StalePendingFileAge      = time.Hour
)

var (
	ErrFileNotFound         = errors.New("file not found")
	ErrFileInvalid          = errors.New("invalid file")
	ErrFileNotReady         = errors.New("file is not ready")
	ErrFileQuotaExceeded    = errors.New("file quota exceeded")
	ErrFileContentType      = errors.New("file content type is not allowed")
	ErrFileForeignReference = errors.New("file does not belong to this workspace")
)

var allowedFileContentTypes = []string{
	"image/png",
	"image/jpeg",
	"image/gif",
	"image/webp",
	"application/pdf",
	"text/plain",
	"text/markdown",
	"application/json",
	"text/csv",
	"application/yaml",
	"text/yaml",
	"application/x-yaml",
	"video/mp4",
	"video/webm",
	"video/quicktime",
	"video/ogg",
	"video/x-m4v",
	"video/x-matroska",
}

var allowedArtifactContentTypes = []string{
	"image/png",
	"image/jpeg",
	"image/webp",
	"video/webm",
	"video/mp4",
}

type File struct {
	ID             uuid.UUID
	InstallationID string
	Scope          string
	OrganizationID *uuid.UUID
	FactoryID      *uuid.UUID
	WorkOrderID    *uuid.UUID
	Filename       string
	ContentType    string
	SizeBytes      int64
	Checksum       *string
	StorageKey     string
	State          string
	Purpose        string `gorm:"default:attachment"`
	PublicID       *uuid.UUID
	CreatedByID    *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (File) TableName() string {
	return "files"
}

func (f File) IsDispatchable(organizationID, factoryID, workOrderID uuid.UUID) bool {
	if f.State != FileStateReady {
		return false
	}
	if f.Purpose != "" && f.Purpose != FilePurposeAttachment {
		return false
	}
	if f.OrganizationID == nil || *f.OrganizationID != organizationID {
		return false
	}
	if f.FactoryID == nil || *f.FactoryID != factoryID {
		return false
	}
	switch f.Scope {
	case blob.ScopeWorkspace:
		return true
	case blob.ScopeTask:
		if f.WorkOrderID == nil {
			return false
		}
		if workOrderID == uuid.Nil {
			return true
		}
		return *f.WorkOrderID == workOrderID
	default:
		return false
	}
}

type CreateFileParams struct {
	Scope          string
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	Filename       string
	ContentType    string
	CreatedByID    uuid.UUID
	Purpose        string
}

func IsAllowedFileContentType(contentType string) bool {
	return slices.Contains(allowedFileContentTypes, normalizeContentType(contentType))
}

func IsAllowedArtifactContentType(contentType string) bool {
	return slices.Contains(allowedArtifactContentTypes, normalizeContentType(contentType))
}

func (f File) MaxBytes() int64 {
	if f.Purpose == FilePurposeArtifact {
		return MaxArtifactFileBytes
	}
	return MaxFileBytes
}

func IsInlineImageContentType(contentType string) bool {
	switch normalizeContentType(contentType) {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func IsInlineVideoContentType(contentType string) bool {
	switch normalizeContentType(contentType) {
	case "video/mp4", "video/webm", "video/quicktime", "video/ogg", "video/x-m4v", "video/x-matroska":
		return true
	default:
		return false
	}
}

func IsInlineMediaContentType(contentType string) bool {
	return IsInlineImageContentType(contentType) || IsInlineVideoContentType(contentType)
}

func CreatePendingFile(tx *gorm.DB, params CreateFileParams) (*File, error) {
	filename, err := sanitizeFileName(params.Filename)
	if err != nil {
		return nil, err
	}
	purpose := strings.TrimSpace(params.Purpose)
	if purpose == "" {
		purpose = FilePurposeAttachment
	}
	if purpose != FilePurposeAttachment && purpose != FilePurposeArtifact {
		return nil, fmt.Errorf("%w: unknown file purpose %q", ErrFileInvalid, purpose)
	}
	contentType := normalizeContentType(params.ContentType)
	if purpose == FilePurposeAttachment && !IsAllowedFileContentType(contentType) {
		return nil, fmt.Errorf("%w: %s", ErrFileContentType, params.ContentType)
	}
	if purpose == FilePurposeArtifact && !IsAllowedArtifactContentType(contentType) {
		return nil, fmt.Errorf("%w: %s", ErrFileContentType, params.ContentType)
	}

	installationID, err := GetInstallationID(tx)
	if err != nil {
		return nil, err
	}

	id := uuid.New()
	storageKey, err := blob.ObjectKey(
		installationID,
		params.Scope,
		params.OrganizationID,
		params.FactoryID,
		params.WorkOrderID,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFileInvalid, err)
	}

	now := time.Now()
	file := &File{
		ID:             id,
		InstallationID: installationID,
		Scope:          params.Scope,
		Filename:       filename,
		ContentType:    contentType,
		StorageKey:     storageKey,
		State:          FileStatePending,
		Purpose:        purpose,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if purpose == FilePurposeArtifact {
		publicID := uuid.New()
		file.PublicID = &publicID
	}
	if params.OrganizationID != uuid.Nil {
		orgID := params.OrganizationID
		file.OrganizationID = &orgID
	}
	if params.FactoryID != uuid.Nil {
		factoryID := params.FactoryID
		file.FactoryID = &factoryID
	}
	if params.WorkOrderID != uuid.Nil {
		workOrderID := params.WorkOrderID
		file.WorkOrderID = &workOrderID
	}
	if params.CreatedByID != uuid.Nil {
		createdBy := params.CreatedByID
		file.CreatedByID = &createdBy
	}

	err = tx.Transaction(func(inner *gorm.DB) error {
		if err := ensureFileQuota(inner, params); err != nil {
			return err
		}
		return inner.Create(file).Error
	})
	if err != nil {
		return nil, err
	}
	return file, nil
}

func FindFile(tx *gorm.DB, id uuid.UUID) (*File, error) {
	var file File
	err := tx.Where("id = ?", id).First(&file).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	return &file, nil
}

func FindReadyArtifactFileByPublicID(tx *gorm.DB, publicID uuid.UUID) (*File, error) {
	var file File
	err := tx.Where(
		"public_id = ? AND purpose = ? AND state = ?",
		publicID,
		FilePurposeArtifact,
		FileStateReady,
	).First(&file).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	return &file, nil
}

func ListReadyWorkspaceFiles(tx *gorm.DB, factoryID uuid.UUID) ([]File, error) {
	var files []File
	err := tx.Where(
		"factory_id = ? AND scope = ? AND state = ? AND purpose = ?",
		factoryID,
		blob.ScopeWorkspace,
		FileStateReady,
		FilePurposeAttachment,
	).Order("created_at ASC, id ASC").Find(&files).Error
	return files, err
}

func ListReadyTaskFiles(tx *gorm.DB, workOrderID uuid.UUID) ([]File, error) {
	var files []File
	err := tx.Where(
		"work_order_id = ? AND scope = ? AND state = ? AND purpose = ?",
		workOrderID,
		blob.ScopeTask,
		FileStateReady,
		FilePurposeAttachment,
	).Order("created_at ASC, id ASC").Find(&files).Error
	return files, err
}

func ListFilesByIDs(tx *gorm.DB, ids []uuid.UUID) ([]File, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var files []File
	err := tx.Where("id IN ?", ids).Find(&files).Error
	return files, err
}

func RestoreFileRefs(tx *gorm.DB, organizationID, factoryID, workOrderID uuid.UUID, markdown string) (string, error) {
	urls := blob.SignedFileURLs(markdown)
	if len(urls) == 0 {
		return markdown, nil
	}

	ids := make([]uuid.UUID, 0, len(urls))
	seen := map[uuid.UUID]struct{}{}
	for _, raw := range urls {
		id, ok := blob.FileIDFromSignedURL(raw)
		if !ok {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	files, err := ListFilesByIDs(tx, ids)
	if err != nil {
		return markdown, err
	}
	allowed := map[uuid.UUID]struct{}{}
	for _, file := range files {
		if file.IsDispatchable(organizationID, factoryID, workOrderID) {
			allowed[file.ID] = struct{}{}
		}
	}

	return blob.RewriteSignedFileURLs(markdown, func(id uuid.UUID) (string, bool) {
		if _, ok := allowed[id]; !ok {
			return "", false
		}
		return blob.FileRef(id), true
	}), nil
}

func ListFilesForFactory(tx *gorm.DB, factoryID uuid.UUID, limit int) ([]File, error) {
	var files []File
	query := tx.Where("factory_id = ?", factoryID).Order("created_at ASC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&files).Error
	return files, err
}

func ListFilesForOrganization(tx *gorm.DB, organizationID uuid.UUID, limit int) ([]File, error) {
	var files []File
	query := tx.Where("organization_id = ?", organizationID).Order("created_at ASC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&files).Error
	return files, err
}

func ListStalePendingFiles(tx *gorm.DB, before time.Time, limit int) ([]File, error) {
	var files []File
	query := tx.Where(
		"state IN ? AND updated_at < ?",
		[]string{FileStatePending, FileStateFailed},
		before,
	).Order("updated_at ASC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&files).Error
	return files, err
}

func CountReadyTaskFiles(tx *gorm.DB, workOrderID uuid.UUID) (int64, error) {
	var count int64
	err := tx.Model(&File{}).Where(
		"work_order_id = ? AND scope = ? AND state = ?",
		workOrderID,
		blob.ScopeTask,
		FileStateReady,
	).Count(&count).Error
	return count, err
}

func CountOpenTaskFiles(tx *gorm.DB, workOrderID uuid.UUID) (int64, error) {
	var count int64
	err := tx.Model(&File{}).Where(
		"work_order_id = ? AND scope = ? AND state IN ?",
		workOrderID,
		blob.ScopeTask,
		[]string{FileStatePending, FileStateReady},
	).Count(&count).Error
	return count, err
}

func LockAndCountOpenTaskFiles(tx *gorm.DB, organizationID, workOrderID uuid.UUID) (int64, error) {
	if err := lockFileQuotaRows(tx, organizationID, workOrderID); err != nil {
		return 0, err
	}
	return CountOpenTaskFiles(tx, workOrderID)
}

func SumReadyOrganizationFileBytes(tx *gorm.DB, organizationID uuid.UUID) (int64, error) {
	var total int64
	err := tx.Model(&File{}).
		Select("COALESCE(SUM(size_bytes), 0)").
		Where("organization_id = ? AND state = ?", organizationID, FileStateReady).
		Scan(&total).Error
	return total, err
}

func (f *File) MarkReady(tx *gorm.DB, sizeBytes int64, checksum string) error {
	if sizeBytes <= 0 {
		return fmt.Errorf("%w: size must be greater than zero", ErrFileInvalid)
	}
	maxBytes := f.MaxBytes()
	if sizeBytes > maxBytes {
		return fmt.Errorf("%w: file exceeds %d bytes", ErrFileQuotaExceeded, maxBytes)
	}
	now := time.Now()
	updates := map[string]any{
		"state":      FileStateReady,
		"size_bytes": sizeBytes,
		"updated_at": now,
	}
	if checksum != "" {
		updates["checksum"] = checksum
	}

	err := tx.Transaction(func(inner *gorm.DB) error {
		if err := ensureReadyFileQuota(inner, f, sizeBytes); err != nil {
			return err
		}
		return inner.Model(f).Updates(updates).Error
	})
	if err != nil {
		return err
	}

	f.State = FileStateReady
	f.SizeBytes = sizeBytes
	f.UpdatedAt = now
	if checksum != "" {
		f.Checksum = &checksum
	}
	return nil
}

func (f *File) MarkFailed(tx *gorm.DB) error {
	now := time.Now()
	f.State = FileStateFailed
	f.UpdatedAt = now
	return tx.Model(f).Updates(map[string]any{
		"state":      FileStateFailed,
		"updated_at": now,
	}).Error
}

func (f *File) ReparentToTask(tx *gorm.DB, workOrderID uuid.UUID, storageKey string) error {
	if f.FactoryID == nil {
		return fmt.Errorf("%w: workspace file is missing a factory", ErrFileInvalid)
	}
	now := time.Now()
	f.Scope = blob.ScopeTask
	f.WorkOrderID = &workOrderID
	f.StorageKey = storageKey
	f.UpdatedAt = now
	return tx.Model(f).Updates(map[string]any{
		"scope":         blob.ScopeTask,
		"work_order_id": workOrderID,
		"storage_key":   storageKey,
		"updated_at":    now,
	}).Error
}

func (f *File) Delete(tx *gorm.DB) error {
	return tx.Delete(f).Error
}

func RememberAbandonedFileObjects(tx *gorm.DB, organizationID, factoryID uuid.UUID, keys []string) error {
	if organizationID == uuid.Nil || factoryID == uuid.Nil || len(keys) == 0 {
		return nil
	}
	installationID, err := GetInstallationID(tx)
	if err != nil {
		return err
	}
	now := time.Now()
	staleAt := now.Add(-StalePendingFileAge)
	orgID := organizationID
	facID := factoryID
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		file := File{
			ID:             uuid.New(),
			InstallationID: installationID,
			Scope:          blob.ScopeWorkspace,
			OrganizationID: &orgID,
			FactoryID:      &facID,
			Filename:       "abandoned-object",
			ContentType:    "text/plain",
			StorageKey:     key,
			State:          FileStateFailed,
			Purpose:        FilePurposeAttachment,
			CreatedAt:      now,
			UpdatedAt:      staleAt,
		}
		err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "storage_key"}},
			DoNothing: true,
		}).Create(&file).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func ensureFileQuota(tx *gorm.DB, params CreateFileParams) error {
	if err := lockFileQuotaRows(tx, params.OrganizationID, params.WorkOrderID); err != nil {
		return err
	}
	if params.OrganizationID != uuid.Nil {
		total, err := SumReadyOrganizationFileBytes(tx, params.OrganizationID)
		if err != nil {
			return err
		}
		if total >= MaxOrganizationFileBytes {
			return fmt.Errorf("%w: organization storage limit reached", ErrFileQuotaExceeded)
		}
	}
	if params.WorkOrderID != uuid.Nil {
		count, err := CountOpenTaskFiles(tx, params.WorkOrderID)
		if err != nil {
			return err
		}
		if count >= MaxFilesPerWorkOrder {
			return fmt.Errorf("%w: task file limit is %d", ErrFileQuotaExceeded, MaxFilesPerWorkOrder)
		}
	}
	return nil
}

func ensureReadyFileQuota(tx *gorm.DB, file *File, sizeBytes int64) error {
	orgID := uuid.Nil
	if file.OrganizationID != nil {
		orgID = *file.OrganizationID
	}
	workOrderID := uuid.Nil
	if file.WorkOrderID != nil {
		workOrderID = *file.WorkOrderID
	}
	if err := lockFileQuotaRows(tx, orgID, workOrderID); err != nil {
		return err
	}

	if file.OrganizationID != nil {
		total, err := SumReadyOrganizationFileBytes(tx, *file.OrganizationID)
		if err != nil {
			return err
		}
		if total+sizeBytes > MaxOrganizationFileBytes {
			return fmt.Errorf("%w: organization storage limit reached", ErrFileQuotaExceeded)
		}
	}
	if file.WorkOrderID == nil || file.Scope != blob.ScopeTask {
		return nil
	}

	count, err := CountReadyTaskFiles(tx, *file.WorkOrderID)
	if err != nil {
		return err
	}
	if count >= MaxFilesPerWorkOrder {
		return fmt.Errorf("%w: task file limit is %d", ErrFileQuotaExceeded, MaxFilesPerWorkOrder)
	}
	return nil
}

func lockFileQuotaRows(tx *gorm.DB, organizationID, workOrderID uuid.UUID) error {
	if organizationID != uuid.Nil {
		var org Organization
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id").
			Where("id = ?", organizationID).
			First(&org).Error
		if err != nil {
			return err
		}
	}
	if workOrderID != uuid.Nil {
		var order FactoryWorkOrder
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id").
			Where("id = ?", workOrderID).
			First(&order).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func sanitizeFileName(name string) (string, error) {
	base := filepath.Base(strings.TrimSpace(name))
	base = strings.ReplaceAll(base, "\x00", "")
	if base == "" || base == "." || base == ".." {
		return "", fmt.Errorf("%w: filename is required", ErrFileInvalid)
	}
	if utf8.RuneCountInString(base) > MaxFileNameBytes {
		return "", fmt.Errorf("%w: filename is too long", ErrFileInvalid)
	}
	return base, nil
}

func normalizeContentType(contentType string) string {
	value := strings.ToLower(strings.TrimSpace(contentType))
	if idx := strings.Index(value, ";"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	if value == "image/jpg" {
		return "image/jpeg"
	}
	if value == "text/yaml" || value == "application/x-yaml" {
		return "application/yaml"
	}
	return value
}
