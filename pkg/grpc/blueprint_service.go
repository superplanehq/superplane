package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/blueprints"
	pb "github.com/superplanehq/superplane/pkg/protos/blueprints"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type BlueprintService struct {
	registry *registry.Registry
}

func NewBlueprintService(registry *registry.Registry) *BlueprintService {
	return &BlueprintService{registry: registry}
}

func (s *BlueprintService) ListBlueprints(ctx context.Context, req *pb.ListBlueprintsRequest) (*pb.ListBlueprintsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return blueprints.ListBlueprints(ctx, s.registry, organizationID)
}

func (s *BlueprintService) DescribeBlueprint(ctx context.Context, req *pb.DescribeBlueprintRequest) (*pb.DescribeBlueprintResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return blueprints.DescribeBlueprint(ctx, s.registry, organizationID, req.Id)
}

func (s *BlueprintService) CreateBlueprint(ctx context.Context, req *pb.CreateBlueprintRequest) (*pb.CreateBlueprintResponse, error) {
	if req.Blueprint == nil {
		return nil, status.Error(codes.InvalidArgument, "blueprint is required")
	}
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return blueprints.CreateBlueprint(ctx, s.registry, organizationID, req.Blueprint)
}

func (s *BlueprintService) UpdateBlueprint(ctx context.Context, req *pb.UpdateBlueprintRequest) (*pb.UpdateBlueprintResponse, error) {
	if req.Blueprint == nil {
		return nil, status.Error(codes.InvalidArgument, "blueprint is required")
	}
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return blueprints.UpdateBlueprint(ctx, s.registry, organizationID, req.Id, req.Blueprint)
}

func (s *BlueprintService) DeleteBlueprint(ctx context.Context, req *pb.DeleteBlueprintRequest) (*pb.DeleteBlueprintResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return blueprints.DeleteBlueprint(ctx, organizationID, req.Id)
}
