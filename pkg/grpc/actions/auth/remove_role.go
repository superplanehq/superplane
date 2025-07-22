package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RemoveRole(ctx context.Context, req *pb.RemoveRoleRequest, authService authorization.Authorization) (*pb.RemoveRoleResponse, error) {
	if req.RoleAssignment.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	var domainTypeStr string

	switch req.RoleAssignment.DomainType {
	case pb.DomainType_DOMAIN_TYPE_ORGANIZATION:
		domainTypeStr = models.DomainOrg
	case pb.DomainType_DOMAIN_TYPE_CANVAS:
		domainTypeStr = models.DomainCanvas
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported domain type")
	}

	roleStr := req.RoleAssignment.Role
	if roleStr == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}

	err := actions.ValidateUUIDs(req.RoleAssignment.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	var userID string
	switch identifier := req.UserIdentifier.(type) {
	case *pb.RemoveRoleRequest_UserId:
		err := actions.ValidateUUIDs(identifier.UserId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user ID")
		}
		userID = identifier.UserId
	case *pb.RemoveRoleRequest_UserEmail:
		// Find user by email - first try with account providers
		user, err := models.FindUserByEmail(identifier.UserEmail)
		if err != nil {
			// If not found by account provider, try to find inactive user by email
			user, err = models.FindInactiveUserByEmail(identifier.UserEmail)
			if err != nil {
				return nil, status.Error(codes.NotFound, "user not found")
			}
		}
		userID = user.ID.String()
	default:
		return nil, status.Error(codes.InvalidArgument, "user identifier must be specified")
	}

	err = authService.RemoveRole(userID, roleStr, req.RoleAssignment.DomainId, domainTypeStr)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to remove role")
	}

	return &pb.RemoveRoleResponse{}, nil
}
