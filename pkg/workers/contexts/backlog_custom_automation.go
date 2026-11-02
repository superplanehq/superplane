package contexts

import (
	"errors"

	"github.com/superplanehq/superplane/pkg/components/factory"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// StartBacklogCustomAutomation emits task context and stores task authority
// before the event is delivered to a runner.
func StartBacklogCustomAutomation(tx *gorm.DB, factoryModel *models.Factory, canvas *models.Canvas, order *models.FactoryWorkOrder, automatic bool) (*models.CanvasEvent, error) {
	if factoryModel == nil || canvas == nil || canvas.FactoryID == nil || *canvas.FactoryID != factoryModel.ID || canvas.ColumnKey == nil || *canvas.ColumnKey != models.CanvasColumnKeyBacklog || order == nil {
		return nil, grpcerrors.FailedPrecondition(errors.New("task is not in Backlog"), "The task is not in Backlog.")
	}
	node, err := models.FindCanvasNode(tx, canvas.ID, "on-task")
	if err != nil {
		return nil, err
	}
	var created []models.CanvasEvent
	err = tx.Transaction(func(db *gorm.DB) error {
		if err := order.LockForUpdate(db); err != nil {
			return err
		}
		if order.FactoryID != factoryModel.ID || order.OrganizationID != factoryModel.OrganizationID || order.State != models.FactoryWorkOrderStateDraft {
			return grpcerrors.FailedPrecondition(errors.New("task is not in Backlog"), "The task is not in Backlog.")
		}
		if err := order.RequireExecutionOwner(db, true); err != nil {
			return err
		}
		if err := order.ValidateSubscriptionCanvasModels(db, canvas.ID, ""); err != nil {
			return err
		}
		events := NewEventContext(db, node, nil, func(events []models.CanvasEvent) { created = append(created, events...) })
		payload := workOrderCreatedPayload(db, order)
		if automatic {
			entry, err := order.BacklogEntryID(db)
			if err != nil {
				return err
			}
			if err := events.EmitDeduplicated(factory.OnWorkOrderPayloadType, payload, order.ID.String()+"|"+entry); err != nil {
				return err
			}
		} else if err := events.Emit(factory.OnWorkOrderPayloadType, payload); err != nil {
			return err
		}
		for i := range created {
			event, err := models.FindCanvasEventForCanvas(db, canvas.ID, created[i].ID)
			if err != nil {
				return err
			}
			created[i] = *event
			run, err := models.FindCanvasRunInTransaction(db, canvas.ID, created[i].RunID)
			if err != nil {
				return err
			}
			if err := models.MaybeLinkCustomAutomationRun(db, canvas, &created[i], run); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil || len(created) == 0 {
		return nil, err
	}
	return &created[0], nil
}
