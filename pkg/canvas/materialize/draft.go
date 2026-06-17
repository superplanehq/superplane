package materialize

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
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

// MaterializeDraft writes the draft projection from a snapshot that the caller
// has already loaded from git outside of any transaction, so no git RPC is held
// across the DB connection.
func (m *DraftMaterializer) MaterializeDraft(
	ctx context.Context,
	tx *gorm.DB,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	branch string,
	commitSHA string,
	ownerID *uuid.UUID,
	snapshot *RepoSnapshot,
) (*models.CanvasVersion, error) {
	if m == nil || m.GitProvider == nil {
		return nil, fmt.Errorf("draft materializer is not configured")
	}

	if err := lockBranchMaterialization(tx, canvasID, branch); err != nil {
		return nil, err
	}

	if isDraftBranch(branch) {
		existing, err := models.FindDraftVersionByBranchInTransaction(tx, canvasID, branch)
		if err == nil &&
			existing.CommitSHA == commitSHA &&
			existing.MaterializationStatus == models.MaterializationStatusReady {
			return existing, nil
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
	} else {
		existing, err := models.FindVersionByCommitSHAInTransaction(tx, canvasID, commitSHA)
		if err == nil && existing.MaterializationStatus == models.MaterializationStatusReady {
			return existing, nil
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	now := time.Now()
	branchName := branch
	version := &models.CanvasVersion{
		WorkflowID:            canvasID,
		OwnerID:               ownerID,
		State:                 draftVersionState(branch),
		Name:                  snapshot.Name,
		Description:           snapshot.Description,
		Nodes:                 datatypes.NewJSONSlice(snapshot.Nodes),
		Edges:                 datatypes.NewJSONSlice(snapshot.Edges),
		ConsolePanels:         datatypes.NewJSONType(snapshot.ConsolePanels),
		ConsoleLayout:         datatypes.NewJSONType(snapshot.ConsoleLayout),
		CommitSHA:             commitSHA,
		GitBranch:             branch,
		MaterializationStatus: models.MaterializationStatusReady,
		MaterializationError:  "",
		CreatedAt:             &now,
		UpdatedAt:             &now,
	}
	if isDraftBranch(branch) {
		version.BranchName = &branchName
	}

	if err := models.UpsertMaterializedVersionInTransaction(tx, version); err != nil {
		return nil, err
	}

	if publishErr := messages.NewCanvasVersionUpdatedMessage(canvasID.String(), version.ID.String()).PublishVersionUpdated(); publishErr != nil {
		log.Errorf("failed to publish canvas version updated message: %v", publishErr)
	}
	if publishErr := messages.NewRepositoryBranchUpdatedMessage(
		canvasID.String(),
		branch,
		commitSHA,
		protoMaterializationStatus(models.MaterializationStatusReady),
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

func markMaterializationError(tx *gorm.DB, canvasID uuid.UUID, branch, commitSHA string, ownerID *uuid.UUID, cause error) error {
	errMsg := cause.Error()
	now := time.Now()
	branchName := branch
	version := &models.CanvasVersion{
		WorkflowID:            canvasID,
		OwnerID:               ownerID,
		State:                 draftVersionState(branch),
		CommitSHA:             commitSHA,
		GitBranch:             branch,
		MaterializationStatus: models.MaterializationStatusError,
		MaterializationError:  errMsg,
		CreatedAt:             &now,
		UpdatedAt:             &now,
	}
	if isDraftBranch(branch) {
		version.BranchName = &branchName
	}
	return models.UpsertMaterializedVersionInTransaction(tx, version)
}

// persistDraftMaterializationError records a draft branch materialization failure
// in its own transaction. It is used when the snapshot load fails before the main
// materialization transaction is opened.
func persistDraftMaterializationError(canvasID uuid.UUID, branch string, ownerID *uuid.UUID, headSHA string, cause error) {
	if err := database.Conn().Transaction(func(tx *gorm.DB) error {
		return markMaterializationError(tx, canvasID, branch, headSHA, ownerID, cause)
	}); err != nil {
		log.Errorf("failed to persist draft materialization error for canvas %s branch %s: %v", canvasID, branch, err)
	}
}
