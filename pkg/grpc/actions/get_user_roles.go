package actions

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetUserRoles(ctx context.Context, req *pb.GetUserRolesRequest, authService authorization.AuthorizationServiceInterface) (*pb.GetUserRolesResponse, error) {
	err := ValidateUUIDs(req.UserId, req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	var roleStrings []string
	switch req.DomainType {
	case pb.DomainType_DOMAIN_TYPE_ORGANIZATION:
		roleStrings, err = authService.GetUserRolesForOrg(req.UserId, req.DomainId)
	case pb.DomainType_DOMAIN_TYPE_CANVAS:
		roleStrings, err = authService.GetUserRolesForCanvas(req.UserId, req.DomainId)
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported domain type")
	}

	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user roles")
	}

	var roles []*pb.UserRole
	for _, roleStr := range roleStrings {
		userRole := &pb.UserRole{}

		switch req.DomainType {
		case pb.DomainType_DOMAIN_TYPE_ORGANIZATION:
			orgRole := convertStringToOrgRole(roleStr)
			if orgRole != pb.OrganizationRole_ORG_ROLE_UNSPECIFIED {
				userRole.Role = &pb.UserRole_OrgRole{OrgRole: orgRole}
				roles = append(roles, userRole)
			}
		case pb.DomainType_DOMAIN_TYPE_CANVAS:
			canvasRole := convertStringToCanvasRole(roleStr)
			if canvasRole != pb.CanvasRole_CANVAS_ROLE_UNSPECIFIED {
				userRole.Role = &pb.UserRole_CanvasRole{CanvasRole: canvasRole}
				roles = append(roles, userRole)
			}
		}
	}

	return &pb.GetUserRolesResponse{
		UserId:     req.UserId,
		DomainType: req.DomainType,
		DomainId:   req.DomainId,
		Roles:      roles,
	}, nil
}
