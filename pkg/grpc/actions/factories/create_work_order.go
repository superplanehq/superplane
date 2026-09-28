package factories

import (
	"context"
	"strings"
	"time"

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

const manualTaskCreateReplayWindow = 2 * time.Minute

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

	target, err := oldestManualTaskIssueIntake(db, factory)
	if err != nil {
		log.WithError(err).Warnf("factory %s: failed to resolve a GitHub issue for a manual task", factory.ID)
		target = nil
	}

	if target != nil {
		existing, findErr := factory.FindRecentIdenticalManualWorkOrder(
			db,
			createdByID,
			title,
			req.GetDescription(),
			time.Now().Add(-manualTaskCreateReplayWindow),
		)
		if findErr != nil {
			return nil, factoryErrorToStatus(findErr, "failed to create work order")
		}
		if existing != nil {
			return finishReplayedManualWorkOrder(ctx, deps, factory, existing)
		}
	}

	marker := ""
	if target != nil {
		marker = ghintegration.NewManualTaskMarker()
	}

	assigneeIDs := []uuid.UUID{createdByID}
	var order *models.FactoryWorkOrder
	var bound storedfiles.BindResult
	err = db.Transaction(func(tx *gorm.DB) error {
		created, createErr := factory.CreateWorkOrder(tx, title, req.GetDescription(), &createdByID, assigneeIDs, nil)
		if createErr != nil {
			return createErr
		}
		if marker != "" {
			if markErr := created.SetPendingGitHubMarker(tx, marker); markErr != nil {
				return markErr
			}
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
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	workCtx := context.WithoutCancel(ctx)
	workDB := database.DB(workCtx)
	if target != nil {
		attachManualTaskGitHubIssue(workCtx, deps, workDB, factory, order, target, marker, title, req.GetDescription())
		reloaded, reloadErr := factory.FindWorkOrder(workDB, order.ID)
		if reloadErr != nil {
			log.WithError(reloadErr).Warnf("factory %s: failed to reload manual task %s after GitHub issue creation", factory.ID, order.ID)
		} else {
			order = reloaded
		}
	}

	workersctx.EmitWorkOrderCreated(workDB, factory, order)

	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderStatusUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}

	serialized, err := loadAndSerializeWorkOrder(workCtx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}

	return &pb.CreateWorkOrderResponse{
		Order: serialized,
	}, nil
}

func finishReplayedManualWorkOrder(
	ctx context.Context,
	deps IntakeDependencies,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
) (*pb.CreateWorkOrderResponse, error) {
	workCtx := context.WithoutCancel(ctx)
	workDB := database.DB(workCtx)
	if order.Origin() == nil && order.OriginLabel != nil {
		if marker, ok := ghintegration.ManualTaskMarkerFromLabel(*order.OriginLabel); ok {
			target, targetErr := oldestManualTaskIssueIntake(workDB, factory)
			if targetErr != nil {
				log.WithError(targetErr).Warnf("factory %s: failed to resolve a GitHub issue for replayed task %s", factory.ID, order.ID)
			} else if target != nil {
				reconcileManualTaskGitHubIssue(workCtx, deps, workDB, factory, order, target, marker)
				reloaded, reloadErr := factory.FindWorkOrder(workDB, order.ID)
				if reloadErr != nil {
					log.WithError(reloadErr).Warnf("factory %s: failed to reload replayed task %s", factory.ID, order.ID)
				} else {
					order = reloaded
				}
			}
		}
	}

	serialized, err := loadAndSerializeWorkOrder(workCtx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create work order")
	}
	return &pb.CreateWorkOrderResponse{Order: serialized}, nil
}
