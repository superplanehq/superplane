package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteGroup(ctx context.Context, domainType, domainID, groupName string, authService authorization.Authorization) (*pb.DeleteGroupResponse, error) {
	if groupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	_, err := authService.GetGroupRole(domainID, domainType, groupName)
	if err != nil {
		log.Errorf("failed to get group %s role in domain %s: %v", groupName, domainID, err)
		return nil, status.Error(codes.NotFound, "group not found")
	}

	users, err := authService.GetGroupUsers(domainID, domainType, groupName)
	if err != nil {
		log.Errorf("failed to get users in group %s: %v", groupName, err)
		return nil, status.Error(codes.Internal, "failed to get group users")
	}

	for _, userID := range users {
		err = authService.RemoveUserFromGroup(domainID, domainType, userID, groupName)
		if err != nil {
			log.Errorf("failed to remove user %s from group %s: %v", userID, groupName, err)
			return nil, status.Error(codes.Internal, "failed to remove users from group")
		}
	}

	err = authService.DeleteGroup(domainID, domainType, groupName)
	if err != nil {
		log.Errorf("failed to delete group %s: %v", groupName, err)
		return nil, status.Error(codes.Internal, "failed to delete group")
	}

	log.Infof("deleted group %s from domain %s", groupName, domainID)

	return &pb.DeleteGroupResponse{}, nil
}
