package serviceaccounts

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"gorm.io/datatypes"
)

func UpdateServiceAccount(ctx context.Context, req *pb.UpdateServiceAccountRequest) (*pb.UpdateServiceAccountResponse, error) {
	_, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	if req.Id == "" {
		return nil, grpcerrors.InvalidArgument(nil, "id is required")
	}

	user, err := models.FindActiveUserByID(orgID, req.Id)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "API key not found")
	}

	if !user.IsServiceAccount() {
		return nil, grpcerrors.NotFound(err, "API key not found")
	}

	if req.Name != "" {
		user.Name = req.Name
	}

	if req.Description != "" {
		user.Description = &req.Description
	}

	db := database.DB(ctx)
	if req.CanvasIds != nil {
		canvasIDs, err := validateServiceAccountCanvasIDs(db, orgID, req.CanvasIds)
		if err != nil {
			return nil, err
		}
		user.ServiceAccountCanvasIDs = datatypes.NewJSONSlice(canvasIDs)
	}

	if req.ClearExpiresAt {
		user.ServiceAccountExpiresAt = nil
	} else if req.ExpiresAt != nil {
		expiresAt := req.ExpiresAt.AsTime()
		if !expiresAt.After(time.Now()) {
			return nil, grpcerrors.InvalidArgument(nil, "expiration must be in the future")
		}
		user.ServiceAccountExpiresAt = &expiresAt
	}

	user.UpdatedAt = time.Now()
	err = db.Save(user).Error
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to update API key")
	}

	creator, err := creatorUserForServiceAccount(db, orgID, user)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to update API key")
	}

	return &pb.UpdateServiceAccountResponse{
		ServiceAccount: serializeServiceAccount(user, creator),
	}, nil
}
