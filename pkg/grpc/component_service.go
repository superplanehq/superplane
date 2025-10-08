package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions/components"
	pb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/registry"
)

type ComponentService struct {
	registry *registry.Registry
}

func NewComponentService(registry *registry.Registry) *ComponentService {
	return &ComponentService{registry: registry}
}

func (s *ComponentService) ListComponents(ctx context.Context, req *pb.ListComponentsRequest) (*pb.ListComponentsResponse, error) {
	return components.ListComponents(ctx, s.registry)
}

func (s *ComponentService) DescribeComponent(ctx context.Context, req *pb.DescribeComponentRequest) (*pb.DescribeComponentResponse, error) {
	return components.DescribeComponent(ctx, s.registry, req.Name)
}

func (s *ComponentService) ListComponentActions(ctx context.Context, req *pb.ListComponentActionsRequest) (*pb.ListComponentActionsResponse, error) {
	return components.ListComponentActions(ctx, s.registry, req.Name)
}
