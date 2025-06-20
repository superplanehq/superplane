package organizations

import (
	"context"
	"errors"
	"time"

	uuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func UpdateOrganization(ctx context.Context, req *pb.UpdateOrganizationRequest) (*pb.UpdateOrganizationResponse, error) {
	if req.IdOrName == "" {
		return nil, status.Error(codes.InvalidArgument, "id_or_name is required")
	}

	if req.Organization == nil {
		return nil, status.Error(codes.InvalidArgument, "organization is required")
	}

	if req.Organization.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "organization metadata is required")
	}

	var organization *models.Organization
	var err error
	if _, parseErr := uuid.Parse(req.IdOrName); parseErr == nil {

		err = actions.ValidateUUIDs(req.IdOrName)
		if err != nil {
			return nil, err
		}
		organization, err = models.FindOrganizationByID(req.IdOrName)
	} else {
		organization, err = models.FindOrganizationByName(req.IdOrName)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "organization not found")
		}

		log.Errorf("Error finding organization for update. Request: %v. Error: %v", req, err)
		return nil, err
	}

	now := time.Now()
	if req.Organization.Metadata.Name != "" {
		organization.Name = req.Organization.Metadata.Name
	}
	if req.Organization.Metadata.DisplayName != "" {
		organization.DisplayName = req.Organization.Metadata.DisplayName
	}
	organization.UpdatedAt = &now

	err = database.Conn().Save(organization).Error
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		log.Errorf("Error updating organization on %v for UpdateOrganization: %v", req, err)
		return nil, err
	}

	response := &pb.UpdateOrganizationResponse{
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
