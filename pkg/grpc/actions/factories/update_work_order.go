package factories

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	workersctx "github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"

	log "github.com/sirupsen/logrus"
)

const (
	workOrderTitleMaxLength       = 256
	workOrderDescriptionMaxLength = 20000
)

func UpdateWorkOrder(
	ctx context.Context,
	organizationID string,
	req *pb.UpdateWorkOrderRequest,
) (*pb.UpdateWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order")
	}

	title, description, err := workOrderContentFromRequest(req)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order")
	}

	db := database.DB(ctx)
	resolvedFactory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order")
	}
	factoryID := resolvedFactory.ID

	resolvedOrder, err := findWorkOrder(db, resolvedFactory, req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order")
	}
	orderID := resolvedOrder.ID
	var bound storedfiles.BindResult
	var planningEvents []models.CanvasEvent
	err = db.Transaction(func(tx *gorm.DB) error {
		factory, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}

		order, err := factory.FindWorkOrder(tx, orderID)
		if err != nil {
			return err
		}

		if err := order.LockForUpdate(tx); err != nil {
			return err
		}
		if req.RefinementModel != nil {
			if err := order.SetRefinementModel(tx, strings.TrimSpace(req.GetRefinementModel())); err != nil {
				return err
			}
		}
		if err := order.UpdateContent(tx, title, description); err != nil {
			return err
		}
		if req.RefinementModel != nil {
			planningEvents, err = restartFailedPlanningAfterModelSave(tx, factory, order)
			if err != nil {
				return err
			}
		}
		if description == nil {
			return nil
		}
		result, bindErr := storedfiles.BindDescriptionFiles(
			ctx,
			tx,
			blob.Current(),
			orgID,
			factory.ID,
			order.ID,
			*description,
		)
		bound = result
		return bindErr
	})
	if delErr := storedfiles.ApplyBindResult(ctx, db, blob.Current(), orgID, factoryID, bound, err); delErr != nil {
		log.WithError(delErr).Warn("Failed to delete file objects after bind")
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order")
	}
	for i := range planningEvents {
		event := &planningEvents[i]
		if err := messages.NewCanvasEventCreatedMessage(event.WorkflowID.String(), orgID.String(), event).Publish(); err != nil {
			log.WithError(err).Warnf("Failed to publish planning event %s", event.ID)
		}
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryID.String(),
		orderID.String(),
		factoryevents.EventTypeOrderUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", orderID)
	}

	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order")
	}

	order, err := factory.FindWorkOrder(db, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order")
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order")
	}

	return &pb.UpdateWorkOrderResponse{Order: serialized}, nil
}

func restartFailedPlanningAfterModelSave(tx *gorm.DB, factory *models.Factory, order *models.FactoryWorkOrder) ([]models.CanvasEvent, error) {
	if order.State != models.FactoryWorkOrderStateDraft || !factory.PlanningEnabled {
		return nil, nil
	}
	_, err := models.FindPlanningSessionByDraftWorkOrder(tx, factory.OrganizationID, factory.ID, order.ID)
	if err == nil {
		return nil, nil
	}
	if !errors.Is(err, models.ErrFactoryPlanningSessionNotFound) {
		return nil, err
	}
	canvases, err := factory.ListCanvases(tx)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(canvases))
	for i := range canvases {
		if canvases[i].LiveVersionID != nil {
			ids = append(ids, canvases[i].ID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(tx, ids)
	if err != nil {
		return nil, err
	}
	backlogIDs := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if spec, ok := specs[id]; ok && models.IsBacklogFactoryApp(spec.Nodes, spec.Edges) {
			backlogIDs = append(backlogIDs, id)
		}
	}
	run, err := order.LatestBacklogRefinementRun(tx, backlogIDs)
	if err != nil || run == nil || run.Result != models.CanvasRunResultFailed {
		return nil, err
	}
	return workersctx.CreateWorkOrderEventsOnCanvas(tx, factory, order, run.WorkflowID)
}

func workOrderContentFromRequest(req *pb.UpdateWorkOrderRequest) (*string, *string, error) {
	if req.Title == nil && req.Description == nil && req.RefinementModel == nil {
		return nil, nil, invalidArgument("title or description is required")
	}

	var title *string
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			return nil, nil, invalidArgument("title is required")
		}
		if utf8.RuneCountInString(trimmed) > workOrderTitleMaxLength {
			return nil, nil, invalidArgument("title is too long")
		}
		title = &trimmed
	}

	var description *string
	if req.Description != nil {
		if utf8.RuneCountInString(*req.Description) > workOrderDescriptionMaxLength {
			return nil, nil, invalidArgument("description is too long")
		}
		value := *req.Description
		description = &value
	}

	return title, description, nil
}
