package organizations

import (
	"context"
	"errors"

	uuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DeleteOrganization(ctx context.Context, req *pb.DeleteOrganizationRequest, authorizationService authorization.Authorization) (*pb.DeleteOrganizationResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if req.IdOrName == "" {
		return nil, status.Error(codes.InvalidArgument, "id_or_name is required")
	}

	var err error
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

	err = models.SoftDeleteOrganization(organization.ID.String())
	if err != nil {
		log.Errorf("Error deleting organization on %v for DeleteOrganization: %v", req, err)
		return nil, err
	}

	log.Infof("Organization %s (%s) soft-deleted by user %s", organization.Name, organization.ID.String(), userID)

	err = authorizationService.DestroyOrganizationRoles(organization.ID.String())
	if err != nil {
		log.Errorf("Error deleting organization roles on %v for DeleteOrganization: %v", req, err)
		return nil, err
	}

	return &pb.DeleteOrganizationResponse{}, nil
}
