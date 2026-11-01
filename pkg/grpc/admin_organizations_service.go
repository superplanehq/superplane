package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	organizationactions "github.com/superplanehq/superplane/pkg/grpc/actions/admin/organizations"
	pb "github.com/superplanehq/superplane/pkg/protos/admin/organizations"
)

type AdminOrganizationsService struct {
	pb.UnimplementedAdminOrganizationsServer
	authService authorization.Authorization
}

func NewAdminOrganizationsService(authService authorization.Authorization) *AdminOrganizationsService {
	return &AdminOrganizationsService{authService: authService}
}

func (s *AdminOrganizationsService) DeleteOrganization(ctx context.Context, req *pb.DeleteOrganizationRequest) (*pb.DeleteOrganizationResponse, error) {
	return organizationactions.DeleteOrganization(ctx, s.authService, req)
}

func (s *AdminOrganizationsService) SuspendOrganization(ctx context.Context, req *pb.SuspendOrganizationRequest) (*pb.SuspensionResponse, error) {
	return organizationactions.SuspendOrganization(ctx, req)
}

func (s *AdminOrganizationsService) ReactivateOrganization(ctx context.Context, req *pb.ReactivateOrganizationRequest) (*pb.SuspensionResponse, error) {
	return organizationactions.ReactivateOrganization(ctx, req)
}

func (s *AdminOrganizationsService) ListSuspensionHistory(ctx context.Context, req *pb.ListSuspensionHistoryRequest) (*pb.SuspensionResponse, error) {
	return organizationactions.ListSuspensionHistory(ctx, req)
}
