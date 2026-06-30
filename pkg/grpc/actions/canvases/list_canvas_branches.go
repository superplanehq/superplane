package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func ListCanvasBranches(ctx context.Context, organizationID, canvasID string) (*pb.ListCanvasBranchesResponse, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	if err := checkCanvasExistence(ctx, database.DB(ctx), orgUUID, canvasUUID); err != nil {
		return nil, grpcerrors.NotFound(err, "canvas not found")
	}

	branches, err := models.ListWorkflowBranchesConn(canvasUUID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list branches")
	}

	protoBranches := make([]*pb.CanvasBranch, 0, len(branches))
	for i := range branches {
		protoBranches = append(protoBranches, serializeCanvasBranch(&branches[i]))
	}

	return &pb.ListCanvasBranchesResponse{Branches: protoBranches}, nil
}

func CreateCanvasBranch(
	ctx context.Context,
	organizationID string,
	canvasID string,
	name string,
	sourceBranch string,
) (*pb.CreateCanvasBranchResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid canvas id")
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	canvas, err := models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "canvas not found")
	}

	if sourceBranch == "" {
		sourceBranch = models.CanvasGitBranchMain
	}

	userUUID := uuid.MustParse(userID)
	var branch *models.WorkflowBranch

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		created, _, createErr := models.CreateBranchFromHeadInTransaction(
			tx,
			canvas.ID,
			sourceBranch,
			name,
			userUUID,
		)
		branch = created
		return createErr
	})
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create branch")
	}

	return &pb.CreateCanvasBranchResponse{
		Branch: serializeCanvasBranch(branch),
	}, nil
}

func serializeCanvasBranch(branch *models.WorkflowBranch) *pb.CanvasBranch {
	if branch == nil {
		return nil
	}

	protoBranch := &pb.CanvasBranch{
		Id:        branch.ID.String(),
		CanvasId:  branch.WorkflowID.String(),
		Name:      branch.Name,
		CreatedAt: timestamppb.New(branch.CreatedAt),
		UpdatedAt: timestamppb.New(branch.UpdatedAt),
	}
	if branch.HeadVersionID != nil {
		protoBranch.HeadVersionId = branch.HeadVersionID.String()
	}
	return protoBranch
}
