package materialize

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type ReconcileDraftBranchDeletionsOptions struct {
	BranchName string
}

// ReconcileDraftBranchDeletionsFromGit removes draft workflow_versions rows whose git
// refs no longer exist. Safe to call repeatedly.
func ReconcileDraftBranchDeletionsFromGit(
	ctx context.Context,
	tx *gorm.DB,
	gitProvider git.Provider,
	canvasID uuid.UUID,
	opts ReconcileDraftBranchDeletionsOptions,
) ([]string, error) {
	if gitProvider == nil {
		return nil, fmt.Errorf("git provider is not configured")
	}

	repository, err := models.FindRepositoryInTransaction(tx, canvasID)
	if err != nil {
		return nil, fmt.Errorf("repository not found: %w", err)
	}

	gitBranches, err := gitProvider.ListBranches(ctx, repository.RepoID, DraftBranchPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list draft branches in git: %w", err)
	}

	gitBranchSet := make(map[string]struct{}, len(gitBranches))
	for _, branch := range gitBranches {
		gitBranchSet[branch] = struct{}{}
	}

	dbBranches, err := models.ListAllDraftBranchVersionsForCanvasInTransaction(tx, canvasID)
	if err != nil {
		return nil, err
	}

	filterBranch := strings.TrimSpace(opts.BranchName)
	removed := make([]string, 0)

	for i := range dbBranches {
		if dbBranches[i].BranchName == nil {
			continue
		}
		branchName := *dbBranches[i].BranchName
		if filterBranch != "" && branchName != filterBranch {
			continue
		}
		if _, exists := gitBranchSet[branchName]; exists {
			continue
		}

		if deleteErr := models.DeleteDraftVersionByBranchInTransaction(tx, canvasID, branchName); deleteErr != nil {
			return nil, deleteErr
		}
		if deleteErr := models.DiscardWorkflowStagingInTransaction(tx, dbBranches[i].ID, nil); deleteErr != nil {
			return nil, deleteErr
		}

		removed = append(removed, branchName)
	}

	return removed, nil
}

// PublishDraftBranchDeletionEvents notifies clients that draft branches were removed from git.
func PublishDraftBranchDeletionEvents(canvasID string, removed []string) {
	for _, branch := range removed {
		if publishErr := messages.NewRepositoryBranchUpdatedMessage(
			canvasID,
			branch,
			"",
			protoMaterializationStatus(models.MaterializationStatusDeleted),
			"",
			"",
		).PublishBranchUpdated(); publishErr != nil {
			log.Errorf("failed to publish draft branch deleted for canvas %s branch %s: %v", canvasID, branch, publishErr)
		}
	}
}
