package materialize

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/canvas/gitref"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

type syncDraftBranchOptions struct {
	HeadSHA             string
	CreatedBy           *uuid.UUID
	DisplayNameOverride string
}

// syncDraftBranchFromGit registers draft workflow_versions metadata when missing
// and materializes the branch tip from git. It reads the head and loads the repo
// snapshot from git before opening the database transaction, so no git RPC is held
// across a pooled DB connection. Safe to call repeatedly.
func syncDraftBranchFromGit(
	ctx context.Context,
	gitProvider git.Provider,
	reg *registry.Registry,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	branchName string,
	opts syncDraftBranchOptions,
) (*models.CanvasVersion, error) {
	if !gitref.IsDraftBranch(branchName) {
		return nil, fmt.Errorf("branch %q is not a draft branch", branchName)
	}

	if gitProvider == nil {
		return nil, fmt.Errorf("git provider is not configured")
	}

	repository, err := models.FindRepositoryUnscoped(canvasID)
	if err != nil {
		return nil, fmt.Errorf("repository not found: %w", err)
	}

	headSHA := strings.TrimSpace(opts.HeadSHA)
	if headSHA == "" {
		headSHA, err = gitProvider.Head(ctx, repository.RepoID, branchName)
		if err != nil {
			return nil, fmt.Errorf("failed to read branch head: %w", err)
		}
	}

	snapshot, loadErr := loadRepoSnapshot(ctx, gitProvider, reg, orgID, repository.RepoID, headSHA)
	if loadErr != nil {
		ownerID := gitref.OwnerFromDraftBranchName(branchName)
		if ownerID == nil {
			ownerID = opts.CreatedBy
		}
		persistDraftMaterializationError(canvasID, branchName, ownerID, headSHA, loadErr)
		return nil, loadErr
	}

	var version *models.CanvasVersion
	txErr := database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := lockBranchMaterialization(tx, canvasID, branchName); err != nil {
			return err
		}

		draftVersion, err := models.FindDraftVersionByBranchInTransaction(tx, canvasID, branchName)
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		if draftVersion == nil {
			label := strings.TrimSpace(opts.DisplayNameOverride)
			if label == "" {
				label, err = nextDraftDisplayNameInTransaction(tx, canvasID)
				if err != nil {
					return err
				}
			}

			ownerID := gitref.OwnerFromDraftBranchName(branchName)
			if ownerID == nil {
				ownerID = opts.CreatedBy
			}

			now := time.Now()
			branch := branchName
			draftVersion = &models.CanvasVersion{
				ID:          uuid.New(),
				WorkflowID:  canvasID,
				OwnerID:     ownerID,
				State:       models.CanvasVersionStateDraft,
				BranchName:  &branch,
				DisplayName: label,
				GitBranch:   branchName,
				CreatedAt:   &now,
				UpdatedAt:   &now,
			}
			if createErr := tx.Create(draftVersion).Error; createErr != nil {
				return createErr
			}
		}

		mat := &draftMaterializer{GitProvider: gitProvider, Registry: reg}
		v, matErr := mat.materializeDraft(ctx, tx, orgID, canvasID, branchName, headSHA, draftVersion.OwnerID, snapshot)
		if matErr != nil {
			return matErr
		}

		if v.DisplayName == "" && draftVersion.DisplayName != "" {
			v.DisplayName = draftVersion.DisplayName
			if saveErr := tx.Save(v).Error; saveErr != nil {
				return saveErr
			}
		}

		version = v
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return version, nil
}

var draftDisplayNamePattern = regexp.MustCompile(`^Draft #(\d+)$`)

// NextDraftDisplayName returns a sequential label such as "Draft #1", "Draft #2".
func NextDraftDisplayName(canvasID uuid.UUID) (string, error) {
	return nextDraftDisplayNameInTransaction(database.Conn(), canvasID)
}

func nextDraftDisplayNameInTransaction(tx *gorm.DB, canvasID uuid.UUID) (string, error) {
	branches, err := models.ListAllDraftBranchVersionsForCanvasInTransaction(tx, canvasID)
	if err != nil {
		return "", err
	}

	highest := 0
	for _, branch := range branches {
		matches := draftDisplayNamePattern.FindStringSubmatch(branch.DisplayName)
		if matches == nil {
			continue
		}
		if n, convErr := strconv.Atoi(matches[1]); convErr == nil && n > highest {
			highest = n
		}
	}

	return fmt.Sprintf("Draft #%d", highest+1), nil
}
