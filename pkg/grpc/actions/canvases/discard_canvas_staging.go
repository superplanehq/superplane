package canvases

import (
	"context"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DiscardCanvasStaging(
	ctx context.Context,
	organizationID string,
	canvasID string,
	versionID string,
	paths []string,
) (*pb.DiscardCanvasStagingResponse, error) {
	_, version, _, err := loadOwnedDraftVersion(ctx, organizationID, canvasID, versionID)
	if err != nil {
		return nil, err
	}

	if err := models.DiscardWorkflowStaging(version.ID, paths); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to discard staging: %v", err)
	}

	state, _, err := stagingSummaryForVersion(version.ID)
	if err != nil {
		return nil, err
	}

	return &pb.DiscardCanvasStagingResponse{StagingSummary: state}, nil
}
