package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteGroup(ctx context.Context, req *pb.DeleteGroupRequest, authService authorization.Authorization) (*pb.DeleteGroupResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	domainType, err := actions.ProtoToDomainType(req.DomainType)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain type")
	}

	groups, err := authService.GetGroups(req.DomainId, domainType)
	if err != nil {
		log.Errorf("failed to list groups in organization %s: %v", req.DomainId, err)
		return nil, status.Error(codes.Internal, "failed to check group existence")
	}

	groupExists := false
	for _, group := range groups {
		if group == req.GroupName {
			groupExists = true
			break
		}
	}

	if !groupExists {
		return nil, status.Error(codes.NotFound, "group not found")
	}

	users, err := authService.GetGroupUsers(req.DomainId, domainType, req.GroupName)
	if err != nil {
		log.Errorf("failed to get users in group %s: %v", req.GroupName, err)
		return nil, status.Error(codes.Internal, "failed to get group users")
	}

	for _, userID := range users {
		err = authService.RemoveUserFromGroup(req.DomainId, domainType, userID, req.GroupName)
		if err != nil {
			log.Errorf("failed to remove user %s from group %s: %v", userID, req.GroupName, err)
			return nil, status.Error(codes.Internal, "failed to remove users from group")
		}
	}

	err = authService.DeleteGroup(req.DomainId, domainType, req.GroupName)
	if err != nil {
		log.Errorf("failed to delete group %s: %v", req.GroupName, err)
		return nil, status.Error(codes.Internal, "failed to delete group")
	}

	err = models.DeleteGroupMetadata(req.GroupName, domainType, req.DomainId)
	if err != nil {
		log.Errorf("failed to delete group metadata for %s: %v", req.GroupName, err)
	}

	log.Infof("deleted group %s from organization %s", req.GroupName, req.DomainId)

	return &pb.DeleteGroupResponse{}, nil
}
