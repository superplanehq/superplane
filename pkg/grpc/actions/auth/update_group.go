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

type UpdateGroupRequest struct {
	DomainID    string
	GroupName   string
	DomainType  pb.DomainType
	Role        string
	DisplayName string
	Description string
}

type UpdateGroupResponse struct {
	Group *pb.Group
}

func UpdateGroup(ctx context.Context, req *UpdateGroupRequest, authService authorization.Authorization) (*UpdateGroupResponse, error) {
	err := actions.ValidateUUIDs(req.DomainID)
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
	currentRole, err := authService.GetGroupRole(req.DomainID, domainType, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.NotFound, "group not found")
	}

	// Update the group role if it changed
	if req.Role != "" && req.Role != currentRole {
		err = authService.UpdateGroupRole(req.DomainID, domainType, req.GroupName, req.Role)
		if err != nil {
			log.Errorf("failed to update group %s role from %s to %s: %v", req.GroupName, currentRole, req.Role, err)
			return nil, status.Error(codes.Internal, "failed to update group role")
		}

		log.Infof("updated group %s role from %s to %s in domain %s (type: %s)", req.GroupName, currentRole, req.Role, req.DomainID, domainType)
	}

	var displayName string
	var description string
	groupMetadata, err := models.FindGroupMetadata(req.GroupName, domainType, req.DomainID)
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

		err = models.UpsertGroupMetadata(req.GroupName, domainType, req.DomainID, displayName, description)
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
		Name:         req.GroupName,
		DomainType:   req.DomainType,
		DomainId:     req.DomainID,
		Role:         updatedRole,
		DisplayName:  displayName,
		Description:  description,
		MembersCount: int32(len(groupUsers)),
		CreatedAt:    groupMetadata.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    groupMetadata.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	return &UpdateGroupResponse{
		Group: group,
	}, nil
}

func ValidateUpdateGroupRequest(req *UpdateGroupRequest) error {
	if req.GroupName == "" {
		return status.Error(codes.InvalidArgument, "group name must be specified")
	}

	if req.DomainType == pb.DomainType_DOMAIN_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "domain type must be specified")
	}

	// At least one field must be provided for update
	if req.Role == "" && req.DisplayName == "" && req.Description == "" {
		return status.Error(codes.InvalidArgument, "at least one field must be provided for update")
	}

	return nil
}
