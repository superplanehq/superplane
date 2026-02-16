package serviceaccounts

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RegenerateServiceAccountToken(ctx context.Context, req *pb.RegenerateServiceAccountTokenRequest) (*pb.RegenerateServiceAccountTokenResponse, error) {
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

	plainToken, err := crypto.Base64String(64)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate new token")
	}

	err = user.UpdateTokenHash(crypto.HashToken(plainToken))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update token")
	}

	return &pb.RegenerateServiceAccountTokenResponse{
		Token: plainToken,
	}, nil
}
