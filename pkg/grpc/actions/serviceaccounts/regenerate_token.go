package serviceaccounts

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
)

func RegenerateServiceAccountToken(ctx context.Context, req *pb.RegenerateServiceAccountTokenRequest) (*pb.RegenerateServiceAccountTokenResponse, error) {
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

	plainToken, err := crypto.Base64String(64)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to generate new token")
	}

	user.UpdatedAt = time.Now()
	user.TokenHash = crypto.HashToken(plainToken)
	err = db.Save(user).Error
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to update token")
	}

	return &pb.RegenerateServiceAccountTokenResponse{
		Token: plainToken,
	}, nil
}
