package serviceaccounts

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteServiceAccount(ctx context.Context, req *pb.DeleteServiceAccountRequest, authService authorization.Authorization) (*pb.DeleteServiceAccountResponse, error) {
	_, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	user, err := models.FindActiveUserByID(orgID, req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "service account not found")
	}

	if !user.IsServiceAccount() {
		return nil, status.Error(codes.NotFound, "service account not found")
	}

	// Remove all RBAC roles before deleting
	roles, err := authService.GetUserRolesForOrg(user.ID.String(), orgID)
	if err != nil {
		log.Errorf("Error determining roles for service account %s: %v", user.ID, err)
	} else {
		for _, role := range roles {
			err = authService.RemoveRole(user.ID.String(), role.Name, orgID, models.DomainTypeOrganization)
			if err != nil {
				log.Errorf("Error removing role %s for service account %s: %v", role.Name, user.ID, err)
			}
		}
	}

	err = user.Delete()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to delete service account")
	}

	return &pb.DeleteServiceAccountResponse{}, nil
}
