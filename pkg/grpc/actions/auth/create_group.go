package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CreateGroup(ctx context.Context, req *pb.CreateGroupRequest, authService authorization.Authorization) (*pb.CreateGroupResponse, error) {
	err := actions.ValidateUUIDs(req.OrgId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	if req.Role == "" {
		return nil, status.Error(codes.InvalidArgument, "role must be specified")
	}

	// TODO: once orgs are implemented, check if the org exists

	err = authService.CreateGroup(req.OrgId, req.GroupName, req.Role)
	if err != nil {
		log.Errorf("failed to create group %s with role %s: %v", req.GroupName, req.Role, err)
		return nil, status.Error(codes.Internal, "failed to create group")
	}

	log.Infof("created group %s with role %s in organization %s", req.GroupName, req.Role, req.OrgId)

	return &pb.CreateGroupResponse{}, nil
}
