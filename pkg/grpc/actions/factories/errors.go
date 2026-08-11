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
	case errors.Is(err, models.ErrFactoryNameRequired):
		return grpcerrors.InvalidArgument(err, "factory name is required")
	case errors.Is(err, models.ErrFactoryNotFound):
		return grpcerrors.NotFound(err, "factory not found")
	case errors.Is(err, models.ErrFactoryWorkOrderNotFound):
		return grpcerrors.NotFound(err, "work order not found")
	case errors.Is(err, models.ErrFactoryLineNotFound):
		return grpcerrors.NotFound(err, "factory line not found")
	case errors.Is(err, models.ErrFactoryLineNameAlreadyExists):
		return grpcerrors.AlreadyExists(err, "factory line with the same name already exists")
	case errors.Is(err, models.ErrFactoryWorkOrderNotDispatchable):
		return grpcerrors.FailedPrecondition(err, "work order cannot be dispatched in its current state")
	case errors.Is(err, models.ErrFactoryWorkOrderInvalidState):
		return grpcerrors.FailedPrecondition(err, err.Error())
	case errors.Is(err, models.ErrFactoryWorkOrderExecutionActive):
		return grpcerrors.FailedPrecondition(err, "work order already has an active execution")
	case errors.Is(err, models.ErrFactoryLineHasNoSteps):
		return grpcerrors.FailedPrecondition(err, "factory line has no steps")
	case errors.Is(err, models.ErrFactoryLineStepNotOnRun):
		return grpcerrors.FailedPrecondition(err, "factory line step entrypoint must use the onRun trigger")
	case errors.Is(err, models.ErrFactoryWorkOrderArtifactInvalid):
		return grpcerrors.InvalidArgument(err, err.Error())
	case errors.Is(err, models.ErrFactoryWorkOrderApprovalNotFound):
		return grpcerrors.NotFound(err, "work order approval not found")
	case errors.Is(err, models.ErrFactoryWorkOrderApprovalAlreadyClosed):
		return grpcerrors.FailedPrecondition(err, "work order approval is already resolved")
	case errors.Is(err, models.ErrFactoryWorkOrderApprovalInvalidStatus):
		return grpcerrors.InvalidArgument(err, "approval status must be approved or rejected")
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
