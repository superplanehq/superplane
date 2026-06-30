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
	branchName string,
	paths []string,
) (*pb.DiscardCanvasStagingResponse, error) {
	canvas, branch, headVersion, userUUID, err := loadBranchForStaging(ctx, organizationID, canvasID, branchName, versionID)
	if err != nil {
		return nil, err
	}

	if err := models.DiscardWorkflowStaging(branch.ID, userUUID, paths); err != nil {
		return nil, grpcerrors.Internal(err, "failed to discard staging")
	}

	state, _, err := stagingSummaryForBranch(branch.ID, userUUID)
	if err != nil {
		return nil, err
	}

	publishStagingUpdated(canvas.ID, headVersion.ID)

	return &pb.DiscardCanvasStagingResponse{StagingSummary: state}, nil
}
