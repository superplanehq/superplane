package factories

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	pbFiles "github.com/superplanehq/superplane/pkg/protos/files"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func loadAndSerializeWorkOrder(ctx context.Context, factory *models.Factory, order *models.FactoryWorkOrder) (*pb.WorkOrder, error) {
	db := database.DB(ctx)
	workOrderIDs := []uuid.UUID{order.ID}
	numbers := map[uuid.UUID]int64{order.ID: order.Number}

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return loadWorkOrderAssigneeUsers(db.WithContext(gctx), order)
	})

	var dispatchesByOrderID map[uuid.UUID][]models.FactoryWorkOrderLineDispatchRecord
	var modelsByExecution map[uuid.UUID][]string
	g.Go(func() error {
		var err error
		dispatchesByOrderID, err = models.ListWorkOrderLineDispatchesByWorkOrderIDs(db.WithContext(gctx), workOrderIDs)
		if err != nil {
			return err
		}
		modelsByExecution = loadModelsForDispatches(db.WithContext(gctx), dispatchesByOrderID)
		return nil
	})

	var creatorAutomations map[uuid.UUID]*factoryevents.AutomationRef
	g.Go(func() error {
		var err error
		creatorAutomations, err = models.ResolveFactoryWorkOrderCreatorAutomations(db.WithContext(gctx), []models.FactoryWorkOrder{*order})
		return err
	})

	var usageByOrder map[uuid.UUID]models.UsageTotals
	g.Go(func() error {
		var err error
		usageByOrder, err = models.SumUsageForWorkOrders(db.WithContext(gctx), workOrderIDs)
		return err
	})

	var byModel map[uuid.UUID][]models.UsageByModel
	var byMachineType map[uuid.UUID][]models.UsageByMachineType
	g.Go(func() error {
		byModel, byMachineType = loadWorkOrderUsageBreakdowns(db.WithContext(gctx), workOrderIDs)
		return nil
	})

	var pullRequestsByOrder map[uuid.UUID][]*pb.FactoryPullRequest
	g.Go(func() error {
		var err error
		pullRequestsByOrder, err = loadSerializedPullRequestsByWorkOrderIDs(gctx, db.WithContext(gctx), workOrderIDs, numbers)
		return err
	})

	var checksByOrder map[uuid.UUID][]models.FactoryWorkOrderCheck
	g.Go(func() error {
		var err error
		checksByOrder, err = models.ListChecksForWorkOrders(db.WithContext(gctx), workOrderIDs)
		return err
	})

	var planningByOrder map[uuid.UUID]*pb.PlanningSessionSummary
	g.Go(func() error {
		var err error
		planningByOrder, err = planningSessionSummariesByWorkOrder(db.WithContext(gctx), workOrderIDs)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	serialized, err := serializeWorkOrder(
		factory,
		order,
		dispatchesByOrderID[order.ID],
		creatorAutomations[order.ID],
		workOrderUsageView{
			Totals:            usageByOrder[order.ID],
			ByModel:           byModel[order.ID],
			ByMachineType:     byMachineType[order.ID],
			ModelsByExecution: modelsByExecution,
		},
	)
	if err != nil {
		return nil, err
	}
	serialized.PullRequests = pullRequestsByOrder[order.ID]
	if err := attachWorkOrderFiles(ctx, db, serialized, order.ID); err != nil {
		return nil, err
	}
	checks, err := serializeChecks(checksByOrder[order.ID])
	if err != nil {
		return nil, err
	}
	serialized.Checks = checks
	serialized.PlanningSession = planningByOrder[order.ID]
	return serialized, nil
}

func loadAndSerializeWorkOrders(ctx context.Context, factory *models.Factory, orders []models.FactoryWorkOrder) ([]*pb.WorkOrderSummary, error) {
	if len(orders) == 0 {
		return nil, nil
	}

	workOrderIDs := make([]uuid.UUID, len(orders))
	orderRefs := make([]*models.FactoryWorkOrder, len(orders))
	numbers := make(map[uuid.UUID]int64, len(orders))
	for i := range orders {
		workOrderIDs[i] = orders[i].ID
		orderRefs[i] = &orders[i]
		numbers[orders[i].ID] = orders[i].Number
	}

	db := database.DB(ctx)
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return loadWorkOrderAssigneeUsers(db.WithContext(gctx), orderRefs...)
	})

	var dispatchesByOrderID map[uuid.UUID][]models.FactoryWorkOrderLineDispatchRecord
	g.Go(func() error {
		var err error
		dispatchesByOrderID, err = models.ListWorkOrderLineDispatchesByWorkOrderIDs(db.WithContext(gctx), workOrderIDs)
		return err
	})

	var creatorAutomations map[uuid.UUID]*factoryevents.AutomationRef
	g.Go(func() error {
		var err error
		creatorAutomations, err = models.ResolveFactoryWorkOrderCreatorAutomations(db.WithContext(gctx), orders)
		return err
	})

	var usageByOrder map[uuid.UUID]models.UsageTotals
	g.Go(func() error {
		var err error
		usageByOrder, err = models.SumUsageForWorkOrders(db.WithContext(gctx), workOrderIDs)
		return err
	})

	var pullRequestsByOrder map[uuid.UUID][]*pb.FactoryPullRequest
	g.Go(func() error {
		var err error
		pullRequestsByOrder, err = loadSerializedPullRequestsByWorkOrderIDs(gctx, db.WithContext(gctx), workOrderIDs, numbers)
		return err
	})

	var checksByOrder map[uuid.UUID][]models.FactoryWorkOrderCheck
	g.Go(func() error {
		var err error
		checksByOrder, err = models.ListChecksForWorkOrders(db.WithContext(gctx), workOrderIDs)
		return err
	})

	var planningByOrder map[uuid.UUID]*pb.PlanningSessionSummary
	g.Go(func() error {
		var err error
		planningByOrder, err = planningSessionSummariesByWorkOrder(db.WithContext(gctx), workOrderIDs)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	result := make([]*pb.WorkOrderSummary, len(orders))
	for i := range orders {
		serialized, err := serializeWorkOrderSummary(
			factory,
			&orders[i],
			dispatchesByOrderID[orders[i].ID],
			creatorAutomations[orders[i].ID],
			workOrderUsageView{Totals: usageByOrder[orders[i].ID]},
		)
		if err != nil {
			return nil, err
		}
		serialized.PullRequests = pullRequestsByOrder[orders[i].ID]
		serialized.CheckScores = serializeCheckScores(checksByOrder[orders[i].ID])
		serialized.PlanningSession = planningByOrder[orders[i].ID]
		result[i] = serialized
	}

	return result, nil
}

func planningSessionSummariesByWorkOrder(db *gorm.DB, ids []uuid.UUID) (map[uuid.UUID]*pb.PlanningSessionSummary, error) {
	sessionsByOrder, err := models.ListAnalysisPlanningSessionsForWorkOrders(db, ids)
	if err != nil {
		return nil, err
	}
	sessions := make([]*models.FactoryPlanningSession, 0, len(sessionsByOrder))
	for _, session := range sessionsByOrder {
		sessions = append(sessions, session)
	}
	executionIDs, err := planningSessionExecutionIDs(db, sessions)
	if err != nil {
		return nil, err
	}
	summaries := make(map[uuid.UUID]*pb.PlanningSessionSummary, len(sessionsByOrder))
	for orderID, session := range sessionsByOrder {
		summaries[orderID] = serializePlanningSessionSummary(session, executionIDs[session.ID])
	}
	return summaries, nil
}

// loadWorkOrderAssigneeUsers reloads assignees with User so the API can
// return the owner name. Mutations such as Start rebuild the in-memory
// slice with IDs only.
func loadWorkOrderAssigneeUsers(db *gorm.DB, orders ...*models.FactoryWorkOrder) error {
	ids := make([]uuid.UUID, 0, len(orders))
	byID := make(map[uuid.UUID]*models.FactoryWorkOrder, len(orders))
	for _, order := range orders {
		if order == nil {
			continue
		}
		ids = append(ids, order.ID)
		byID[order.ID] = order
		order.Assignees = nil
	}
	if len(ids) == 0 {
		return nil
	}

	var assignees []models.FactoryWorkOrderAssignee
	if err := db.Preload("User").Where("work_order_id IN ?", ids).Find(&assignees).Error; err != nil {
		return err
	}
	for i := range assignees {
		order := byID[assignees[i].WorkOrderID]
		if order == nil {
			continue
		}
		order.Assignees = append(order.Assignees, assignees[i])
	}
	return nil
}

func attachWorkOrderFiles(ctx context.Context, db *gorm.DB, order *pb.WorkOrder, workOrderID uuid.UUID) error {
	records, err := models.ListReadyTaskFiles(db, workOrderID)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}

	provider := blob.Current()
	files := make([]*pbFiles.File, 0, len(records))
	for i := range records {
		downloadURL, err := storedfiles.DownloadURL(ctx, provider, &records[i], blob.UIDownloadTTL)
		if err != nil {
			return err
		}
		item := &pbFiles.File{
			Id:          records[i].ID.String(),
			Filename:    records[i].Filename,
			ContentType: records[i].ContentType,
			SizeBytes:   records[i].SizeBytes,
			Scope:       records[i].Scope,
			State:       records[i].State,
			DownloadUrl: downloadURL,
			CreatedAt:   timestamppb.New(records[i].CreatedAt),
		}
		if records[i].Checksum != nil {
			item.Checksum = *records[i].Checksum
		}
		files = append(files, item)
	}
	order.Files = files
	return nil
}

func loadWorkOrderUsageBreakdowns(
	db *gorm.DB,
	workOrderIDs []uuid.UUID,
) (map[uuid.UUID][]models.UsageByModel, map[uuid.UUID][]models.UsageByMachineType) {
	byModel, err := models.ListUsageByModelForWorkOrders(db, workOrderIDs)
	if err != nil {
		log.WithError(err).Warnf(
			"work order listing: model usage breakdown unavailable for %d order(s)",
			len(workOrderIDs),
		)
		byModel = map[uuid.UUID][]models.UsageByModel{}
	}
	byMachine, err := models.ListUsageByMachineTypeForWorkOrders(db, workOrderIDs)
	if err != nil {
		log.WithError(err).Warnf(
			"work order listing: compute usage breakdown unavailable for %d order(s)",
			len(workOrderIDs),
		)
		byMachine = map[uuid.UUID][]models.UsageByMachineType{}
	}
	return byModel, byMachine
}

func loadModelsForDispatches(
	db *gorm.DB,
	dispatchesByOrderID map[uuid.UUID][]models.FactoryWorkOrderLineDispatchRecord,
) map[uuid.UUID][]string {
	ids := executionIDsFromDispatches(dispatchesByOrderID)
	modelsByExecution, err := models.ListModelsForWorkOrderExecutions(db, ids)
	if err != nil {
		log.WithError(err).Warnf(
			"work order listing: execution models unavailable for %d execution(s)",
			len(ids),
		)
		return map[uuid.UUID][]string{}
	}
	return modelsByExecution
}

func executionIDsFromDispatches(
	dispatchesByOrderID map[uuid.UUID][]models.FactoryWorkOrderLineDispatchRecord,
) []uuid.UUID {
	ids := make([]uuid.UUID, 0)
	for _, dispatches := range dispatchesByOrderID {
		for _, dispatch := range dispatches {
			for _, execution := range dispatch.Executions {
				ids = append(ids, execution.ID)
			}
		}
	}
	return ids
}
