package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListDraftBranches(
	ctx context.Context,
	organizationID string,
	canvasID string,
) (*pb.ListDraftBranchesResponse, error) {
	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization id: %v", err)
	}

	if _, err := models.FindCanvas(orgUUID, canvasUUID); err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	branches, err := models.ListDraftBranchesForCanvas(canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list draft branches: %v", err)
	}

	states, err := models.ListRepositoryMaterializationStatesForCanvas(canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list materialization state: %v", err)
	}
	stateByBranch := make(map[string]*models.RepositoryMaterializationState, len(states))
	for i := range states {
		stateByBranch[states[i].Branch] = &states[i]
	}

	protoBranches := make([]*pb.CanvasDraftBranch, 0, len(branches))
	for i := range branches {
		protoBranches = append(protoBranches, serializeDraftBranch(&branches[i], organizationID, stateByBranch[branches[i].BranchName]))
	}

	return &pb.ListDraftBranchesResponse{Branches: protoBranches}, nil
}
