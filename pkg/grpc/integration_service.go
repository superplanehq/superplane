package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions/integrations"
	pb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/pkg/registry"
)

type IntegrationService struct {
	encryptor            crypto.Encryptor
	authorizationService authorization.Authorization
	registry             *registry.Registry
}

func NewIntegrationService(encryptor crypto.Encryptor, authService authorization.Authorization, registry *registry.Registry) *IntegrationService {
	return &IntegrationService{
		encryptor:            encryptor,
		authorizationService: authService,
		registry:             registry,
	}
}

func (s *IntegrationService) CreateIntegration(ctx context.Context, req *pb.CreateIntegrationRequest) (*pb.CreateIntegrationResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return integrations.CreateIntegration(ctx, s.encryptor, s.registry, domainType, domainID, req.Integration)
}

func (s *IntegrationService) DescribeIntegration(ctx context.Context, req *pb.DescribeIntegrationRequest) (*pb.DescribeIntegrationResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return integrations.DescribeIntegration(ctx, domainType, domainID, req.IdOrName)
}

func (s *IntegrationService) ListIntegrations(ctx context.Context, req *pb.ListIntegrationsRequest) (*pb.ListIntegrationsResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return integrations.ListIntegrations(ctx, domainType, domainID)
}
