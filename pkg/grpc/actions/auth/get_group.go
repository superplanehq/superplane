package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/authorization"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetGroupRequest struct {
	DomainID   string
	GroupName  string
	DomainType pb.DomainType
}

type GetGroupResponse struct {
	Group *pb.Group
}

func GetGroup(ctx context.Context, req *GetGroupRequest, authService authorization.Authorization) (*GetGroupResponse, error) {
	err := actions.ValidateUUIDs(req.DomainID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	groupReq := &GroupRequest{
		DomainID:   req.DomainID,
		GroupName:  req.GroupName,
		DomainType: req.DomainType,
	}

	err = ValidateGroupRequest(groupReq)
	if err != nil {
		return nil, err
	}

	domainType, err := ConvertDomainType(req.DomainType)
	if err != nil {
		return nil, err
	}

	// Check if the group exists by getting its role
	role, err := authService.GetGroupRole(req.DomainID, domainType, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.NotFound, "group not found")
	}

	groupMetadata, err := models.FindGroupMetadata(req.GroupName, domainType, req.DomainID)
	var displayName, description, createdAt, updatedAt string
	if err == nil {
		displayName = groupMetadata.DisplayName
		description = groupMetadata.Description
		createdAt = groupMetadata.CreatedAt.Format("2006-01-02T15:04:05Z")
		updatedAt = groupMetadata.UpdatedAt.Format("2006-01-02T15:04:05Z")
	} else {
		// Use fallback values when metadata is not found
		displayName = req.GroupName
		description = ""
	}

	membersCount, err := authService.GetGroupMembersCount(req.DomainID, domainType, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group members count")
	}

	group := &pb.Group{
		Name:         req.GroupName,
		DomainType:   req.DomainType,
		DomainId:     req.DomainID,
		Role:         role,
		DisplayName:  displayName,
		Description:  description,
		MembersCount: int32(membersCount),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}

	return &GetGroupResponse{
		Group: group,
	}, nil
}
