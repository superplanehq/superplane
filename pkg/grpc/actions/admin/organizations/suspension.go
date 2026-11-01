package organizations

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/admin/organizations"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func suspensionOrganization(ctx context.Context, id string) (*models.Organization, uuid.UUID, error) {
	accountID, ok := authentication.GetAccountIDFromMetadata(ctx)
	if !ok {
		return nil, uuid.Nil, grpcerrors.Unauthenticated(nil, "Account authentication is required.")
	}
	admin, err := models.FindAccountByID(accountID)
	if err != nil || !admin.IsInstallationAdmin() {
		return nil, uuid.Nil, grpcerrors.PermissionDenied(nil, "Installation admin access is required.")
	}
	orgID, err := uuid.Parse(id)
	if err != nil {
		return nil, uuid.Nil, grpcerrors.InvalidArgument(err, "Invalid organization ID.")
	}
	organization, err := models.FindOrganizationByIDInTransaction(database.DB(ctx), orgID.String())
	if err != nil {
		return nil, uuid.Nil, deletionError(err)
	}
	return organization, admin.ID, nil
}

func SuspendOrganization(ctx context.Context, req *pb.SuspendOrganizationRequest) (*pb.SuspensionResponse, error) {
	organization, actor, err := suspensionOrganization(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if err := organization.Suspend(database.DB(ctx), actor, req.GetReason()); err != nil {
		if errors.Is(err, models.ErrSuspensionReasonRequired) {
			return nil, grpcerrors.InvalidArgument(err, models.ErrSuspensionReasonRequired.Error())
		}
		return nil, grpcerrors.Internal(err, "Could not suspend the organization. Try again.")
	}
	return suspensionResponse(ctx, organization)
}

func ReactivateOrganization(ctx context.Context, req *pb.ReactivateOrganizationRequest) (*pb.SuspensionResponse, error) {
	organization, actor, err := suspensionOrganization(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if err := organization.Reactivate(database.DB(ctx), actor); err != nil {
		if errors.Is(err, models.ErrSuspensionCancellationPending) {
			return nil, grpcerrors.Conflict(err, "Cancellation is pending. Wait for existing work to stop, then try again.")
		}
		return nil, grpcerrors.Internal(err, "Could not reactivate the organization. Try again.")
	}
	return suspensionResponse(ctx, organization)
}

func ListSuspensionHistory(ctx context.Context, req *pb.ListSuspensionHistoryRequest) (*pb.SuspensionResponse, error) {
	organization, _, err := suspensionOrganization(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return suspensionResponse(ctx, organization)
}

func suspensionResponse(ctx context.Context, organization *models.Organization) (*pb.SuspensionResponse, error) {
	events, err := models.ListOrganizationSuspensionEvents(database.DB(ctx), organization.ID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "Could not load suspension history. Try again.")
	}
	response := &pb.SuspensionResponse{Suspended: organization.IsSuspended()}
	for _, event := range events {
		entry := &pb.SuspensionEvent{Id: event.ID.String(), Action: event.Action, Reason: event.Reason,
			AccountId: event.AccountID.String(), CreatedAt: timestamppb.New(event.CreatedAt)}
		if event.CancellationCompletedAt != nil {
			entry.CancellationCompletedAt = timestamppb.New(*event.CancellationCompletedAt)
		}
		response.Events = append(response.Events, entry)
	}
	return response, nil
}
