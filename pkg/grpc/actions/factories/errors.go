package factories

import (
	"errors"

	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func factoryErrorToStatus(err error, internalMessage string) error {
	switch {
	case errors.Is(err, models.ErrFactoryNameAlreadyExists):
		return grpcerrors.AlreadyExists(err, "factory with the same name already exists")
	case errors.Is(err, models.ErrFactoryNotFound):
		return grpcerrors.NotFound(err, "factory not found")
	case errors.Is(err, models.ErrFactorySourceNotFound):
		return grpcerrors.NotFound(err, "factory source not found")
	case errors.Is(err, models.ErrFactoryAgentNotFound):
		return grpcerrors.NotFound(err, "factory agent not found")
	case errors.Is(err, models.ErrFactoryAgentNameAlreadyExists):
		return grpcerrors.AlreadyExists(err, "factory agent with the same name already exists")
	case errors.Is(err, models.ErrFactoryWorkOrderNotFound):
		return grpcerrors.NotFound(err, "work order not found")
	case errors.Is(err, models.ErrFactoryWorkOrderSourceAlreadyExists):
		return grpcerrors.AlreadyExists(err, "work order for source already exists")
	case errors.Is(err, models.ErrFactoryAgentAssignmentNotFound):
		return grpcerrors.NotFound(err, "agent assignment not found")
	case errors.Is(err, errInvalidArgument):
		return grpcerrors.InvalidArgument(err, err.Error())
	case errors.Is(err, gorm.ErrRecordNotFound):
		return grpcerrors.NotFound(err, "resource not found")
	default:
		return grpcerrors.Internal(err, internalMessage)
	}
}

var errInvalidArgument = errors.New("invalid argument")

func invalidArgument(message string) error {
	return errors.Join(errInvalidArgument, errors.New(message))
}
