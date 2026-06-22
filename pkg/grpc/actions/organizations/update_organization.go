package organizations

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func UpdateOrganization(ctx context.Context, orgID string, pbOrganization *pb.Organization) (*pb.UpdateOrganizationResponse, error) {
	if pbOrganization == nil {
		return nil, status.Error(codes.InvalidArgument, "organization is required")
	}

	if pbOrganization.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "organization metadata is required")
	}

	organization, err := models.FindOrganizationByID(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "organization not found")
		}

		log.WithError(err).
			WithField("organization_id", orgID).
			Error("failed to load organization for update")
		return nil, status.Error(codes.Internal, "failed to update organization")
	}

	if pbOrganization.Metadata.Name != "" {
		organization.Name = pbOrganization.Metadata.Name
	}

	if pbOrganization.Metadata.Description != "" {
		organization.Description = pbOrganization.Metadata.Description
	}

	now := time.Now()
	organization.UpdatedAt = &now
	err = database.Conn().Save(organization).Error
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		log.WithError(err).
			WithField("organization_id", orgID).
			Error("failed to save organization update")
		return nil, status.Error(codes.Internal, "failed to update organization")
	}

	response := &pb.UpdateOrganizationResponse{
		Organization: &pb.Organization{
			Metadata: &pb.Organization_Metadata{
				Id:          organization.ID.String(),
				Name:        organization.Name,
				Description: organization.Description,
				CreatedAt:   protoTime(organization.CreatedAt),
				UpdatedAt:   protoTime(organization.UpdatedAt),
			},
			Spec: &pb.Organization_Spec{
				EnabledExperimentalFeatures: []string(organization.EnabledExperimentalFeatures),
			},
		},
	}

	return response, nil
}
