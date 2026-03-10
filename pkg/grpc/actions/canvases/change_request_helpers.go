package canvases

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func isOpenCanvasChangeRequestStatus(status string) bool {
	return status == models.CanvasChangeRequestStatusOpen
}

func resolveCanvasChangeRequestBaseAndLiveInTransaction(
	tx *gorm.DB,
	canvas *models.Canvas,
	request *models.CanvasChangeRequest,
) (baseNodes []models.Node, baseEdges []models.Edge, liveNodes []models.Node, liveEdges []models.Edge, err error) {
	if canvas.LiveVersionID == nil {
		return nil, nil, nil, nil, gorm.ErrRecordNotFound
	}

	liveVersion, liveErr := models.FindCanvasVersionInTransaction(tx, canvas.ID, *canvas.LiveVersionID)
	if liveErr != nil {
		return nil, nil, nil, nil, liveErr
	}
	liveNodes = append([]models.Node(nil), liveVersion.Nodes...)
	liveEdges = append([]models.Edge(nil), liveVersion.Edges...)

	var baseVersionID *uuid.UUID
	if request.BasedOnVersionID != nil {
		baseVersionID = request.BasedOnVersionID
	} else {
		baseVersionID = canvas.LiveVersionID
	}

	if baseVersionID == nil {
		baseNodes = append([]models.Node(nil), liveNodes...)
		baseEdges = append([]models.Edge(nil), liveEdges...)
		return baseNodes, baseEdges, liveNodes, liveEdges, nil
	}

	baseVersion, baseErr := models.FindCanvasVersionInTransaction(tx, canvas.ID, *baseVersionID)
	if baseErr != nil {
		return nil, nil, nil, nil, baseErr
	}

	baseNodes = append([]models.Node(nil), baseVersion.Nodes...)
	baseEdges = append([]models.Edge(nil), baseVersion.Edges...)
	return baseNodes, baseEdges, liveNodes, liveEdges, nil
}
