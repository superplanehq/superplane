package serviceaccounts

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
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

	db := database.DB(ctx)
	user, err := models.FindActiveUserByIDInTransaction(db, orgID, req.Id)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "service account not found")
	}

	if !user.IsServiceAccount() {
		return nil, grpcerrors.NotFound(err, "service account not found")
	}

	if req.Name != "" {
		user.Name = req.Name
	}

	if req.Description != "" {
		user.Description = &req.Description
	}

	user.UpdatedAt = time.Now()
	err = db.Save(user).Error
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to update service account")
	}

	creator, err := creatorUserForServiceAccount(db, orgID, user)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to update service account")
	}

	return &pb.UpdateServiceAccountResponse{
		ServiceAccount: serializeServiceAccount(user, creator),
	}, nil
}
