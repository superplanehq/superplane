package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions/integrations"
	pb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/pkg/registry"
)

type IntegrationService struct {
	encryptor crypto.Encryptor
	registry  *registry.Registry
}

func NewIntegrationService(encryptor crypto.Encryptor, registry *registry.Registry) *IntegrationService {
	return &IntegrationService{
		encryptor: encryptor,
		registry:  registry,
	}
}

func (s *IntegrationService) ListIntegrations(ctx context.Context, req *pb.ListIntegrationsRequest) (*pb.ListIntegrationsResponse, error) {
	return integrations.ListIntegrations(ctx, s.registry)
}
