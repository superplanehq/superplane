package canvases

import (
	"errors"

	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func canvasFolderErrorToStatus(err error) error {
	switch {
	case errors.Is(err, models.ErrCanvasFolderTitleRequired):
		return status.Error(codes.InvalidArgument, "canvas folder title is required")
	case errors.Is(err, models.ErrCanvasFolderTitleTooLong):
		return status.Error(codes.InvalidArgument, "canvas folder title must be 128 characters or less")
	case errors.Is(err, models.ErrCanvasFolderInvalidBackgroundColor):
		return status.Error(codes.InvalidArgument, "invalid canvas folder background color")
	case errors.Is(err, models.ErrCanvasFolderInvalidMoveDirection):
		return status.Error(codes.InvalidArgument, "invalid canvas folder move direction")
	case errors.Is(err, models.ErrCanvasFolderTitleAlreadyExists):
		return status.Error(codes.AlreadyExists, "canvas folder with the same title already exists")
	case errors.Is(err, gorm.ErrRecordNotFound):
		return status.Error(codes.NotFound, "canvas folder not found")
	default:
		return status.Error(codes.Internal, "failed to update canvas folder")
	}
}
