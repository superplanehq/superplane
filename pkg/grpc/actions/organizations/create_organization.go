package organizations

import (
	"context"
	"errors"

	uuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateOrganization(ctx context.Context, req *pb.CreateOrganizationRequest, authorizationService authorization.Authorization) (*pb.CreateOrganizationResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)

	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	userIDUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	if req.Organization == nil || req.Organization.Metadata == nil || req.Organization.Metadata.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "organization name is required")
	}

	if req.Organization.Metadata.DisplayName == "" {
		return nil, status.Error(codes.InvalidArgument, "organization display name is required")
	}

	organization, err := models.CreateOrganization(userIDUUID, req.Organization.Metadata.Name, req.Organization.Metadata.DisplayName)
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		log.Errorf("Error creating organization on %v for CreateOrganization: %v", req, err)
		return nil, err
	}

	err = authorizationService.SetupOrganizationRoles(organization.ID.String())
	if err != nil {
		log.Errorf("Error setting up organization roles for %v for CreateOrganization: %v", req, err)
		return nil, status.Error(codes.Internal, "error setting up organization roles")
	}
	log.Infof("Set all roles for organization %s (%s)", organization.Name, organization.ID.String())

	err = authorizationService.CreateOrganizationOwner(userID, organization.ID.String())
	if err != nil {
		log.Errorf("Error creating organization owner for %v for CreateOrganization: %v", req, err)
		return nil, status.Error(codes.Internal, "error creating organization owner")
	}
	log.Infof("Created organization owner for %s (%s) for user %s", organization.Name, organization.ID.String(), userID)

	response := &pb.CreateOrganizationResponse{
		Organization: &pb.Organization{
			Metadata: &pb.Organization_Metadata{
				Id:          organization.ID.String(),
				Name:        organization.Name,
				DisplayName: organization.DisplayName,
				CreatedBy:   organization.CreatedBy.String(),
				CreatedAt:   timestamppb.New(*organization.CreatedAt),
				UpdatedAt:   timestamppb.New(*organization.UpdatedAt),
			},
		},
	}

	return response, nil
}
