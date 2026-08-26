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
	case errors.Is(err, models.ErrFactoryKeyRequired):
		return grpcerrors.InvalidArgument(err, "workspace key is required")
	case errors.Is(err, models.ErrFactoryKeyInvalid):
		return grpcerrors.InvalidArgument(err, "workspace key must be 2 to 5 uppercase letters")
	case errors.Is(err, models.ErrFactoryKeyAlreadyExists):
		return grpcerrors.AlreadyExists(err, "workspace key already exists in this organization")
	case errors.Is(err, models.ErrFactoryNotFound):
		return grpcerrors.NotFound(err, "factory not found")
	case errors.Is(err, models.ErrFactoryHostedSpendBudgetNegative):
		return grpcerrors.InvalidArgument(err, "hosted spend limit cannot be negative")
	case errors.Is(err, models.ErrModelNotInParentList):
		return grpcerrors.InvalidArgument(err, "model is not in the parent selected-model list")
	case errors.Is(err, models.ErrFactoryOnboardingInvalidIssuesSource):
		return grpcerrors.InvalidArgument(err, "invalid issues source")
	case errors.Is(err, models.ErrFactoryOnboardingInvalidAgentHarness):
		return grpcerrors.InvalidArgument(err, "invalid agent harness")
	case errors.Is(err, models.ErrFactoryOnboardingInvalidIntegrationID):
		return grpcerrors.InvalidArgument(err, "invalid integration id")
	case errors.Is(err, models.ErrFactoryOnboardingInvalidAppID):
		return grpcerrors.InvalidArgument(err, "invalid provisioned app id")
	case errors.Is(err, models.ErrFactoryOnboardingInvalidLineID):
		return grpcerrors.InvalidArgument(err, "invalid provisioned line id")
	case errors.Is(err, models.ErrFactoryOnboardingInvalidRepository):
		return grpcerrors.InvalidArgument(err, "repository must use the owner/name format")
	case errors.Is(err, models.ErrFactoryOnboardingVCSIntegrationRequired):
		return grpcerrors.InvalidArgument(err, "version control integration is required to complete onboarding")
	case errors.Is(err, models.ErrFactoryOnboardingAgentIntegrationRequired):
		return grpcerrors.InvalidArgument(err, "agent integration or hosted credit is required to complete onboarding")
	case errors.Is(err, models.ErrFactoryOnboardingHostedAgentUnavailable):
		return grpcerrors.InvalidArgument(err, "agent integration or SuperPlane-hosted models are required to complete onboarding")
	case errors.Is(err, models.ErrFactoryOnboardingAppRepositoryRequired):
		return grpcerrors.InvalidArgument(err, "app repository is required to complete onboarding")
	case errors.Is(err, models.ErrFactoryOnboardingBacklogRepoRequired):
		return grpcerrors.InvalidArgument(err, "backlog repository is required to complete onboarding")
	case errors.Is(err, models.ErrFactoryOnboardingIssuesSourceRequired):
		return grpcerrors.InvalidArgument(err, "issues source is required to complete onboarding")
	case errors.Is(err, models.ErrFactoryOnboardingAgentHarnessRequired):
		return grpcerrors.InvalidArgument(err, "agent harness is required to complete onboarding")
	case errors.Is(err, models.ErrFactoryOnboardingAppIDRequired):
		return grpcerrors.InvalidArgument(err, "provisioned app id is required to complete onboarding")
	case errors.Is(err, models.ErrFactoryOnboardingLineIDRequired):
		return grpcerrors.InvalidArgument(err, "provisioned line id is required to complete onboarding")
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
	case errors.Is(err, models.ErrFactoryWorkOrderLineDispatchActive):
		return grpcerrors.FailedPrecondition(err, "work order already has an active line dispatch")
	case errors.Is(err, models.ErrFactoryLineHasNoSteps):
		return grpcerrors.FailedPrecondition(err, "factory line has no steps")
	case errors.Is(err, models.ErrFactoryLineStepNotOnRun):
		return grpcerrors.FailedPrecondition(err, "factory line step entrypoint must use the onRun trigger")
	case errors.Is(err, models.ErrFactoryWorkOrderArtifactInvalid):
		return grpcerrors.InvalidArgument(err, err.Error())
	case errors.Is(err, models.ErrFactoryIntakeNotFound):
		return grpcerrors.NotFound(err, "factory intake not found")
	case errors.Is(err, models.ErrFactoryIntakeSourceInvalid):
		return grpcerrors.InvalidArgument(err, "intake source is not supported")
	case errors.Is(err, models.ErrFactoryIntakeCanvasInUse):
		return grpcerrors.AlreadyExists(err, "canvas already implements an intake")
	case errors.Is(err, models.ErrFactoryIntakeCanvasRequired):
		return grpcerrors.InvalidArgument(err, "intake canvas is required")
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
