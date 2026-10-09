package storedfiles

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// CopiedTaskFiles is a task-file copy for a forked work order.
// CopiedKeys are new objects. Delete them when the fork transaction fails.
type CopiedTaskFiles struct {
	Description string
	Plan        string
	CopiedKeys  []string
	refs        map[uuid.UUID]string
}

// Rewrite replaces copied sp-file:// ids in markdown.
func (c CopiedTaskFiles) Rewrite(markdown string) string {
	if markdown == "" || len(c.refs) == 0 {
		return markdown
	}
	return blob.RewriteFileRefs(markdown, c.refs)
}

// CopyReadyTaskFiles copies each ready task file onto the target task.
// BindDescriptionFiles rejects a file owned by another task, and a workspace
// reparent would move the source file, so the fork must duplicate first.
func CopyReadyTaskFiles(
	ctx context.Context,
	tx *gorm.DB,
	provider blob.Provider,
	sourceOrderID, targetOrderID, createdByID uuid.UUID,
	description, plan string,
) (result CopiedTaskFiles, err error) {
	result.Description = description
	result.Plan = plan
	result.refs = map[uuid.UUID]string{}

	defer func() {
		if err != nil {
			_ = DeleteObjects(ctx, provider, result.CopiedKeys)
		}
	}()

	sources, err := taskFilesToCopy(tx, sourceOrderID, description, plan)
	if err != nil {
		return result, err
	}
	if len(sources) == 0 {
		return result, nil
	}
	if provider == nil {
		return result, blob.ErrProviderNotConfigured
	}

	for i := range sources {
		copied, copyErr := copyReadyTaskFile(ctx, tx, provider, targetOrderID, createdByID, &sources[i])
		if copyErr != nil {
			err = copyErr
			return result, err
		}
		result.CopiedKeys = append(result.CopiedKeys, copied.StorageKey)
		result.refs[sources[i].ID] = blob.FileRef(copied.ID)
	}

	result.Description = result.Rewrite(description)
	result.Plan = result.Rewrite(plan)
	return result, nil
}

func taskFilesToCopy(tx *gorm.DB, sourceOrderID uuid.UUID, description, plan string) ([]models.File, error) {
	ready, err := models.ListReadyTaskFiles(tx, sourceOrderID)
	if err != nil {
		return nil, err
	}
	byID := map[uuid.UUID]models.File{}
	for i := range ready {
		byID[ready[i].ID] = ready[i]
	}

	referenced := append(blob.FileIDsInMarkdown(description), blob.FileIDsInMarkdown(plan)...)
	if len(referenced) == 0 {
		return ready, nil
	}
	listed, err := models.ListFilesByIDs(tx, referenced)
	if err != nil {
		return nil, err
	}
	found := map[uuid.UUID]models.File{}
	for i := range listed {
		found[listed[i].ID] = listed[i]
	}

	ordered := make([]models.File, 0, len(ready))
	seen := map[uuid.UUID]struct{}{}
	appendFile := func(file models.File) {
		if _, ok := seen[file.ID]; ok {
			return
		}
		seen[file.ID] = struct{}{}
		ordered = append(ordered, file)
	}
	for i := range ready {
		appendFile(ready[i])
	}

	for _, id := range referenced {
		file, ok := found[id]
		if !ok {
			return nil, fmt.Errorf("%w: %s", models.ErrFileNotFound, id)
		}
		if _, copied := byID[id]; copied {
			continue
		}
		if err := referencedTaskFile(file, sourceOrderID); err != nil {
			return nil, err
		}
		if file.Scope == blob.ScopeTask && file.WorkOrderID != nil && *file.WorkOrderID == sourceOrderID {
			appendFile(file)
		}
	}
	return ordered, nil
}

func referencedTaskFile(file models.File, sourceOrderID uuid.UUID) error {
	if file.State != models.FileStateReady {
		return fmt.Errorf("%w: %s", models.ErrFileNotReady, file.ID)
	}
	if file.Scope != blob.ScopeTask {
		return nil
	}
	if file.WorkOrderID == nil || *file.WorkOrderID != sourceOrderID {
		return models.ErrFileForeignReference
	}
	return nil
}

func copyReadyTaskFile(
	ctx context.Context,
	tx *gorm.DB,
	provider blob.Provider,
	targetOrderID, createdByID uuid.UUID,
	source *models.File,
) (*models.File, error) {
	if source.OrganizationID == nil || source.FactoryID == nil {
		return nil, models.ErrFileForeignReference
	}
	reader, err := provider.Get(ctx, source.StorageKey)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	created, err := models.CreatePendingFile(tx, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: *source.OrganizationID,
		FactoryID:      *source.FactoryID,
		WorkOrderID:    targetOrderID,
		Filename:       source.Filename,
		ContentType:    source.ContentType,
		CreatedByID:    createdByID,
		Purpose:        source.Purpose,
	})
	if err != nil {
		return nil, err
	}
	if err := CompleteUpload(ctx, tx, provider, created, reader); err != nil {
		return nil, err
	}
	return created, nil
}
