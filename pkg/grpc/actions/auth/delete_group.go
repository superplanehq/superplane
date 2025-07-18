package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteOrganizationGroup(ctx context.Context, req *pb.DeleteOrganizationGroupRequest, authService authorization.Authorization) (*pb.DeleteOrganizationGroupResponse, error) {
	err := actions.ValidateUUIDs(req.OrganizationId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	domainType := authorization.DomainOrg

	groups, err := authService.GetGroups(req.OrganizationId, domainType)
	if err != nil {
		log.Errorf("failed to list groups in organization %s: %v", req.OrganizationId, err)
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

	users, err := authService.GetGroupUsers(req.OrganizationId, domainType, req.GroupName)
	if err != nil {
		log.Errorf("failed to get users in group %s: %v", req.GroupName, err)
		return nil, status.Error(codes.Internal, "failed to get group users")
	}

	for _, userID := range users {
		err = authService.RemoveUserFromGroup(req.OrganizationId, domainType, userID, req.GroupName)
		if err != nil {
			log.Errorf("failed to remove user %s from group %s: %v", userID, req.GroupName, err)
			return nil, status.Error(codes.Internal, "failed to remove users from group")
		}
	}

	err = authService.DeleteGroup(req.OrganizationId, domainType, req.GroupName)
	if err != nil {
		log.Errorf("failed to delete group %s: %v", req.GroupName, err)
		return nil, status.Error(codes.Internal, "failed to delete group")
	}

	err = models.DeleteGroupMetadata(req.GroupName, domainType, req.OrganizationId)
	if err != nil {
		log.Errorf("failed to delete group metadata for %s: %v", req.GroupName, err)
	}

	log.Infof("deleted group %s from organization %s", req.GroupName, req.OrganizationId)

	return &pb.DeleteOrganizationGroupResponse{}, nil
}

func DeleteCanvasGroup(ctx context.Context, req *pb.DeleteCanvasGroupRequest, authService authorization.Authorization) (*pb.DeleteCanvasGroupResponse, error) {
	canvasID, err := ConvertCanvasIdOrNameToId(req.CanvasIdOrName)
	if err != nil {
		return nil, err
	}

	err = actions.ValidateUUIDs(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	if req.GroupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	domainType := authorization.DomainCanvas

	groups, err := authService.GetGroups(canvasID, domainType)
	if err != nil {
		log.Errorf("failed to list groups in canvas %s: %v", canvasID, err)
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

	users, err := authService.GetGroupUsers(canvasID, domainType, req.GroupName)
	if err != nil {
		log.Errorf("failed to get users in group %s: %v", req.GroupName, err)
		return nil, status.Error(codes.Internal, "failed to get group users")
	}

	for _, userID := range users {
		err = authService.RemoveUserFromGroup(canvasID, domainType, userID, req.GroupName)
		if err != nil {
			log.Errorf("failed to remove user %s from group %s: %v", userID, req.GroupName, err)
			return nil, status.Error(codes.Internal, "failed to remove users from group")
		}
	}

	err = authService.DeleteGroup(canvasID, domainType, req.GroupName)
	if err != nil {
		log.Errorf("failed to delete group %s: %v", req.GroupName, err)
		return nil, status.Error(codes.Internal, "failed to delete group")
	}

	err = models.DeleteGroupMetadata(req.GroupName, domainType, canvasID)
	if err != nil {
		log.Errorf("failed to delete group metadata for %s: %v", req.GroupName, err)
	}

	log.Infof("deleted group %s from canvas %s", req.GroupName, canvasID)

	return &pb.DeleteCanvasGroupResponse{}, nil
}
