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

func CreateGroup(ctx context.Context, req *CreateGroupRequest, authService authorization.Authorization) (*CreateGroupResponse, error) {
	err := actions.ValidateUUIDs(req.DomainID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid UUIDs")
	}

	err = ValidateCreateGroupRequest(req)
	if err != nil {
		return nil, err
	}

	domainType, err := ConvertDomainType(req.DomainType)
	if err != nil {
		return nil, err
	}

	err = authService.CreateGroup(req.DomainID, domainType, req.GroupName, req.Role)
	if err != nil {
		log.Errorf("failed to create group %s with role %s in domain %s: %v", req.GroupName, req.Role, req.DomainID, err)
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

		err = models.UpsertGroupMetadata(req.GroupName, domainType, req.DomainID, displayName, description)
		if err != nil {
			log.Errorf("failed to create group metadata for %s: %v", req.GroupName, err)
		}
	}

	log.Infof("created group %s with role %s in domain %s (type: %s)", req.GroupName, req.Role, req.DomainID, req.DomainType.String())

	group := &pb.Group{
		Name:        req.GroupName,
		DomainType:  req.DomainType,
		DomainId:    req.DomainID,
		Role:        req.Role,
		DisplayName: displayName,
		Description: description,
	}

	return &CreateGroupResponse{
		Group: group,
	}, nil
}
