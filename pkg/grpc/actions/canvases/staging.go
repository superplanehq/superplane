package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func buildStaging(ctx context.Context, canvas *models.Canvas, rows []models.WorkflowStagedFile) (*pb.Staging, error) {
	state := &pb.Staging{}

	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.DB(ctx), canvas.ID)
	if err != nil {
		return nil, err
	}

	spec, err := effectiveCanvasSpec(canvas, liveVersion, canvas.OrganizationID.String(), rows)
	if err != nil {
		return nil, err
	}
	state.Spec = spec

	if len(rows) == 0 {
		return state, nil
	}

	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		paths = append(paths, row.Path)
	}

	base := findStagingBaseVersionID(rows)
	state.HasStaging = true
	state.StagedPaths = paths
	state.BaseVersionId = base.String()
	state.Stale = canvas.LiveVersionID.String() != base.String()

	return state, nil
}

func findStagingBaseVersionID(rows []models.WorkflowStagedFile) uuid.UUID {
	if len(rows) == 0 {
		return uuid.Nil
	}
	return rows[0].BaseVersionID
}
