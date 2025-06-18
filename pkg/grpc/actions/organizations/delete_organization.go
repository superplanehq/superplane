package organizations

import (
	"context"
	"errors"

	uuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DeleteOrganization(ctx context.Context, req *pb.DeleteOrganizationRequest) (*pb.DeleteOrganizationResponse, error) {
	requesterID, err := uuid.Parse(req.RequesterId)
	if err != nil {
		log.Errorf("Error reading requester id on %v for DeleteOrganization: %v", req, err)
		return nil, err
	}

	if req.IdOrName == "" {
		return nil, status.Error(codes.InvalidArgument, "id_or_name is required")
	}

	var organization *models.Organization
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

		log.Errorf("Error finding organization for deletion. Request: %v. Error: %v", req, err)
		return nil, err
	}

	err = database.Conn().Delete(organization).Error
	if err != nil {
		log.Errorf("Error deleting organization on %v for DeleteOrganization: %v", req, err)
		return nil, err
	}

	log.Infof("Organization %s (%s) deleted by user %s", organization.Name, organization.ID.String(), requesterID.String())

	return &pb.DeleteOrganizationResponse{}, nil
}
