package organizations

import (
	"context"
	"errors"

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
	user, userIsSet := authentication.GetUserFromContext(ctx)

	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if req.Organization == nil || req.Organization.Metadata == nil || req.Organization.Metadata.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "organization name is required")
	}

	if req.Organization.Metadata.DisplayName == "" {
		return nil, status.Error(codes.InvalidArgument, "organization display name is required")
	}

	organization, err := models.CreateOrganization(user.ID, req.Organization.Metadata.Name, req.Organization.Metadata.DisplayName)
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		log.Errorf("Error creating organization on %v for CreateOrganization: %v", req, err)
		return nil, err
	}

	authorizationService.SetupOrganizationRoles(organization.ID.String())
	authorizationService.CreateOrganizationOwner(organization.ID.String(), user.ID.String())

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
