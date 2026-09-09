package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/files"
	pb "github.com/superplanehq/superplane/pkg/protos/files"
)

type FilesService struct {
	pb.UnimplementedFilesServer
}

func NewFilesService(_ authorization.Authorization) *FilesService {
	return &FilesService{}
}

func (s *FilesService) CreateFactoryFile(ctx context.Context, req *pb.CreateFactoryFileRequest) (*pb.CreateFactoryFileResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return files.CreateFactoryFile(ctx, organizationID, req)
}

func (s *FilesService) ListFactoryFiles(ctx context.Context, req *pb.ListFactoryFilesRequest) (*pb.ListFactoryFilesResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return files.ListFactoryFiles(ctx, organizationID, req)
}

func (s *FilesService) CreateWorkOrderFile(ctx context.Context, req *pb.CreateWorkOrderFileRequest) (*pb.CreateWorkOrderFileResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return files.CreateWorkOrderFile(ctx, organizationID, req)
}

func (s *FilesService) ListWorkOrderFiles(ctx context.Context, req *pb.ListWorkOrderFilesRequest) (*pb.ListWorkOrderFilesResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return files.ListWorkOrderFiles(ctx, organizationID, req)
}
