package actions

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CreateGroup(ctx context.Context, req *pb.CreateGroupRequest, authService authorization.AuthorizationServiceInterface) (*pb.CreateGroupResponse, error) {
	err := ValidateUUIDs(req.OrgId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	if req.Role == pb.OrganizationRole_ORG_ROLE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "role must be specified")
	}

	roleStr := convertOrgRoleToString(req.Role)
	if roleStr == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid role")
	}

	err = authService.CreateGroup(req.OrgId, req.GroupName, roleStr)
	if err != nil {
		log.Errorf("failed to create group %s with role %s: %v", req.GroupName, roleStr, err)
		return nil, status.Error(codes.Internal, "failed to create group")
	}

	log.Infof("created group %s with role %s in organization %s", req.GroupName, roleStr, req.OrgId)

	return &pb.CreateGroupResponse{}, nil
}
