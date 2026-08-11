package factories

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

// ResolveWorkOrderApproval marks a pending approval as approved or rejected
// and records the corresponding timeline event. It does not pause/resume the
// underlying line run - that will land in a follow-up change once the runner
// wiring is in place.
func ResolveWorkOrderApproval(
	ctx context.Context,
	organizationID string,
	req *pb.ResolveWorkOrderApprovalRequest,
) (*pb.ResolveWorkOrderApprovalResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to resolve work order approval")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to resolve work order approval")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to resolve work order approval")
	}

	approvalID, err := parseApprovalID(req.GetApprovalId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to resolve work order approval")
	}

	status, ok := workOrderApprovalStatusFromProto(req.GetStatus())
	if !ok || status == models.FactoryWorkOrderApprovalStatusPending {
		return nil, factoryErrorToStatus(models.ErrFactoryWorkOrderApprovalInvalidStatus, "failed to resolve work order approval")
	}

	userIDStr, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	resolverID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to resolve work order approval")
	}

	db := database.DB(ctx)
	err = db.Transaction(func(tx *gorm.DB) error {
		approval, err := models.LockFactoryWorkOrderApproval(tx, orgID, approvalID)
		if err != nil {
			return err
		}

		if approval.WorkOrderID != orderID || approval.FactoryID != factoryID {
			return models.ErrFactoryWorkOrderApprovalNotFound
		}

		return approval.Resolve(tx, status, resolverID, req.GetComment())
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to resolve work order approval")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryID.String(),
		orderID.String(),
		factoryevents.EventTypeOrderApprovalResolved,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", orderID)
	}

	loaded, err := models.FindFactoryWorkOrderApproval(database.DB(ctx), orgID, approvalID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to resolve work order approval")
	}

	return &pb.ResolveWorkOrderApprovalResponse{
		Approval: serializeWorkOrderApproval(loaded),
	}, nil
}
