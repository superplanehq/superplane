package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteGroup(ctx context.Context, domainType string, domainID string, req *pb.DeleteGroupRequest, authService authorization.Authorization) (*pb.DeleteGroupResponse, error) {
	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	_, err := authService.GetGroupRole(domainID, domainType, req.GroupName)
	if err != nil {
		log.Errorf("failed to get group %s role in domain %s: %v", req.GroupName, domainID, err)
		return nil, status.Error(codes.NotFound, "group not found")
	}

	users, err := authService.GetGroupUsers(domainID, domainType, req.GroupName)
	if err != nil {
		log.Errorf("failed to get users in group %s: %v", req.GroupName, err)
		return nil, status.Error(codes.Internal, "failed to get group users")
	}

	for _, userID := range users {
		err = authService.RemoveUserFromGroup(domainID, domainType, userID, req.GroupName)
		if err != nil {
			log.Errorf("failed to remove user %s from group %s: %v", userID, req.GroupName, err)
			return nil, status.Error(codes.Internal, "failed to remove users from group")
		}
	}

	err = authService.DeleteGroup(domainID, domainType, req.GroupName)
	if err != nil {
		log.Errorf("failed to delete group %s: %v", req.GroupName, err)
		return nil, status.Error(codes.Internal, "failed to delete group")
	}

	err = models.DeleteGroupMetadata(req.GroupName, domainType, domainID)
	if err != nil {
		log.Errorf("failed to delete group metadata for %s: %v", req.GroupName, err)
		return nil, status.Error(codes.Internal, "failed to delete group metadata")
	}

	log.Infof("deleted group %s from domain %s", req.GroupName, domainID)

	return &pb.DeleteGroupResponse{}, nil
}
