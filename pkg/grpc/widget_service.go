package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions/widgets"
	pb "github.com/superplanehq/superplane/pkg/protos/widgets"
	"github.com/superplanehq/superplane/pkg/registry"
)

type WidgetService struct {
	registry *registry.Registry
}

func NewWidgetService(registry *registry.Registry) *WidgetService {
	return &WidgetService{registry: registry}
}

func (s *WidgetService) ListWidgets(ctx context.Context, req *pb.ListWidgetsRequest) (*pb.ListWidgetsResponse, error) {
	return widgets.ListWidgets(ctx, s.registry)
}

func (s *WidgetService) DescribeWidget(ctx context.Context, req *pb.DescribeWidgetRequest) (*pb.DescribeWidgetResponse, error) {
	return widgets.DescribeWidget(ctx, s.registry, req.Name)
}
