package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListAccessibleOrganizations(ctx context.Context, req *pb.ListAccessibleOrganizationsRequest, authService authorization.Authorization) (*pb.ListAccessibleOrganizationsResponse, error) {
	err := actions.ValidateUUIDs(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	orgIDs, err := authService.GetAccessibleOrgsForUser(req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get accessible organizations")
	}

	return &pb.ListAccessibleOrganizationsResponse{
		OrgIds: orgIDs,
	}, nil
}
