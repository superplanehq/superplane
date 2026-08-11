package factories

import (
	"context"
	"errors"
	"strings"

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

// CreateWorkOrderApproval opens a pending approval on the given work order,
// optionally tied to an existing execution. The approver defaults to
// unassigned; anyone with the required permissions can resolve.
func CreateWorkOrderApproval(
	ctx context.Context,
	organizationID string,
	req *pb.CreateWorkOrderApprovalRequest,
) (*pb.CreateWorkOrderApprovalResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order approval")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order approval")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order approval")
	}

	userIDStr, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	requesterID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to create work order approval")
	}

	var executionID *uuid.UUID
	if trimmed := strings.TrimSpace(req.GetExecutionId()); trimmed != "" {
		parsed, err := parseExecutionID(trimmed)
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to create work order approval")
		}
		executionID = &parsed
	}

	var approverID *uuid.UUID
	if trimmed := strings.TrimSpace(req.GetApproverId()); trimmed != "" {
		parsed, err := uuid.Parse(trimmed)
		if err != nil {
			return nil, factoryErrorToStatus(invalidArgument("invalid approver id"), "failed to create work order approval")
		}
		approverID = &parsed
	}

	title := strings.TrimSpace(req.GetTitle())
	if title == "" {
		return nil, factoryErrorToStatus(invalidArgument("title is required"), "failed to create work order approval")
	}

	var approval *models.FactoryWorkOrderApproval
	db := database.DB(ctx)
	err = db.Transaction(func(tx *gorm.DB) error {
		f, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}

		order, err := f.FindWorkOrder(tx, orderID)
		if err != nil {
			return err
		}

		if approverID != nil {
			if _, err := models.FindActiveUserByIDInTransaction(tx, orgID.String(), approverID.String()); err != nil {
				return invalidArgument("approver not found")
			}
		}

		if executionID != nil {
			// Cross-order attachment would let a client pin an approval on
			// this order to another order's execution — reject up front.
			if _, err := models.FindWorkOrderExecutionByOrder(tx, orgID, factoryID, orderID, *executionID); err != nil {
				if errors.Is(err, models.ErrFactoryWorkOrderExecutionNotFound) {
					return invalidArgument("execution not found for this work order")
				}
				return err
			}
		}

		approval = models.NewFactoryWorkOrderApproval(
			order,
			executionID,
			title,
			req.GetMessage(),
			approverID,
			&requesterID,
		)

		if err := tx.Create(approval).Error; err != nil {
			return err
		}

		return order.RecordApprovalRequested(tx, approval, requesterID)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order approval")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryID.String(),
		orderID.String(),
		factoryevents.EventTypeOrderApprovalRequested,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", orderID)
	}

	// Reload with preloaded relations for serialization
	loaded, err := models.FindFactoryWorkOrderApproval(database.DB(ctx), orgID, approval.ID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order approval")
	}

	return &pb.CreateWorkOrderApprovalResponse{
		Approval: serializeWorkOrderApproval(loaded),
	}, nil
}
