package organizations

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func UpdateOrganization(ctx context.Context, orgID string, pbOrganization *pb.Organization) (*pb.UpdateOrganizationResponse, error) {
	if pbOrganization == nil {
		return nil, grpcerrors.InvalidArgument(nil, "organization is required")
	}

	if pbOrganization.Metadata == nil {
		return nil, grpcerrors.InvalidArgument(nil, "organization metadata is required")
	}

	organization, err := models.FindOrganizationByID(orgID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "organization not found")
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
			return nil, grpcerrors.InvalidArgument(err, "invalid organization update")
		}

		log.Errorf("Error updating organization %s: %v", orgID, err)
		return nil, err
	}

	response := &pb.UpdateOrganizationResponse{
		Organization: &pb.Organization{
			Metadata: &pb.Organization_Metadata{
				Id:          organization.ID.String(),
				Name:        organization.Name,
				Description: organization.Description,
				CreatedAt:   timestamppb.New(*organization.CreatedAt),
				UpdatedAt:   timestamppb.New(*organization.UpdatedAt),
			},
			Spec: &pb.Organization_Spec{
				EnabledExperimentalFeatures: []string(organization.EnabledExperimentalFeatures),
			},
		},
	}

	return response, nil
}
