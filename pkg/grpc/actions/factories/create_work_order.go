package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	ghintegration "github.com/superplanehq/superplane/pkg/integrations/github"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	workersctx "github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

func CreateWorkOrder(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.CreateWorkOrderRequest,
) (*pb.CreateWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	title := strings.TrimSpace(req.GetTitle())
	if title == "" {
		return nil, factoryErrorToStatus(invalidArgument("title is required"), "failed to create work order")
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	createdByID, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to create work order")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	assigneeIDs := []uuid.UUID{createdByID}
	openedIssue, hasIssue := manualTaskGitHubOrigin(ctx, deps, db, factory, title, req.GetDescription())
	var order *models.FactoryWorkOrder
	var bound storedfiles.BindResult
	var reused bool
	err = db.Transaction(func(tx *gorm.DB) error {
		var created *models.FactoryWorkOrder
		var createErr error
		if hasIssue {
			if lockErr := ghintegration.LockIssueWorkOrder(tx, factory, openedIssue.origin.URL); lockErr != nil {
				return lockErr
			}
			existing, findErr := ghintegration.FindIssueWorkOrder(tx, factory, openedIssue.origin.URL)
			if findErr != nil {
				return findErr
			}
			if existing != nil {
				order = existing
				reused = true
				return nil
			}
			created, createErr = factory.CreateWorkOrderWithOrigin(
				tx,
				title,
				req.GetDescription(),
				&createdByID,
				assigneeIDs,
				nil,
				openedIssue.origin,
			)
		} else {
			created, createErr = factory.CreateWorkOrder(tx, title, req.GetDescription(), &createdByID, assigneeIDs, nil)
		}
		if createErr != nil {
			return createErr
		}
		order = created
		result, bindErr := storedfiles.BindDescriptionFiles(
			ctx,
			tx,
			blob.Current(),
			orgID,
			factory.ID,
			order.ID,
			order.Description,
		)
		bound = result
		return bindErr
	})
	if delErr := storedfiles.ApplyBindResult(ctx, db, blob.Current(), orgID, factory.ID, bound, err); delErr != nil {
		log.WithError(delErr).Warn("Failed to delete file objects after bind")
	}
	if err != nil {
		if hasIssue {
			closeManualTaskGitHubIssue(deps, db, openedIssue)
		}
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}
	if reused {
		serialized, serializeErr := loadAndSerializeWorkOrder(ctx, factory, order)
		if serializeErr != nil {
			return nil, factoryErrorToStatus(serializeErr, "failed to create work order")
		}
		return &pb.CreateWorkOrderResponse{Order: serialized}, nil
	}

	workersctx.EmitWorkOrderCreated(db, factory, order)

	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderStatusUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	return &pb.CreateWorkOrderResponse{
		Order: serialized,
	}, nil
}
