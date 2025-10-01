package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions/manifests"
	pb "github.com/superplanehq/superplane/pkg/protos/manifests"
	"github.com/superplanehq/superplane/pkg/registry"
)

type ManifestService struct {
	pb.UnimplementedManifestServiceServer
	Registry *registry.Registry
}

func NewManifestService(registry *registry.Registry) *ManifestService {
	return &ManifestService{
		Registry: registry,
	}
}

func (s *ManifestService) GetManifests(ctx context.Context, req *pb.GetManifestsRequest) (*pb.GetManifestsResponse, error) {
	return manifests.GetManifests(ctx, req, s.Registry)
}
