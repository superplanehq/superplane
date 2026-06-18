package materialize

import (
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func materializedAt(version *models.CanvasVersion, commitSHA string) bool {
	return version != nil &&
		version.CommitSHA == commitSHA &&
		version.MaterializationStatus == models.MaterializationStatusReady
}

func publishCanvasUpdated(canvasID, organizationID string) {
	if publishErr := messages.NewCanvasUpdatedMessage(canvasID, organizationID).PublishUpdated(); publishErr != nil {
		log.Errorf("failed to publish canvas updated message: %v", publishErr)
	}
}

func publishCanvasVersionUpdated(canvasID, versionID string) {
	if publishErr := messages.NewCanvasVersionUpdatedMessage(canvasID, versionID).PublishVersionUpdated(); publishErr != nil {
		log.Errorf("failed to publish canvas version updated message: %v", publishErr)
	}
}

func persistLiveMaterializationError(canvasID uuid.UUID, headSHA string, cause error) uuid.UUID {
	var versionID uuid.UUID
	if err := database.Conn().Transaction(func(tx *gorm.DB) error {
		id, markErr := markMaterializationError(tx, canvasID, models.CanvasGitBranchMain, headSHA, nil, cause)
		versionID = id
		return markErr
	}); err != nil {
		log.Errorf("failed to persist live materialization error for canvas %s: %v", canvasID, err)
		return uuid.Nil
	}

	return versionID
}

func persistDraftMaterializationError(canvasID uuid.UUID, branch string, ownerID *uuid.UUID, headSHA string, cause error) uuid.UUID {
	var versionID uuid.UUID
	if err := database.Conn().Transaction(func(tx *gorm.DB) error {
		id, markErr := markMaterializationError(tx, canvasID, branch, headSHA, ownerID, cause)
		versionID = id
		return markErr
	}); err != nil {
		log.Errorf("failed to persist draft materialization error for canvas %s branch %s: %v", canvasID, branch, err)
		return uuid.Nil
	}

	return versionID
}

func markMaterializationError(tx *gorm.DB, canvasID uuid.UUID, branch, commitSHA string, ownerID *uuid.UUID, cause error) (uuid.UUID, error) {
	errMsg := cause.Error()
	now := time.Now()
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
	if err := models.UpsertMaterializedVersionInTransaction(tx, version); err != nil {
		return uuid.Nil, err
	}

	return version.ID, nil
}

func draftVersionState(branch string) string {
	if branch == models.CanvasGitBranchMain {
		return models.CanvasVersionStatePublished
	}
	return models.CanvasVersionStateDraft
}
