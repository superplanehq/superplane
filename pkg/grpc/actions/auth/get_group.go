package auth

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/groups"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func GetGroup(ctx context.Context, domainType string, domainID string, req *pb.GetGroupRequest, authService authorization.Authorization) (*pb.GetGroupResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid domain ID")
	}

	groupReq := &GroupRequest{
		DomainID:   req.DomainId,
		GroupName:  req.GroupName,
		DomainType: req.DomainType,
	}

	err = ValidateGroupRequest(groupReq)
	if err != nil {
		return nil, err
	}


	// Check if the group exists by getting its role
	role, err := authService.GetGroupRole(req.DomainId, domainType, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.NotFound, "group not found")
	}

	groupMetadata, err := models.FindGroupMetadata(req.GroupName, domainType, req.DomainId)
	var displayName, description string
	var createdAt, updatedAt *timestamppb.Timestamp
	if err == nil {
		displayName = groupMetadata.DisplayName
		description = groupMetadata.Description
		createdAt = timestamppb.New(groupMetadata.CreatedAt)
		updatedAt = timestamppb.New(groupMetadata.UpdatedAt)
	} else {
		// Use fallback values when metadata is not found
		displayName = req.GroupName
		description = ""
	}

	groupUsers, err := authService.GetGroupUsers(req.DomainId, domainType, req.GroupName)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get group members count")
	}

	group := &pb.Group{
		Metadata: &pb.Group_Metadata{
			Name:       req.GroupName,
			DomainType: req.DomainType,
			DomainId:   req.DomainId,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
		},
		Spec: &pb.Group_Spec{
			Description: description,
			Role:        role,
			DisplayName: displayName,
		},
		Status: &pb.Group_Status{
			MembersCount: int32(len(groupUsers)),
		},
	}

	return &pb.GetGroupResponse{
		Group: group,
	}, nil
}
