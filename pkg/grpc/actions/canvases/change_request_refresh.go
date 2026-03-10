package canvases

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func refreshCanvasChangeRequestDiffInTransaction(
	tx *gorm.DB,
	canvas *models.Canvas,
	version *models.CanvasVersion,
	request *models.CanvasChangeRequest,
) error {
	baseNodes, baseEdges, liveNodes, liveEdges, err := resolveCanvasChangeRequestBaseAndLiveInTransaction(tx, canvas, request)
	if err != nil {
		return err
	}

	diff := computeCanvasChangeRequestDiff(baseNodes, baseEdges, liveNodes, liveEdges, version.Nodes, version.Edges)
	now := time.Now()

	request.ChangedNodeIDs = datatypes.NewJSONSlice(diff.ChangedNodeIDs)
	request.ConflictingNodeIDs = datatypes.NewJSONSlice(diff.ConflictingNodeIDs)
	request.UpdatedAt = &now

	if request.Status != models.CanvasChangeRequestStatusPublished &&
		request.Status != models.CanvasChangeRequestStatusRejected {
		request.Status = models.CanvasChangeRequestStatusOpen
	}

	return tx.Save(request).Error
}

func refreshOpenCanvasChangeRequestsInTransaction(
	tx *gorm.DB,
	organizationID uuid.UUID,
	canvasID uuid.UUID,
	skipRequestID uuid.UUID,
) error {
	canvas, err := models.FindCanvasInTransaction(tx, organizationID, canvasID)
	if err != nil {
		return err
	}

	var requests []models.CanvasChangeRequest
	if err := tx.
		Where("workflow_id = ?", canvasID).
		Where("id <> ?", skipRequestID).
		Where("status = ?", models.CanvasChangeRequestStatusOpen).
		Order("created_at DESC").
		Find(&requests).
		Error; err != nil {
		return err
	}

	for i := range requests {
		request := &requests[i]
		version, versionErr := models.FindCanvasVersionInTransaction(tx, canvasID, request.VersionID)
		if versionErr != nil {
			if errors.Is(versionErr, gorm.ErrRecordNotFound) {
				continue
			}
			return versionErr
		}

		if version.IsPublished {
			continue
		}

		if err := refreshCanvasChangeRequestDiffInTransaction(tx, canvas, version, request); err != nil {
			return err
		}
	}

	return nil
}
