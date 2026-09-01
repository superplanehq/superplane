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
	"gorm.io/gorm"
)

func UpdateOrganization(ctx context.Context, orgID string, pbOrganization *pb.Organization) (*pb.UpdateOrganizationResponse, error) {
	if pbOrganization == nil {
		return nil, grpcerrors.InvalidArgument(nil, "organization is required")
	}

	if pbOrganization.Metadata == nil {
		return nil, grpcerrors.InvalidArgument(nil, "organization metadata is required")
	}

	tx := database.Conn()
	organization, err := models.FindOrganizationByIDOrSlug(tx, orgID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "organization not found")
	}

	if pbOrganization.Metadata.Name != "" {
		organization.Name = pbOrganization.Metadata.Name
	}

	if pbOrganization.Metadata.Description != "" {
		organization.Description = pbOrganization.Metadata.Description
	}

	if err := applyOrganizationSlugUpdate(tx, organization, pbOrganization.Metadata.Slug); err != nil {
		return nil, err
	}

	now := time.Now()
	organization.UpdatedAt = &now
	err = tx.Save(organization).Error
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
				Slug:        organization.Slug,
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

// applyOrganizationSlugUpdate validates and applies a requested slug change.
// An empty requestedSlug leaves an existing slug untouched, but an
// organization created before slugs existed still gets one generated from
// its name. A non-empty requestedSlug must already be URL-friendly (the
// caller controls the exact value, so we reject invalid input instead of
// silently reshaping it) and must not be reserved or already taken by
// another active organization.
func applyOrganizationSlugUpdate(tx *gorm.DB, organization *models.Organization, requestedSlug string) error {
	if requestedSlug == "" {
		if organization.Slug == "" {
			slug, err := models.GenerateUniqueOrganizationSlug(tx, organization.Name, organization.ID)
			if err != nil {
				return grpcerrors.Internal(err, "failed to generate organization slug")
			}
			organization.Slug = slug
		}
		return nil
	}

	if requestedSlug == organization.Slug {
		return nil
	}

	if requestedSlug != models.Slugify(requestedSlug) {
		return grpcerrors.InvalidArgument(nil, "organization slug must use lowercase letters, numbers, and dashes only")
	}

	if models.IsReservedOrganizationSlug(requestedSlug) {
		return grpcerrors.InvalidArgument(nil, "organization slug is reserved")
	}

	available, err := models.IsOrganizationSlugAvailable(tx, requestedSlug, organization.ID)
	if err != nil {
		return grpcerrors.Internal(err, "failed to validate organization slug")
	}
	if !available {
		return grpcerrors.InvalidArgument(models.ErrSlugAlreadyUsed, "organization slug is already in use")
	}

	organization.Slug = requestedSlug
	return nil
}
