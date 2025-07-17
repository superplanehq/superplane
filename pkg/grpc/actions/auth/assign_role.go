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

func AssignRole(ctx context.Context, req *pb.AssignRoleRequest, authService authorization.Authorization) (*pb.AssignRoleResponse, error) {
	if req.RoleAssignment.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	var domainTypeStr string

	switch req.RoleAssignment.DomainType {
	case pb.DomainType_DOMAIN_TYPE_ORGANIZATION:
		domainTypeStr = authorization.DomainOrg
	case pb.DomainType_DOMAIN_TYPE_CANVAS:
		domainTypeStr = authorization.DomainCanvas
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
	case *pb.AssignRoleRequest_UserId:
		err := actions.ValidateUUIDs(identifier.UserId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user ID")
		}
		userID = identifier.UserId
	case *pb.AssignRoleRequest_UserEmail:
		user, err := models.FindUserByEmail(identifier.UserEmail)
		if err != nil {
			user, err = models.FindInactiveUserByEmail(identifier.UserEmail)
			if err != nil {
				// Create inactive user with email
				user = &models.User{
					Name:     identifier.UserEmail,
					IsActive: false,
				}
				if err := user.Create(); err != nil {
					return nil, status.Error(codes.Internal, "failed to create user")
				}
			}
		}
		userID = user.ID.String()
	default:
		return nil, status.Error(codes.InvalidArgument, "user identifier must be specified")
	}

	err = authService.AssignRole(userID, roleStr, req.RoleAssignment.DomainId, domainTypeStr)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to assign role")
	}

	return &pb.AssignRoleResponse{}, nil
}
