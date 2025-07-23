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

func CreateGroup(ctx context.Context, req *pb.CreateGroupRequest, authService authorization.Authorization) (*pb.CreateGroupResponse, error) {
	err := actions.ValidateUUIDs(req.DomainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	err = ValidateCreateGroupRequest(req)
	if err != nil {
		return nil, err
	}

	domainType, err := actions.ProtoToDomainType(req.DomainType)
	if err != nil {
		return nil, err
	}

	err = authService.CreateGroup(req.DomainId, domainType, req.GroupName, req.Role)
	if err != nil {
		log.Errorf("failed to create group %s with role %s in domain %s: %v", req.GroupName, req.Role, req.DomainId, err)
		return nil, status.Error(codes.Internal, "failed to create group")
	}

	var displayName string
	var description string
	if req.DisplayName != "" || req.Description != "" {
		displayName = req.DisplayName
		if displayName == "" {
			displayName = req.GroupName
		}
		description = req.Description

		err = models.UpsertGroupMetadata(req.GroupName, domainType, req.DomainId, displayName, description)
		if err != nil {
			log.Errorf("failed to create group metadata for %s: %v", req.GroupName, err)
		}
	}

	log.Infof("created group %s with role %s in domain %s (type: %s)", req.GroupName, req.Role, req.DomainId, req.DomainType.String())

	group := &pb.Group{
		Metadata: &pb.Group_Metadata{
			Name:       req.GroupName,
			DomainType: actions.DomainTypeToProto(domainType),
			DomainId:   req.DomainId,
			CreatedAt:  timestamppb.Now(),
			UpdatedAt:  timestamppb.Now(),
		},
		Spec: &pb.Group_Spec{
			DisplayName: displayName,
			Description: description,
		},
	}

	return &pb.CreateGroupResponse{
		Group: group,
	}, nil
}
