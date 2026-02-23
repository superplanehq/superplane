package serviceaccounts

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func CreateServiceAccount(ctx context.Context, req *pb.CreateServiceAccountRequest, authService authorization.Authorization) (*pb.CreateServiceAccountResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	validRoles := map[string]bool{
		models.RoleOrgAdmin:  true,
		models.RoleOrgViewer: true,
	}

	if req.Role == "" {
		return nil, status.Error(codes.InvalidArgument, "role is required")
	}

	if !validRoles[req.Role] {
		return nil, status.Error(codes.InvalidArgument, "invalid role for service account; must be org_admin or org_viewer")
	}

	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	createdByUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	var description *string
	if req.Description != "" {
		description = &req.Description
	}

	plainToken, err := crypto.Base64String(64)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	var sa *models.User
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var txErr error
		sa, txErr = models.CreateServiceAccount(tx, orgUUID, req.Name, description, createdByUUID)
		if txErr != nil {
			return txErr
		}

		sa.TokenHash = crypto.HashToken(plainToken)
		sa.UpdatedAt = sa.CreatedAt
		txErr = tx.Save(sa).Error
		if txErr != nil {
			return txErr
		}

		txErr = authService.AssignRole(sa.ID.String(), req.Role, orgID, models.DomainTypeOrganization)
		return txErr
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create service account: %v", err)
	}

	return &pb.CreateServiceAccountResponse{
		ServiceAccount: serializeServiceAccount(sa),
		Token:          plainToken,
	}, nil
}
