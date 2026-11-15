package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/enterprise"
	actions "github.com/superplanehq/superplane/pkg/grpc/actions/admin/sso"
	pb "github.com/superplanehq/superplane/pkg/protos/admin/sso"
)

type AdminSSOService struct {
	pb.UnimplementedAdminSSOServer
	sso enterprise.SSO
}

func NewAdminSSOService(sso enterprise.SSO) *AdminSSOService {
	return &AdminSSOService{sso: sso}
}

func (s *AdminSSOService) GetOrganizationSSO(ctx context.Context, req *pb.GetOrganizationSSORequest) (*pb.GetOrganizationSSOResponse, error) {
	return actions.GetOrganization(ctx, s.sso, req.GetOrganizationId())
}

func (s *AdminSSOService) DisableOrganizationSSO(ctx context.Context, req *pb.DisableOrganizationSSORequest) (*pb.GetOrganizationSSOResponse, error) {
	return actions.DisableOrganization(ctx, s.sso, req.GetOrganizationId())
}

func (s *AdminSSOService) VerifyOrganizationDomain(ctx context.Context, req *pb.VerifyOrganizationDomainRequest) (*pb.VerifyOrganizationDomainResponse, error) {
	accountID, err := accountIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return actions.VerifyDomain(ctx, s.sso, req.GetOrganizationId(), req.GetDomainId(), accountID)
}
