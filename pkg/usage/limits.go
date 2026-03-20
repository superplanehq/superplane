package usage

import (
	"context"
	"errors"

	pb "github.com/superplanehq/superplane/pkg/protos/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func EnsureAccountWithinLimits(ctx context.Context, usageService Service, accountID string, state *pb.AccountState) error {
	if usageService == nil || !usageService.Enabled() {
		return nil
	}

	response, err := usageService.CheckAccountLimits(ctx, accountID, state)
	if err == nil {
		return limitViolationError(response.GetViolations())
	}

	if status.Code(err) != codes.NotFound {
		return mapLimitCheckError("check account limits", err)
	}

	if _, setupErr := usageService.SetupAccount(ctx, accountID); setupErr != nil && status.Code(setupErr) != codes.AlreadyExists {
		return mapLimitCheckError("set up usage account", setupErr)
	}

	response, err = usageService.CheckAccountLimits(ctx, accountID, state)
	if err != nil {
		return mapLimitCheckError("check account limits after setup", err)
	}

	return limitViolationError(response.GetViolations())
}

func EnsureOrganizationWithinLimits(
	ctx context.Context,
	usageService Service,
	organizationID string,
	state *pb.OrganizationState,
	canvas *pb.CanvasState,
) error {
	if usageService == nil || !usageService.Enabled() {
		return nil
	}

	response, err := usageService.CheckOrganizationLimits(ctx, organizationID, state, canvas)
	if err == nil {
		return limitViolationError(response.GetViolations())
	}

	if status.Code(err) != codes.NotFound {
		return mapLimitCheckError("check organization limits", err)
	}

	if syncErr := SyncOrganizationForce(ctx, usageService, organizationID); syncErr != nil {
		return mapLimitSyncError(syncErr)
	}

	response, err = usageService.CheckOrganizationLimits(ctx, organizationID, state, canvas)
	if err != nil {
		return mapLimitCheckError("check organization limits after sync", err)
	}

	return limitViolationError(response.GetViolations())
}

func limitViolationError(violations []*pb.LimitViolation) error {
	if len(violations) == 0 {
		return nil
	}

	switch violations[0].GetLimit() {
	case pb.LimitName_LIMIT_NAME_MAX_ORGANIZATIONS:
		return status.Error(codes.ResourceExhausted, "account organization limit exceeded")
	case pb.LimitName_LIMIT_NAME_MAX_CANVASES:
		return status.Error(codes.ResourceExhausted, "organization canvas limit exceeded")
	case pb.LimitName_LIMIT_NAME_MAX_NODES_PER_CANVAS:
		return status.Error(codes.ResourceExhausted, "canvas node limit exceeded")
	case pb.LimitName_LIMIT_NAME_MAX_USERS:
		return status.Error(codes.ResourceExhausted, "organization user limit exceeded")
	case pb.LimitName_LIMIT_NAME_MAX_INTEGRATIONS:
		return status.Error(codes.ResourceExhausted, "organization integration limit exceeded")
	default:
		return status.Error(codes.ResourceExhausted, "organization usage limit exceeded")
	}
}

func mapLimitCheckError(action string, err error) error {
	if status.Code(err) == codes.ResourceExhausted {
		return err
	}

	return status.Errorf(codes.Internal, "%s: %v", action, err)
}

func mapLimitSyncError(err error) error {
	switch {
	case errors.Is(err, ErrNoBillingAccountCandidate), errors.Is(err, gorm.ErrRecordNotFound):
		return status.Error(codes.FailedPrecondition, "organization has no billing account candidate")
	case status.Code(err) == codes.ResourceExhausted:
		return status.Error(codes.ResourceExhausted, "organization exceeds configured account usage limits")
	default:
		return status.Error(codes.Internal, "failed to set up organization usage")
	}
}
