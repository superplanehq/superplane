package materialize

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type draftMaterializer struct {
	GitProvider git.Provider
	Registry    *registry.Registry
}

// persist runs the draft materialization inside a single transaction: it holds
// the branch lock, runs the authoritative idempotency check, creates a new
// CanvasVersion instance, and upserts the workflow_versions row.
func (m *draftMaterializer) persist(
	canvasID uuid.UUID,
	branchName string,
	headSHA string,
	createdBy *uuid.UUID,
	displayNameOverride string,
	snapshot *repoSnapshot,
) (*models.CanvasVersion, error) {
	var version *models.CanvasVersion
	txErr := database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := lockBranchMaterialization(tx, canvasID, branchName); err != nil {
			return err
		}

		draftVersion, done, idErr := draftAlreadyMaterializedInTransaction(tx, canvasID, branchName, headSHA)
		if idErr != nil {
			return idErr
		}
		if done {
			version = draftVersion
			return nil
		}

		if draftVersion == nil {
			label := strings.TrimSpace(displayNameOverride)
			if label == "" {
				generated, genErr := models.NextDraftDisplayNameInTransaction(tx, canvasID)
				if genErr != nil {
					return genErr
				}
				label = generated
			}

			now := time.Now()
			draftVersion = &models.CanvasVersion{
				ID:          uuid.New(),
				WorkflowID:  canvasID,
				OwnerID:     createdBy,
				State:       models.CanvasVersionStateDraft,
				DisplayName: label,
				GitBranch:   branchName,
				CreatedAt:   &now,
				UpdatedAt:   &now,
			}
			if createErr := tx.Create(draftVersion).Error; createErr != nil {
				return createErr
			}
		}

		v, matErr := m.materializeDraftInTransaction(tx, canvasID, branchName, headSHA, draftVersion.OwnerID, snapshot)
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

// draftAlreadyMaterializedInTransaction is authoritative, lock-protected
// idempotency check for a draft branch.
//
// It returns the existing draft version for branch (nil when none exists) and a
// boolean flag that is true when that version is already materialized at headSHA.
func draftAlreadyMaterializedInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	branch string,
	headSHA string,
) (*models.CanvasVersion, bool, error) {
	version, err := models.FindDraftVersionByBranchInTransaction(tx, canvasID, branch)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return version, materializedAt(version, headSHA), nil
}

func (m *draftMaterializer) materializeDraftInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	branch string,
	commitSHA string,
	ownerID *uuid.UUID,
	snapshot *repoSnapshot,
) (*models.CanvasVersion, error) {
	if m == nil || m.GitProvider == nil {
		return nil, fmt.Errorf("draft materializer is not configured")
	}
	if !models.IsDraftBranch(branch) {
		return nil, fmt.Errorf("branch %q is not a draft branch", branch)
	}

	now := time.Now()
	version := &models.CanvasVersion{
		WorkflowID:            canvasID,
		OwnerID:               ownerID,
		State:                 models.CanvasVersionStateDraft,
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

	if err := models.UpsertMaterializedVersionInTransaction(tx, version); err != nil {
		return nil, err
	}

	publishCanvasVersionUpdated(canvasID.String(), version.ID.String())

	return version, nil
}
