package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/grpc/actions/integrations"
	pb "github.com/superplanehq/superplane/pkg/protos/integrations"
)

type IntegrationService struct {
	encryptor            crypto.Encryptor
	specValidator        executors.SpecValidator
	authorizationService authorization.Authorization
}

func NewIntegrationService(encryptor crypto.Encryptor, authService authorization.Authorization) *IntegrationService {
	return &IntegrationService{
		encryptor:            encryptor,
		specValidator:        executors.SpecValidator{Encryptor: encryptor},
		authorizationService: authService,
	}
}

func (s *IntegrationService) CreateIntegration(ctx context.Context, req *pb.CreateIntegrationRequest) (*pb.CreateIntegrationResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainID := ctx.Value(authorization.DomainIdContextKey).(string)
	return integrations.CreateIntegration(ctx, s.encryptor, domainType, domainID, req.Integration)
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
