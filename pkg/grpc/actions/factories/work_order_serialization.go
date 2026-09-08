package factories

import (
	"context"

	"github.com/google/uuid"
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

	serialized, err := serializeWorkOrder(
		factory,
		order,
		dispatchesByOrderID[order.ID],
		creatorAutomations[order.ID],
		usageByOrder[order.ID],
	)
	if err != nil {
		return nil, err
	}
	if err := attachWorkOrderFiles(ctx, db, serialized, order.ID); err != nil {
		return nil, err
	}
	return serialized, nil
}

func loadAndSerializeWorkOrders(ctx context.Context, factory *models.Factory, orders []models.FactoryWorkOrder) ([]*pb.WorkOrder, error) {
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

	result := make([]*pb.WorkOrder, len(orders))
	for i := range orders {
		serialized, err := serializeWorkOrder(
			factory,
			&orders[i],
			dispatchesByOrderID[orders[i].ID],
			creatorAutomations[orders[i].ID],
			usageByOrder[orders[i].ID],
		)
		if err != nil {
			return nil, err
		}
		result[i] = serialized
	}

	return result, nil
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
