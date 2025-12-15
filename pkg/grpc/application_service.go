package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions/applications"
	pb "github.com/superplanehq/superplane/pkg/protos/applications"
	"github.com/superplanehq/superplane/pkg/registry"
)

type ApplicationService struct {
	encryptor crypto.Encryptor
	registry  *registry.Registry
}

func NewApplicationService(encryptor crypto.Encryptor, registry *registry.Registry) *ApplicationService {
	return &ApplicationService{
		encryptor: encryptor,
		registry:  registry,
	}
}

func (s *ApplicationService) ListApplications(ctx context.Context, req *pb.ListApplicationsRequest) (*pb.ListApplicationsResponse, error) {
	return applications.ListApplications(ctx, s.registry)
}
