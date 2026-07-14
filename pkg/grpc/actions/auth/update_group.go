package auth

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func UpdateGroup(ctx context.Context, domainType string, domainID string, groupName string, groupSpec *pb.Group_Spec, authService authorization.Authorization) (*pb.UpdateGroupResponse, error) {
	if groupName == "" {
		return nil, grpcerrors.InvalidArgument(nil, "group name must be specified")
	}

	currentRole, err := authService.GetGroupRole(ctx, domainID, domainType, groupName)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "group not found")
	}

	var displayName string
	var description string
	groupModelMetadata, err := models.FindGroupMetadata(groupName, domainType, domainID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to get group metadata")
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

	updatingRole := currentRole
	if groupSpec != nil {
		if groupSpec.Role != "" {
			updatingRole = groupSpec.Role
		}

		err = authService.UpdateGroup(domainID, domainType, groupName, updatingRole, displayName, description)
		if err != nil {
			log.Errorf("failed to update group %s role from %s to %s: %v", groupName, currentRole, groupSpec.Role, err)
			return nil, grpcerrors.Internal(err, "failed to update group role")
		}

		log.Infof("updated group %s role from %s to %s in domain %s (type: %s)", groupName, currentRole, groupSpec.Role, domainID, domainType)
	}

	groupUsers, err := authService.GetGroupUsers(ctx, domainID, domainType, groupName)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to get group members count")
	}

	group := &pb.Group{
		Metadata: &pb.Group_Metadata{
			Name:       groupName,
			DomainType: actions.DomainTypeToProto(domainType),
			DomainId:   domainID,
			CreatedAt:  timestamppb.New(groupModelMetadata.CreatedAt),
			UpdatedAt:  timestamppb.Now(),
		},
		Spec: &pb.Group_Spec{
			Role:        updatingRole,
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
