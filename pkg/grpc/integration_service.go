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

func (s *IntegrationService) UpdateIntegration(ctx context.Context, req *pb.UpdateIntegrationRequest) (*pb.UpdateIntegrationResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return integrations.UpdateIntegration(ctx, s.encryptor, s.registry, domainType, domainID, req.IdOrName, req.Integration)
}

func (s *IntegrationService) ListResources(ctx context.Context, req *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return integrations.ListResources(ctx, s.registry, domainType, domainID, req.IdOrName, req.Type)
}

func (s *IntegrationService) ListComponents(ctx context.Context, req *pb.ListComponentsRequest) (*pb.ListComponentsResponse, error) {
	return integrations.ListComponents(ctx, s.registry, req.Type)
}
