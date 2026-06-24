package serviceaccounts

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
)

func ListServiceAccounts(ctx context.Context) (*pb.ListServiceAccountsResponse, error) {
	_, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	users, err := models.FindServiceAccountsByOrganization(orgID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list service accounts")
	}

	creatorsByID, err := creatorsByIDForServiceAccounts(orgID, users)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to list service accounts")
	}

	serviceAccounts := make([]*pb.ServiceAccount, len(users))
	for i := range users {
		var creator *models.User
		if users[i].CreatedBy != nil {
			creator = creatorsByID[users[i].CreatedBy.String()]
		}
		serviceAccounts[i] = serializeServiceAccount(&users[i], creator)
	}

	return &pb.ListServiceAccountsResponse{
		ServiceAccounts: serviceAccounts,
	}, nil
}
