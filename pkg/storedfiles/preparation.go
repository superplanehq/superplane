package storedfiles

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// Preparation uploads task objects before the task exists. Persist its rows
// after task insertion, and apply its object keys through BindResult on rollback.
type Preparation struct {
	files      []preparedFile
	bindings   []preparedBinding
	failedKeys []string
}

type preparedBinding struct {
	file        models.File
	workOrderID uuid.UUID
	storageKey  string
}

// PrepareBindings copies workspace objects before task number allocation.
// The database bindings remain part of the caller's creation transaction.
func (p *Preparation) PrepareBindings(ctx context.Context, tx *gorm.DB, provider blob.Provider, organizationID, factoryID, workOrderID uuid.UUID, markdown string) error {
	ids := blob.FileIDsInMarkdown(markdown)
	files, err := models.ListFilesByIDs(tx, ids)
	if err != nil {
		return err
	}
	byID := map[uuid.UUID]models.File{}
	for _, file := range files {
		byID[file.ID] = file
	}
	for _, prepared := range p.files {
		byID[prepared.file.ID] = *prepared.file
	}
	for _, id := range ids {
		file, ok := byID[id]
		if !ok {
			return fmt.Errorf("%w: %s", models.ErrFileNotFound, id)
		}
		if !file.IsDispatchable(organizationID, factoryID, workOrderID) {
			if file.State != models.FileStateReady {
				return models.ErrFileNotReady
			}
			return models.ErrFileForeignReference
		}
		if file.Scope == blob.ScopeTask {
			continue
		}
		if len(p.files)+len(p.bindings) >= models.MaxFilesPerWorkOrder {
			return models.ErrFileQuotaExceeded
		}
		if provider == nil {
			return blob.ErrProviderNotConfigured
		}
		key, err := blob.ObjectKey(file.InstallationID, blob.ScopeTask, organizationID, factoryID, workOrderID, file.ID)
		if err != nil {
			return err
		}
		reader, err := provider.Get(ctx, file.StorageKey)
		if err != nil {
			return err
		}
		p.bindings = append(p.bindings, preparedBinding{file: file, workOrderID: workOrderID, storageKey: key})
		err = provider.Put(ctx, key, reader, blob.PutOptions{ContentType: file.ContentType})
		_ = reader.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

type preparedFile struct {
	file   *models.File
	params models.CreateFileParams
}

func (p *Preparation) store(ctx context.Context, tx *gorm.DB, provider blob.Provider, params models.CreateFileParams, body io.Reader) (*models.File, error) {
	file, err := models.NewPendingFile(tx, params)
	if err != nil {
		return nil, err
	}
	upload, err := StorePendingUpload(ctx, provider, file, body)
	if err != nil {
		p.failedKeys = append(p.failedKeys, file.StorageKey)
		return nil, err
	}
	file.State = models.FileStateReady
	file.SizeBytes = upload.SizeBytes
	file.Checksum = &upload.Checksum
	p.files = append(p.files, preparedFile{file: file, params: params})
	return file, nil
}

func (p *Preparation) Persist(tx *gorm.DB) error {
	for _, prepared := range p.files {
		if err := prepared.file.Insert(tx, prepared.params); err != nil {
			return err
		}
	}
	if len(p.bindings) > 0 {
		first := p.bindings[0]
		count, err := models.LockAndCountOpenTaskFiles(tx, *first.file.OrganizationID, first.workOrderID)
		if err != nil {
			return err
		}
		if count+int64(len(p.bindings)) > models.MaxFilesPerWorkOrder {
			return models.ErrFileQuotaExceeded
		}
	}
	for _, binding := range p.bindings {
		if err := binding.file.ReparentToTask(tx, binding.workOrderID, binding.storageKey); err != nil {
			return err
		}
	}
	return nil
}

func (p *Preparation) ObjectKeys() []string {
	keys := make([]string, 0, len(p.files))
	for _, prepared := range p.files {
		keys = append(keys, prepared.file.StorageKey)
	}
	for _, binding := range p.bindings {
		keys = append(keys, binding.storageKey)
	}
	keys = append(keys, p.failedKeys...)
	return keys
}

func (p *Preparation) BindResult() BindResult {
	result := BindResult{CopiedKeys: p.ObjectKeys(), StaleKeys: append([]string(nil), p.failedKeys...)}
	for _, binding := range p.bindings {
		result.StaleKeys = append(result.StaleKeys, binding.file.StorageKey)
	}
	return result
}

func openFileCount(tx *gorm.DB, workOrderID uuid.UUID, preparation []*Preparation) (int64, error) {
	if len(preparation) > 0 {
		return int64(len(preparation[0].files)), nil
	}
	return models.CountOpenTaskFiles(tx, workOrderID)
}
