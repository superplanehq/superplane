package materialize

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DraftMaterializer struct {
	GitProvider git.Provider
	Registry    *registry.Registry
}

func (m *DraftMaterializer) MaterializeDraft(
	ctx context.Context,
	tx *gorm.DB,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	branch string,
	commitSHA string,
	ownerID *uuid.UUID,
) (*models.CanvasVersion, error) {
	if m == nil || m.GitProvider == nil {
		return nil, fmt.Errorf("draft materializer is not configured")
	}

	existing, err := models.FindVersionBySHAInTransaction(tx, canvasID, commitSHA)
	if err == nil && existing.MaterializationStatus == models.MaterializationStatusReady {
		return existing, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	repository, err := models.FindRepositoryInTransaction(tx, canvasID)
	if err != nil {
		return nil, fmt.Errorf("repository not found: %w", err)
	}

	snapshot, loadErr := LoadRepoSnapshot(ctx, m.GitProvider, m.Registry, orgID, repository.RepoID, commitSHA)
	if loadErr != nil {
		if markErr := markMaterializationError(tx, canvasID, branch, commitSHA, loadErr); markErr != nil {
			return nil, markErr
		}
		return nil, loadErr
	}

	now := time.Now()
	version := &models.CanvasVersion{
		ID:                      commitSHA,
		WorkflowID:              canvasID,
		OwnerID:                 ownerID,
		State:                   draftVersionState(branch),
		Name:                    snapshot.Name,
		Description:             snapshot.Description,
		ChangeManagementEnabled: snapshot.ChangeManagementEnabled,
		ChangeRequestApprovers:  datatypes.NewJSONSlice(snapshot.ChangeRequestApprovers),
		Nodes:                   datatypes.NewJSONSlice(snapshot.Nodes),
		Edges:                   datatypes.NewJSONSlice(snapshot.Edges),
		ConsolePanels:           datatypes.NewJSONType(snapshot.ConsolePanels),
		ConsoleLayout:           datatypes.NewJSONType(snapshot.ConsoleLayout),
		GitBranch:               branch,
		MaterializationStatus:   models.MaterializationStatusReady,
		MaterializationError:    "",
		CreatedAt:               &now,
		UpdatedAt:               &now,
	}

	if err := models.UpsertMaterializedVersionInTransaction(tx, version); err != nil {
		return nil, err
	}

	if isDraftBranch(branch) {
		if err := models.UpdateDraftBranchTipInTransaction(tx, canvasID, branch, commitSHA); err != nil {
			return nil, err
		}
	}

	if err := models.UpsertRepositoryMaterializationStateInTransaction(tx, &models.RepositoryMaterializationState{
		CanvasID:        canvasID,
		Branch:          branch,
		HeadSHA:         commitSHA,
		MaterializedSHA: commitSHA,
		Status:          models.MaterializationStatusReady,
		Error:           "",
	}); err != nil {
		return nil, err
	}

	if publishErr := messages.NewCanvasVersionUpdatedMessage(canvasID.String(), commitSHA).PublishVersionUpdated(); publishErr != nil {
		log.Errorf("failed to publish canvas version updated message: %v", publishErr)
	}
	if publishErr := messages.NewRepositoryBranchUpdatedMessage(
		canvasID.String(),
		branch,
		commitSHA,
		models.MaterializationStatusReady,
		"",
		"",
	).PublishBranchUpdated(); publishErr != nil {
		log.Errorf("failed to publish repository branch updated message: %v", publishErr)
	}

	return version, nil
}

func draftVersionState(branch string) string {
	if branch == models.CanvasGitBranchMain {
		return models.CanvasVersionStatePublished
	}
	return models.CanvasVersionStateDraft
}

func isDraftBranch(branch string) bool {
	return strings.HasPrefix(branch, DraftBranchPrefix)
}

func markMaterializationError(tx *gorm.DB, canvasID uuid.UUID, branch, commitSHA string, cause error) error {
	errMsg := cause.Error()
	now := time.Now()
	version := &models.CanvasVersion{
		ID:                    commitSHA,
		WorkflowID:            canvasID,
		State:                 draftVersionState(branch),
		GitBranch:             branch,
		MaterializationStatus: models.MaterializationStatusError,
		MaterializationError:  errMsg,
		CreatedAt:             &now,
		UpdatedAt:             &now,
	}
	if upsertErr := models.UpsertMaterializedVersionInTransaction(tx, version); upsertErr != nil {
		return upsertErr
	}

	return models.UpsertRepositoryMaterializationStateInTransaction(tx, &models.RepositoryMaterializationState{
		CanvasID:        canvasID,
		Branch:          branch,
		HeadSHA:         commitSHA,
		MaterializedSHA: "",
		Status:          models.MaterializationStatusError,
		Error:           errMsg,
	})
}
