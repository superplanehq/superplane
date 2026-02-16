package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	service_accounts "github.com/superplanehq/superplane/pkg/grpc/actions/service_accounts"
	pb "github.com/superplanehq/superplane/pkg/protos/service_accounts"
)

type ServiceAccountsService struct {
	pb.UnimplementedServiceAccountsServer
	authService authorization.Authorization
}

func NewServiceAccountsService(authService authorization.Authorization) *ServiceAccountsService {
	return &ServiceAccountsService{
		authService: authService,
	}
}

func (s *ServiceAccountsService) CreateServiceAccount(ctx context.Context, req *pb.CreateServiceAccountRequest) (*pb.CreateServiceAccountResponse, error) {
	return service_accounts.CreateServiceAccount(ctx, req, s.authService)
}

func (s *ServiceAccountsService) ListServiceAccounts(ctx context.Context, req *pb.ListServiceAccountsRequest) (*pb.ListServiceAccountsResponse, error) {
	return service_accounts.ListServiceAccounts(ctx)
}

func (s *ServiceAccountsService) DescribeServiceAccount(ctx context.Context, req *pb.DescribeServiceAccountRequest) (*pb.DescribeServiceAccountResponse, error) {
	return service_accounts.DescribeServiceAccount(ctx, req)
}

func (s *ServiceAccountsService) UpdateServiceAccount(ctx context.Context, req *pb.UpdateServiceAccountRequest) (*pb.UpdateServiceAccountResponse, error) {
	return service_accounts.UpdateServiceAccount(ctx, req)
}

func (s *ServiceAccountsService) DeleteServiceAccount(ctx context.Context, req *pb.DeleteServiceAccountRequest) (*pb.DeleteServiceAccountResponse, error) {
	return service_accounts.DeleteServiceAccount(ctx, req, s.authService)
}

func (s *ServiceAccountsService) RegenerateServiceAccountToken(ctx context.Context, req *pb.RegenerateServiceAccountTokenRequest) (*pb.RegenerateServiceAccountTokenResponse, error) {
	return service_accounts.RegenerateServiceAccountToken(ctx, req)
}
