package storedfiles

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func DuplicateTaskFiles(
	ctx context.Context,
	tx *gorm.DB,
	provider blob.Provider,
	organizationID, factoryID, workOrderID, createdByID uuid.UUID,
	markdown string,
) (string, []string, error) {
	ids := blob.FileIDsInMarkdown(markdown)
	if len(ids) == 0 {
		return markdown, nil, nil
	}

	files, err := models.ListFilesByIDs(tx, ids)
	if err != nil {
		return markdown, nil, err
	}
	byID := map[uuid.UUID]models.File{}
	for _, file := range files {
		byID[file.ID] = file
	}

	var copiedKeys []string
	replacements := map[uuid.UUID]string{}
	for _, id := range ids {
		file, ok := byID[id]
		if !ok || !isCopyableTaskFile(file, organizationID, factoryID, workOrderID) {
			continue
		}
		if provider == nil {
			return markdown, copiedKeys, blob.ErrProviderNotConfigured
		}
		copiedID, storageKey, copyErr := copyTaskAttachment(ctx, tx, provider, organizationID, factoryID, workOrderID, createdByID, file)
		if storageKey != "" {
			copiedKeys = append(copiedKeys, storageKey)
		}
		if copyErr != nil {
			return markdown, copiedKeys, copyErr
		}
		replacements[id] = blob.FileRef(copiedID)
	}
	if len(replacements) == 0 {
		return markdown, copiedKeys, nil
	}
	return blob.RewriteFileRefs(markdown, replacements), copiedKeys, nil
}

func isCopyableTaskFile(file models.File, organizationID, factoryID, workOrderID uuid.UUID) bool {
	if file.State != models.FileStateReady {
		return false
	}
	if file.Purpose != "" && file.Purpose != models.FilePurposeAttachment {
		return false
	}
	if file.Scope != blob.ScopeTask {
		return false
	}
	if file.OrganizationID == nil || *file.OrganizationID != organizationID {
		return false
	}
	if file.FactoryID == nil || *file.FactoryID != factoryID {
		return false
	}
	if file.WorkOrderID == nil || *file.WorkOrderID == workOrderID {
		return false
	}
	return true
}

func copyTaskAttachment(
	ctx context.Context,
	tx *gorm.DB,
	provider blob.Provider,
	organizationID, factoryID, workOrderID, createdByID uuid.UUID,
	source models.File,
) (uuid.UUID, string, error) {
	created, err := models.CreatePendingFile(tx, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: organizationID,
		FactoryID:      factoryID,
		WorkOrderID:    workOrderID,
		Filename:       source.Filename,
		ContentType:    source.ContentType,
		CreatedByID:    createdByID,
		Purpose:        models.FilePurposeAttachment,
	})
	if err != nil {
		return uuid.Nil, "", err
	}

	reader, err := provider.Get(ctx, source.StorageKey)
	if err != nil {
		return uuid.Nil, created.StorageKey, err
	}
	defer reader.Close()

	if err := CompleteUpload(ctx, tx, provider, created, reader); err != nil {
		return uuid.Nil, created.StorageKey, err
	}
	return created.ID, created.StorageKey, nil
}
