package materialize

import (
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// sweepDeletedDraftBranches drops every draft version whose git branch is absent
// from gitBranchSet, returning the IDs of the dropped versions (opportunistic healing path).
func sweepDeletedDraftBranches(canvasID uuid.UUID, gitBranchSet map[string]struct{}) ([]uuid.UUID, error) {
	var removed []uuid.UUID
	txErr := database.Conn().Transaction(func(tx *gorm.DB) error {
		dbBranches, err := models.ListAllDraftBranchVersionsForCanvasInTransaction(tx, canvasID)
		if err != nil {
			return err
		}

		for i := range dbBranches {
			branch := dbBranches[i].GitBranch
			if branch == "" {
				continue
			}
			if _, exists := gitBranchSet[branch]; exists {
				continue
			}

			if delErr := deleteDraftVersionInTransaction(tx, canvasID, &dbBranches[i]); delErr != nil {
				return delErr
			}
			removed = append(removed, dbBranches[i].ID)
		}

		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return removed, nil
}

// deleteDraftBranch drops the projection of a single draft branch, returning the
// dropped version ID. It returns uuid.Nil when there was no projection to delete.
func deleteDraftBranch(canvasID uuid.UUID, branch string) (uuid.UUID, error) {
	var removed uuid.UUID
	txErr := database.Conn().Transaction(func(tx *gorm.DB) error {
		version, err := models.FindDraftVersionByBranchInTransaction(tx, canvasID, branch)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}

		if delErr := deleteDraftVersionInTransaction(tx, canvasID, version); delErr != nil {
			return delErr
		}
		removed = version.ID

		return nil
	})
	if txErr != nil {
		return uuid.Nil, txErr
	}

	return removed, nil
}

func deleteDraftVersionInTransaction(tx *gorm.DB, canvasID uuid.UUID, version *models.CanvasVersion) error {
	if err := models.DeleteDraftVersionByBranchInTransaction(tx, canvasID, version.GitBranch); err != nil {
		return err
	}

	return models.DiscardWorkflowStagingInTransaction(tx, version.ID, nil)
}
