package canvases

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func DiscardCanvasStaging(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
	paths []string,
) (*pb.DiscardCanvasStagingResponse, error) {
	canvas, version, _, err := loadOwnedDraftVersion(ctx, organizationID, canvasID, versionID)
	if err != nil {
		return nil, err
	}

	if err := models.DiscardWorkflowStaging(version.ID, paths); err != nil {
		return nil, grpcerrors.Internal(err, "failed to discard staging")
	}

	state, _, err := stagingSummaryForVersion(version.ID)
	if err != nil {
		return nil, err
	}

	publishStagingUpdated(canvas.ID, version.ID)

	return &pb.DiscardCanvasStagingResponse{StagingSummary: state}, nil
}
