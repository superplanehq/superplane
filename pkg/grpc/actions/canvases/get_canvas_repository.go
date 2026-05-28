package canvases

import (
	"context"

	"github.com/superplanehq/superplane/pkg/git"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetCanvasRepository(
	ctx context.Context,
	organizationID string,
	canvasID string,
	storage git.Provider,
	options CanvasRepositoryStorageOptions,
) (*pb.GetCanvasRepositoryResponse, error) {
	if storage == nil {
		return nil, status.Error(codes.FailedPrecondition, "canvas file storage is not configured")
	}

	repository, err := loadCanvasRepository(organizationID, canvasID)
	if err != nil {
		return nil, err
	}

	return &pb.GetCanvasRepositoryResponse{Repository: serializeCanvasRepository(ctx, repository, storage)}, nil
}
