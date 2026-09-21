package factories

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	pbFiles "github.com/superplanehq/superplane/pkg/protos/files"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func loadAndSerializeWorkOrder(ctx context.Context, factory *models.Factory, order *models.FactoryWorkOrder) (*pb.WorkOrder, error) {
	db := database.DB(ctx)
	if err := loadWorkOrderAssigneeUsers(db, order); err != nil {
		return nil, err
	}

	dispatchesByOrderID, err := models.ListWorkOrderLineDispatchesByWorkOrderIDs(db, []uuid.UUID{order.ID})
	if err != nil {
		return nil, err
	}

	creatorAutomations, err := models.ResolveFactoryWorkOrderCreatorAutomations(db, []models.FactoryWorkOrder{*order})
	if err != nil {
		return nil, err
	}

	usageByOrder, err := models.SumUsageForWorkOrders(db, []uuid.UUID{order.ID})
	if err != nil {
		return nil, err
	}
	byModel, byMachineType := loadWorkOrderUsageBreakdowns(db, []uuid.UUID{order.ID})

	serialized, err := serializeWorkOrder(
		factory,
		order,
		dispatchesByOrderID[order.ID],
		creatorAutomations[order.ID],
		workOrderUsageView{
			Totals:            usageByOrder[order.ID],
			ByModel:           byModel[order.ID],
			ByMachineType:     byMachineType[order.ID],
			ModelsByExecution: loadModelsForDispatches(db, dispatchesByOrderID),
		},
	)
	if err != nil {
		return nil, err
	}
	if err := attachWorkOrderFiles(ctx, db, serialized, order.ID); err != nil {
		return nil, err
	}
	if err := attachDescribedWorkOrderChecks(db, []*pb.WorkOrder{serialized}); err != nil {
		return nil, err
	}
	return serialized, nil
}

func loadAndSerializeWorkOrders(ctx context.Context, factory *models.Factory, orders []models.FactoryWorkOrder) ([]*pb.WorkOrderSummary, error) {
	if len(orders) == 0 {
		return nil, nil
	}

	workOrderIDs := make([]uuid.UUID, len(orders))
	orderRefs := make([]*models.FactoryWorkOrder, len(orders))
	for i := range orders {
		workOrderIDs[i] = orders[i].ID
		orderRefs[i] = &orders[i]
	}

	db := database.DB(ctx)
	if err := loadWorkOrderAssigneeUsers(db, orderRefs...); err != nil {
		return nil, err
	}

	dispatchesByOrderID, err := models.ListWorkOrderLineDispatchesByWorkOrderIDs(db, workOrderIDs)
	if err != nil {
		return nil, err
	}

	creatorAutomations, err := models.ResolveFactoryWorkOrderCreatorAutomations(db, orders)
	if err != nil {
		return nil, err
	}

	usageByOrder, err := models.SumUsageForWorkOrders(db, workOrderIDs)
	if err != nil {
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
		result[i] = serialized
	}

	if err := attachListedWorkOrderCheckScores(db, result); err != nil {
		return nil, err
	}

	return result, nil
}

func attachDescribedWorkOrderChecks(db *gorm.DB, orders []*pb.WorkOrder) error {
	return applyWorkOrderChecks(db, orders, func(order *pb.WorkOrder, checks []models.FactoryWorkOrderCheck) error {
		serialized, err := serializeChecks(checks)
		if err != nil {
			return err
		}
		order.Checks = serialized
		return nil
	})
}

func attachListedWorkOrderCheckScores(db *gorm.DB, orders []*pb.WorkOrderSummary) error {
	return applyWorkOrderChecks(db, orders, func(order *pb.WorkOrderSummary, checks []models.FactoryWorkOrderCheck) error {
		order.CheckScores = serializeCheckScores(checks)
		return nil
	})
}

type identifiedWorkOrder interface {
	GetId() string
}

func applyWorkOrderChecks[T identifiedWorkOrder](
	db *gorm.DB,
	orders []T,
	apply func(T, []models.FactoryWorkOrderCheck) error,
) error {
	ids := make([]uuid.UUID, 0, len(orders))
	for _, order := range orders {
		if any(order) == nil || order.GetId() == "" {
			continue
		}
		id, err := uuid.Parse(order.GetId())
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}

	grouped, err := models.ListChecksForWorkOrders(db, ids)
	if err != nil {
		return err
	}

	for _, order := range orders {
		if any(order) == nil || order.GetId() == "" {
			continue
		}
		id, err := uuid.Parse(order.GetId())
		if err != nil {
			return err
		}
		if err := apply(order, grouped[id]); err != nil {
			return err
		}
	}
	return nil
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
