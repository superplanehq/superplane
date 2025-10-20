package organizations

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteOrganization(ctx context.Context, authService authorization.Authorization, orgID string) (*pb.DeleteOrganizationResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	organization, err := models.FindOrganizationByID(orgID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "organization not found")
	}

	//
	// TODO: the organization deletion should be in a transaction
	//

	err = models.SoftDeleteOrganization(organization.ID.String())
	if err != nil {
		log.Errorf("Error deleting organization %s: %v", orgID, err)
		return nil, err
	}

	log.Infof("Organization %s (%s) soft-deleted by user %s", organization.Name, organization.ID.String(), userID)

	err = authService.DestroyOrganizationRoles(organization.ID.String())
	if err != nil {
		log.Errorf("Error deleting organization roles for %s: %v", orgID, err)
		return nil, err
	}

	err = authService.DestroyGlobalCanvasRoles(orgID)
	if err != nil {
		log.Errorf("Error deleting global canvas roles for %s: %v", orgID, err)
		return nil, err
	}

	return &pb.DeleteOrganizationResponse{}, nil
}
