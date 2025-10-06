package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions/primitives"
	pb "github.com/superplanehq/superplane/pkg/protos/primitives"
	"github.com/superplanehq/superplane/pkg/registry"
)

type PrimitiveService struct {
	registry *registry.Registry
}

func NewPrimitiveService(registry *registry.Registry) *PrimitiveService {
	return &PrimitiveService{registry: registry}
}

func (s *PrimitiveService) ListPrimitives(ctx context.Context, req *pb.ListPrimitivesRequest) (*pb.ListPrimitivesResponse, error) {
	return primitives.ListPrimitives(ctx, s.registry)
}

func (s *PrimitiveService) DescribePrimitive(ctx context.Context, req *pb.DescribePrimitiveRequest) (*pb.DescribePrimitiveResponse, error) {
	return primitives.DescribePrimitive(ctx, s.registry, req.Name)
}
