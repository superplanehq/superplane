package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListOrganizationUsersForRole(ctx context.Context, req *pb.ListOrganizationUsersForRoleRequest, authService authorization.Authorization) (*pb.ListOrganizationUsersForRoleResponse, error) {
	err := actions.ValidateUUIDs(req.OrgId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	roleStr := req.GetRole()
	if roleStr == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}

	userIDs, err := authService.GetOrgUsersForRole(roleStr, req.OrgId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get organization users for role")
	}

	return &pb.ListOrganizationUsersForRoleResponse{
		UserIds: userIDs,
	}, nil
}

func ListCanvasUsersForRole(ctx context.Context, req *pb.ListCanvasUsersForRoleRequest, authService authorization.Authorization) (*pb.ListCanvasUsersForRoleResponse, error) {
	err := actions.ValidateUUIDs(req.CanvasId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas ID")
	}

	roleStr := req.GetRole()
	if roleStr == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}

	userIDs, err := authService.GetCanvasUsersForRole(roleStr, req.CanvasId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get canvas users for role")
	}

	return &pb.ListCanvasUsersForRoleResponse{
		UserIds: userIDs,
	}, nil
}
