package grpc

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/superplanehq/superplane/pkg/enterprise"
	actions "github.com/superplanehq/superplane/pkg/grpc/actions/sso"
	pb "github.com/superplanehq/superplane/pkg/protos/sso"
)

type SSOService struct {
	pb.UnimplementedSSOServer
	sso enterprise.SSO
}

func NewSSOService(sso enterprise.SSO) *SSOService {
	return &SSOService{sso: sso}
}

func (s *SSOService) GetConnection(ctx context.Context, _ *empty.Empty) (*pb.GetConnectionResponse, error) {
	return actions.GetConnection(ctx, s.sso)
}

func (s *SSOService) PutConnection(ctx context.Context, req *pb.PutConnectionRequest) (*pb.ConnectionResponse, error) {
	return actions.PutConnection(ctx, s.sso, req)
}

func (s *SSOService) ActivateConnection(ctx context.Context, _ *empty.Empty) (*pb.ConnectionResponse, error) {
	return actions.ActivateConnection(ctx, s.sso)
}

func (s *SSOService) DisableConnection(ctx context.Context, _ *empty.Empty) (*pb.ConnectionResponse, error) {
	return actions.DisableConnection(ctx, s.sso)
}

func (s *SSOService) DeleteConnection(ctx context.Context, _ *empty.Empty) (*empty.Empty, error) {
	return actions.DeleteConnection(ctx, s.sso)
}

func (s *SSOService) ListDomains(ctx context.Context, _ *empty.Empty) (*pb.ListDomainsResponse, error) {
	return actions.ListDomains(ctx, s.sso)
}

func (s *SSOService) CreateDomain(ctx context.Context, req *pb.CreateDomainRequest) (*pb.DomainResponse, error) {
	return actions.CreateDomain(ctx, s.sso, req)
}

func (s *SSOService) VerifyDomain(ctx context.Context, req *pb.VerifyDomainRequest) (*pb.DomainResponse, error) {
	return actions.VerifyDomain(ctx, s.sso, req)
}

func (s *SSOService) DeleteDomain(ctx context.Context, req *pb.DeleteDomainRequest) (*empty.Empty, error) {
	return actions.DeleteDomain(ctx, s.sso, req)
}

func (s *SSOService) ListSCIMTokens(ctx context.Context, _ *empty.Empty) (*pb.ListSCIMTokensResponse, error) {
	return actions.ListSCIMTokens(ctx, s.sso)
}

func (s *SSOService) CreateSCIMToken(ctx context.Context, req *pb.CreateSCIMTokenRequest) (*pb.CreateSCIMTokenResponse, error) {
	return actions.CreateSCIMToken(ctx, s.sso, req)
}

func (s *SSOService) RevokeSCIMToken(ctx context.Context, req *pb.RevokeSCIMTokenRequest) (*empty.Empty, error) {
	return actions.RevokeSCIMToken(ctx, s.sso, req)
}
