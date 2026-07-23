package apikeys

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/api_keys"
	"gorm.io/gorm"
)

func CreateAPIKey(ctx context.Context, req *pb.CreateAPIKeyRequest, authService authorization.Authorization) (*pb.CreateAPIKeyResponse, error) {
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

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, grpcerrors.InvalidArgument(nil, "name is required")
	}

	if req.Role == "" {
		return nil, grpcerrors.InvalidArgument(nil, "role is required")
	}

	// org_owner is reserved for human users and must never be assignable to an
	// API key / service account to prevent privilege escalation.
	if req.Role == models.RoleOrgOwner {
		return nil, grpcerrors.InvalidArgument(nil, "invalid role for API key; org_owner cannot be assigned")
	}

	// Accept any role that exists in this organization (built-in or custom).
	// A lookup error means the role is not valid for this organization.
	if _, err := authService.GetRoleDefinition(ctx, req.Role, models.DomainTypeOrganization, orgID); err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "invalid role for API key; role does not exist in this organization")
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

	db := database.DB(ctx)
	canvasIDs, err := validateAPIKeyCanvasIDs(db, orgID, req.CanvasIds)
	if err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		value := req.ExpiresAt.AsTime()
		if !value.After(time.Now()) {
			return nil, grpcerrors.InvalidArgument(nil, "expiration must be in the future")
		}
		expiresAt = &value
	}

	plainToken, err := crypto.Base64String(64)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to generate token")
	}

	var apiKey *models.User
	err = db.Transaction(func(tx *gorm.DB) error {
		var txErr error
		apiKey, txErr = models.CreateAPIKey(tx, orgUUID, name, description, createdByUUID, expiresAt, canvasIDs)
		if txErr != nil {
			return txErr
		}

		apiKey.TokenHash = crypto.HashToken(plainToken)
		apiKey.UpdatedAt = apiKey.CreatedAt
		txErr = tx.Save(apiKey).Error
		if txErr != nil {
			return txErr
		}

		txErr = authService.AssignRole(apiKey.ID.String(), req.Role, orgID, models.DomainTypeOrganization)
		return txErr
	})

	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create API key")
	}

	creator, err := creatorUserForAPIKey(db, orgID, apiKey)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create API key")
	}

	return &pb.CreateAPIKeyResponse{
		ApiKey: serializeAPIKey(apiKey, creator),
		Token:  plainToken,
	}, nil
}
