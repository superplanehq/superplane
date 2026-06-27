package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvasfolders"
	pb "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
)

type CanvasFolderService struct{}

func NewCanvasFolderService() *CanvasFolderService {
	return &CanvasFolderService{}
}

func (s *CanvasFolderService) ListCanvasFolders(ctx context.Context, req *pb.ListCanvasFoldersRequest) (*pb.ListCanvasFoldersResponse, error) {
	organizationID := authorization.OrganizationIDFromContext(ctx)
	return canvasfolders.ListCanvasFolders(ctx, organizationID)
}

func (s *CanvasFolderService) CreateCanvasFolder(ctx context.Context, req *pb.CreateCanvasFolderRequest) (*pb.CreateCanvasFolderResponse, error) {
	organizationID := authorization.OrganizationIDFromContext(ctx)
	return canvasfolders.CreateCanvasFolder(ctx, organizationID, req.Folder)
}

func (s *CanvasFolderService) UpdateCanvasFolder(ctx context.Context, req *pb.UpdateCanvasFolderRequest) (*pb.UpdateCanvasFolderResponse, error) {
	organizationID := authorization.OrganizationIDFromContext(ctx)
	return canvasfolders.UpdateCanvasFolder(ctx, organizationID, req.Id, req.Folder, req.ReplaceMembership)
}

func (s *CanvasFolderService) UpdateCanvasFolderPosition(
	ctx context.Context,
	req *pb.UpdateCanvasFolderPositionRequest,
) (*pb.UpdateCanvasFolderPositionResponse, error) {
	organizationID := authorization.OrganizationIDFromContext(ctx)
	return canvasfolders.UpdateCanvasFolderPosition(ctx, organizationID, req.Id, req.Direction)
}

func (s *CanvasFolderService) DeleteCanvasFolder(ctx context.Context, req *pb.DeleteCanvasFolderRequest) (*pb.DeleteCanvasFolderResponse, error) {
	organizationID := authorization.OrganizationIDFromContext(ctx)
	return canvasfolders.DeleteCanvasFolder(ctx, organizationID, req.Id)
}
