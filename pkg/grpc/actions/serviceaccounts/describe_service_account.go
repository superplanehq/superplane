package serviceaccounts

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DescribeServiceAccount(ctx context.Context, req *pb.DescribeServiceAccountRequest) (*pb.DescribeServiceAccountResponse, error) {
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

	var creator *models.User
	if user.CreatedBy != nil {
		creators, creatorsErr := models.FindMaybeDeletedUsersByIDs([]uuid.UUID{*user.CreatedBy})
		if creatorsErr != nil {
			return nil, status.Error(codes.Internal, "failed to load service account creator")
		}
		if len(creators) > 0 {
			creator = &creators[0]
		}
	}

	return &pb.DescribeServiceAccountResponse{
		ServiceAccount: serializeServiceAccount(user, creator),
	}, nil
}
