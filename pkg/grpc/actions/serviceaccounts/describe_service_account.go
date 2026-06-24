package serviceaccounts

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
)

func DescribeServiceAccount(ctx context.Context, req *pb.DescribeServiceAccountRequest) (*pb.DescribeServiceAccountResponse, error) {
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
		return nil, grpcerrors.NotFound(err, "service account not found")
	}

	if !user.IsServiceAccount() {
		return nil, grpcerrors.NotFound(err, "service account not found")
	}

	creator, err := creatorUserForServiceAccount(orgID, user)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe service account")
	}

	return &pb.DescribeServiceAccountResponse{
		ServiceAccount: serializeServiceAccount(user, creator),
	}, nil
}
