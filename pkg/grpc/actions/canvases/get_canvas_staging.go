package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"gorm.io/gorm"
)

func GetCanvasStaging(ctx context.Context, organizationID string, canvasID string) (*pb.StagingSummary, error) {
	db := database.DB(ctx)

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid organization_id")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	canvas, err := models.FindCanvasInTransaction(db, organizationUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "canvas not found")
		}

		return nil, grpcerrors.Internal(err, "failed to load canvas")
	}

	userUUID := uuid.MustParse(userID)
	rows, err := models.ListStagedFilesForUser(db, canvas.ID, userUUID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load staging")
	}

	return buildStagingSummary(canvas, rows), nil
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
