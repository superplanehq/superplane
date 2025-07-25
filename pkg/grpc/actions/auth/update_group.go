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

func UpdateGroup(ctx context.Context, domainType string, domainID string, groupName string, groupSpec *pb.Group_Spec, authService authorization.Authorization) (*pb.UpdateGroupResponse, error) {
	if groupName == "" {
		return nil, status.Error(codes.InvalidArgument, "group name must be specified")
	}

	currentRole, err := authService.GetGroupRole(domainID, domainType, groupName)
	if err != nil {
		return nil, status.Error(codes.NotFound, "group not found")
	}

	if groupSpec != nil && groupSpec.Role != "" && groupSpec.Role != currentRole {
		err = authService.UpdateGroup(domainID, domainType, groupName, groupSpec.Role, groupSpec.DisplayName, groupSpec.Description)
		if err != nil {
			log.Errorf("failed to update group %s role from %s to %s: %v", groupName, currentRole, groupSpec.Role, err)
			return nil, status.Error(codes.Internal, "failed to update group role")
		}

		log.Infof("updated group %s role from %s to %s in domain %s (type: %s)", groupName, currentRole, groupSpec.Role, domainID, domainType)
	}

	var displayName string
	var description string
	groupModelMetadata, err := models.FindGroupMetadata(groupName, domainType, domainID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group metadata")
	}

	if groupSpec != nil && (groupSpec.DisplayName != "" || groupSpec.Description != "") {
		displayName = groupSpec.DisplayName
		if displayName == "" {
			displayName = groupModelMetadata.DisplayName
		}

		description = groupSpec.Description
		if description == "" {
			description = groupModelMetadata.Description
		}

	} else {
		displayName = groupModelMetadata.DisplayName
		description = groupModelMetadata.Description
	}

	updatedRole := groupSpec.Role
	if updatedRole == "" {
		updatedRole = currentRole
	}

	groupUsers, err := authService.GetGroupUsers(domainID, domainType, groupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group members count")
	}

	group := &pb.Group{
		Metadata: &pb.Group_Metadata{
			Name:       groupName,
			DomainType: actions.DomainTypeToProto(domainType),
			DomainId:   domainID,
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
