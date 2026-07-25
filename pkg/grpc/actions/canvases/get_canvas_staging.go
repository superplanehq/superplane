package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

func GetCanvasStaging(
	ctx context.Context,
	db *gorm.DB,
	registry *registry.Registry,
	canvas *models.Canvas,
) (*CanvasStagingState, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	return BuildCanvasStagingState(ctx, db, registry, canvas, uuid.MustParse(userID))
}

func buildStagingSummary(canvas *models.Canvas, rows []models.WorkflowStagedFile) *pb.StagingSummary {
	state := &pb.StagingSummary{}
	if len(rows) == 0 {
		return state
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

	return state
}

func findStagingBaseVersionID(rows []models.WorkflowStagedFile) uuid.UUID {
	if len(rows) == 0 {
		return uuid.Nil
	}
	return rows[0].BaseVersionID
}
