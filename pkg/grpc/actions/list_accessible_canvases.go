package actions

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListAccessibleCanvases(ctx context.Context, req *pb.ListAccessibleCanvasesRequest, authService authorization.AuthorizationServiceInterface) (*pb.ListAccessibleCanvasesResponse, error) {
	// Validate UUID
	err := ValidateUUIDs(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	// Get accessible canvases
	canvasIDs, err := authService.GetAccessibleCanvasesForUser(req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get accessible canvases")
	}

	return &pb.ListAccessibleCanvasesResponse{
		CanvasIds: canvasIDs,
	}, nil
}
