package organizations

import (
	"context"
	"errors"

	uuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateOrganization(ctx context.Context, req *pb.CreateOrganizationRequest) (*pb.CreateOrganizationResponse, error) {
	requesterID, err := uuid.Parse(req.RequesterId)
	if err != nil {
		log.Errorf("Error reading requester id on %v for CreateOrganization: %v", req, err)
		return nil, err
	}

	// Extract name and display name from the Organization metadata
	if req.Organization == nil || req.Organization.Metadata == nil || req.Organization.Metadata.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "organization name is required")
	}

	if req.Organization.Metadata.DisplayName == "" {
		return nil, status.Error(codes.InvalidArgument, "organization display name is required")
	}

	organization, err := models.CreateOrganization(requesterID, req.Organization.Metadata.Name, req.Organization.Metadata.DisplayName)
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		log.Errorf("Error creating organization on %v for CreateOrganization: %v", req, err)
		return nil, err
	}

	// Create response using nested structure
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
