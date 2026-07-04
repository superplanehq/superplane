package canvasfolders

import (
	"errors"

	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func canvasFolderErrorToStatus(err error, internalMessage string) error {
	switch {
	case errors.Is(err, models.ErrCanvasFolderTitleRequired):
		return grpcerrors.InvalidArgument(err, "canvas folder title is required")
	case errors.Is(err, models.ErrCanvasFolderTitleTooLong):
		return grpcerrors.InvalidArgument(err, "canvas folder title must be 128 characters or less")
	case errors.Is(err, models.ErrCanvasFolderInvalidBackgroundColor):
		return grpcerrors.InvalidArgument(err, "invalid canvas folder background color")
	case errors.Is(err, models.ErrCanvasFolderInvalidMoveDirection):
		return grpcerrors.InvalidArgument(err, "invalid canvas folder move direction")
	case errors.Is(err, models.ErrCanvasFolderTitleAlreadyExists):
		return grpcerrors.AlreadyExists(err, "canvas folder with the same title already exists")
	case errors.Is(err, models.ErrCanvasFolderCanvasNotFound):
		return grpcerrors.NotFound(err, "canvas not found")
	case errors.Is(err, gorm.ErrRecordNotFound):
		return grpcerrors.NotFound(err, "canvas folder not found")
	default:
		return grpcerrors.Internal(err, internalMessage)
	}
}
