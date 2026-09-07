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
)

const (
	FileStatePending = "pending"
	FileStateReady   = "ready"
	FileStateFailed  = "failed"

	MaxFileBytes             = 10 << 20
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
	CreatedByID    *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (File) TableName() string {
	return "files"
}

type CreateFileParams struct {
	Scope          string
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	Filename       string
	ContentType    string
	CreatedByID    uuid.UUID
}

func IsAllowedFileContentType(contentType string) bool {
	return slices.Contains(allowedFileContentTypes, normalizeContentType(contentType))
}

func IsInlineImageContentType(contentType string) bool {
	switch normalizeContentType(contentType) {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func CreatePendingFile(tx *gorm.DB, params CreateFileParams) (*File, error) {
	filename, err := sanitizeFileName(params.Filename)
	if err != nil {
		return nil, err
	}
	contentType := normalizeContentType(params.ContentType)
	if !IsAllowedFileContentType(contentType) {
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

	if err := ensureFileQuota(tx, params); err != nil {
		return nil, err
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
		CreatedAt:      now,
		UpdatedAt:      now,
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

	if err := tx.Create(file).Error; err != nil {
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

func ListReadyWorkspaceFiles(tx *gorm.DB, factoryID uuid.UUID) ([]File, error) {
	var files []File
	err := tx.Where(
		"factory_id = ? AND scope = ? AND state = ?",
		factoryID,
		blob.ScopeWorkspace,
		FileStateReady,
	).Order("created_at ASC, id ASC").Find(&files).Error
	return files, err
}

func ListReadyTaskFiles(tx *gorm.DB, workOrderID uuid.UUID) ([]File, error) {
	var files []File
	err := tx.Where(
		"work_order_id = ? AND scope = ? AND state = ?",
		workOrderID,
		blob.ScopeTask,
		FileStateReady,
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
	if sizeBytes > MaxFileBytes {
		return fmt.Errorf("%w: file exceeds %d bytes", ErrFileQuotaExceeded, MaxFileBytes)
	}
	now := time.Now()
	f.State = FileStateReady
	f.SizeBytes = sizeBytes
	f.UpdatedAt = now
	updates := map[string]any{
		"state":      FileStateReady,
		"size_bytes": sizeBytes,
		"updated_at": now,
	}
	if checksum != "" {
		f.Checksum = &checksum
		updates["checksum"] = checksum
	}
	return tx.Model(f).Updates(updates).Error
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

func ensureFileQuota(tx *gorm.DB, params CreateFileParams) error {
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
		count, err := CountReadyTaskFiles(tx, params.WorkOrderID)
		if err != nil {
			return err
		}
		if count >= MaxFilesPerWorkOrder {
			return fmt.Errorf("%w: task file limit is %d", ErrFileQuotaExceeded, MaxFilesPerWorkOrder)
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
	return value
}
