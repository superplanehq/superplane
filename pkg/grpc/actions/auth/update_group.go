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
	"google.golang.org/protobuf/types/known/timestamppb"
)

func UpdateGroup(ctx context.Context, req *pb.UpdateGroupRequest, authService authorization.Authorization) (*pb.UpdateGroupResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	err = ValidateUpdateGroupRequest(req)
	if err != nil {
		return nil, err
	}

	domainType, err := actions.ProtoToDomainType(req.DomainType)
	if err != nil {
		return nil, err
	}

	// Check if the group exists by getting its current role
	currentRole, err := authService.GetGroupRole(req.DomainId, domainType, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.NotFound, "group not found")
	}

	// Update the group role if it changed
	if req.Role != "" && req.Role != currentRole {
		err = authService.UpdateGroupRole(req.DomainId, domainType, req.GroupName, req.Role)
		if err != nil {
			log.Errorf("failed to update group %s role from %s to %s: %v", req.GroupName, currentRole, req.Role, err)
			return nil, status.Error(codes.Internal, "failed to update group role")
		}

		log.Infof("updated group %s role from %s to %s in domain %s (type: %s)", req.GroupName, currentRole, req.Role, req.DomainId, domainType)
	}

	var displayName string
	var description string
	groupMetadata, err := models.FindGroupMetadata(req.GroupName, domainType, req.DomainId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group metadata")
	}

	if req.DisplayName != "" || req.Description != "" {
		displayName = req.DisplayName
		if displayName == "" {
			displayName = groupMetadata.DisplayName
		}

		description = req.Description
		if description == "" {
			description = groupMetadata.Description
		}

		err = models.UpsertGroupMetadata(req.GroupName, domainType, req.DomainId, displayName, description)
		if err != nil {
			log.Errorf("failed to update group metadata for %s: %v", req.GroupName, err)
			// Don't fail the entire operation for metadata errors
		}
	}

	// Get the updated role (in case it changed)
	updatedRole := req.Role
	if updatedRole == "" {
		updatedRole = currentRole
	}

	groupUsers, err := authService.GetGroupUsers(groupMetadata.DomainID, domainType, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group members count")
	}

	group := &pb.Group{
		Metadata: &pb.Group_Metadata{
			Name:       req.GroupName,
			DomainType: req.DomainType,
			DomainId:   req.DomainId,
			CreatedAt:  timestamppb.Now(),
			UpdatedAt:  timestamppb.Now(),
		},
		Spec: &pb.Group_Spec{
			Role:        updatedRole,
			DisplayName: displayName,
			Description: description,
		},
		Status: &pb.Group_Status{
			MembersCount: int32(len(groupUsers)),
		},
	}

	return &pb.UpdateGroupResponse{
		Group: group,
	}, nil
}
