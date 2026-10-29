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
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	workersctx "github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

func CreateWorkOrder(ctx context.Context, organizationID string, req *pb.CreateWorkOrderRequest) (*pb.CreateWorkOrderResponse, error) {
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

	var order *models.FactoryWorkOrder
	var bound storedfiles.BindResult
	err = db.Transaction(func(tx *gorm.DB) error {
		assigneeIDs, err := workOrderCreationOwners(tx, factory, createdByID, req)
		if err != nil {
			return err
		}
		if !factory.OnboardingConfigValue().CodexSubscription {
			if err := validateRefinementModel(tx, factory, strings.TrimSpace(req.GetModel())); err != nil {
				return err
			}
		}
		created, err := factory.CreateWorkOrder(tx, title, req.GetDescription(), &createdByID, assigneeIDs, nil)
		if err != nil {
			return err
		}
		order = created
		if err := order.SetRefinementModel(tx, strings.TrimSpace(req.GetModel())); err != nil {
			return err
		}
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

	workersctx.EmitWorkOrderCreated(db, factory, order)
	creationSource := "manual"
	if productanalytics.SourceFromContext(ctx, productanalytics.SourceAPI) == productanalytics.SourceMCP {
		creationSource = "mcp"
	}
	captureFactoryProductEvent(
		ctx,
		db,
		factory,
		order.ID,
		productanalytics.EventFactoryTaskCreated,
		productanalytics.StableEventID(productanalytics.EventFactoryTaskCreated, order.ID.String()),
		productanalytics.Properties{"creation_source": creationSource},
	)

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

func workOrderCreationOwners(tx *gorm.DB, factory *models.Factory, creatorID uuid.UUID, req *pb.CreateWorkOrderRequest) ([]uuid.UUID, error) {
	if !factory.OnboardingConfigValue().CodexSubscription {
		return factory.DefaultSubscriptionAssignees(tx, &creatorID)
	}
	if req.GetUnassigned() {
		if len(req.GetAssigneeIds()) > 0 {
			return nil, invalidArgument("An unassigned task cannot have an owner.")
		}
		return nil, nil
	}
	if len(req.GetAssigneeIds()) == 0 {
		return factory.DefaultSubscriptionAssignees(tx, &creatorID)
	}
	owners := make([]uuid.UUID, 0, len(req.GetAssigneeIds()))
	for _, raw := range req.GetAssigneeIds() {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, invalidArgument("The owner ID is invalid.")
		}
		owners = append(owners, id)
	}
	return owners, nil
}

func validateRefinementModel(tx *gorm.DB, factory *models.Factory, model string) error {
	if model == "" {
		return nil
	}
	if !factory.PlanningEnabled {
		return invalidArgument("Model selection requires Planning.")
	}
	canvases, err := factory.ListCanvases(tx)
	if err != nil {
		return err
	}
	ids := make([]uuid.UUID, 0, len(canvases))
	for _, canvas := range canvases {
		if canvas.LiveVersionID != nil {
			ids = append(ids, canvas.ID)
		}
	}
	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(tx, ids)
	if err != nil {
		return err
	}
	for _, spec := range specs {
		if !models.IsBacklogFactoryApp(spec.Nodes, spec.Edges) {
			continue
		}
		for _, node := range spec.Nodes {
			if node.ID != backlogRefinementNodeID {
				continue
			}
			accepted, err := models.RunnerAcceptsModel(tx, factory.OrganizationID, &factory.ID, node.ComponentName(), node.Configuration, model)
			if err != nil {
				return err
			}
			if accepted {
				return nil
			}
		}
	}
	return invalidArgument("The refinement agent cannot use this model.")
}
