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
	"gorm.io/datatypes"
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
		return nil, status.Error(codes.NotFound, "organization not found")
	}

	if pbOrganization.Metadata.Name != "" {
		organization.Name = pbOrganization.Metadata.Name
	}

	if pbOrganization.Metadata.Description != "" {
		organization.Description = pbOrganization.Metadata.Description
	}

	if pbOrganization.Spec != nil && pbOrganization.Spec.ChangeManagementEnabled != nil {
		organization.ChangeManagementEnabled = *pbOrganization.Spec.ChangeManagementEnabled
	}

	if pbOrganization.Spec != nil && pbOrganization.Spec.AllowedOauthProviders != nil {
		list := pbOrganization.Spec.AllowedOauthProviders.GetProviders()
		list = models.NormalizeAllowedOAuthProviders(list)
		if err := models.ValidateAllowedOAuthProviders(list); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		organization.AllowedProviders = datatypes.JSONSlice[string](list)
	}

	if pbOrganization.Spec != nil && pbOrganization.Spec.AllowDirectEmailInviteCompletion != nil {
		organization.AllowDirectEmailInviteCompletion = *pbOrganization.Spec.AllowDirectEmailInviteCompletion
	}

	now := time.Now()
	organization.UpdatedAt = &now
	err = database.Conn().Save(organization).Error
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		log.Errorf("Error updating organization %s: %v", orgID, err)
		return nil, err
	}

	response := &pb.UpdateOrganizationResponse{
		Organization: organizationToProto(organization),
	}

	return response, nil
}
