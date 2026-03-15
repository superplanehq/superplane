package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/extensions"
	pb "github.com/superplanehq/superplane/pkg/protos/extensions"
	"github.com/superplanehq/superplane/pkg/registry"
)

type ExtensionService struct {
	registry *registry.Registry
	storage  *extensions.ExtensionStorage
}

func NewExtensionService(registry *registry.Registry) *ExtensionService {
	return &ExtensionService{registry: registry, storage: extensions.NewExtensionStorage()}
}

func (s *ExtensionService) ListExtensions(ctx context.Context, req *pb.ListExtensionsRequest) (*pb.ListExtensionsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return extensions.ListExtensions(ctx, s.storage, organizationID)
}

func (s *ExtensionService) CreateExtension(ctx context.Context, req *pb.CreateExtensionRequest) (*pb.CreateExtensionResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return extensions.CreateExtension(ctx, s.storage, organizationID, req.Name, req.Description)
}

func (s *ExtensionService) CreateVersion(ctx context.Context, req *pb.CreateVersionRequest) (*pb.CreateVersionResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return extensions.CreateVersion(ctx, s.storage, organizationID, req.ExtensionId, req.Bundle, req.Digest)
}

func (s *ExtensionService) UpdateVersion(ctx context.Context, req *pb.UpdateVersionRequest) (*pb.UpdateVersionResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return extensions.UpdateVersion(ctx, s.storage, organizationID, req.ExtensionId, req.VersionId, req.Bundle, req.Digest)
}

func (s *ExtensionService) PublishVersion(ctx context.Context, req *pb.PublishVersionRequest) (*pb.PublishVersionResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return extensions.PublishVersion(ctx, s.storage, organizationID, req.ExtensionId, req.VersionId, req.Version)
}

func (s *ExtensionService) ListVersions(ctx context.Context, req *pb.ListVersionsRequest) (*pb.ListVersionsResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return extensions.ListVersions(ctx, s.storage, organizationID, req.ExtensionId)
}
