package serviceaccounts

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListServiceAccounts(ctx context.Context) (*pb.ListServiceAccountsResponse, error) {
	_, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	orgID, orgIsSet := authentication.GetOrganizationIdFromMetadata(ctx)
	if !orgIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	users, err := models.FindServiceAccountsByOrganization(orgID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list service accounts")
	}

	serviceAccounts := make([]*pb.ServiceAccount, len(users))
	for i := range users {
		serviceAccounts[i] = serializeServiceAccount(&users[i])
	}

	return &pb.ListServiceAccountsResponse{
		ServiceAccounts: serviceAccounts,
	}, nil
}
