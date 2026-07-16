package serviceaccounts

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"gorm.io/gorm"
)

func CreateServiceAccount(ctx context.Context, req *pb.CreateServiceAccountRequest, authService authorization.Authorization) (*pb.CreateServiceAccountResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	if req.Name == "" {
		return nil, grpcerrors.InvalidArgument(nil, "name is required")
	}

	if req.Role == "" {
		return nil, grpcerrors.InvalidArgument(nil, "role is required")
	}

	// Validate the role against the organization's roles (default and custom)
	// instead of a hardcoded allow-list, so custom roles can be assigned to
	// service accounts.
	if _, err := authService.GetRoleDefinition(ctx, req.Role, models.DomainTypeOrganization, orgID); err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid role for service account")
	}

	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid organization ID")
	}

	createdByUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid user ID")
	}

	var description *string
	if req.Description != "" {
		description = &req.Description
	}

	plainToken, err := crypto.Base64String(64)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to generate token")
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
		return nil, grpcerrors.Internal(err, "failed to create service account")
	}

	db := database.DB(ctx)
	creator, err := creatorUserForServiceAccount(db, orgID, sa)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create service account")
	}

	return &pb.CreateServiceAccountResponse{
		ServiceAccount: serializeServiceAccount(sa, creator),
		Token:          plainToken,
	}, nil
}
