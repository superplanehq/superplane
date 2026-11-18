package contexts

import (
	"errors"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/components/factory"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
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

// StartTaskColumnCustomAutomation starts an Implement or Done automation for one task.
func StartTaskColumnCustomAutomation(tx *gorm.DB, factoryModel *models.Factory, canvas *models.Canvas, order *models.FactoryWorkOrder, automatic bool) (*models.CanvasEvent, error) {
	column, message, ok := taskColumnAutomation(canvas, factoryModel, order)
	if !ok {
		return nil, grpcerrors.FailedPrecondition(errors.New("task is not in this column"), "The task is not in this column.")
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
		dedupKey, err := taskColumnDedupKey(db, order, column)
		if err != nil {
			return grpcerrors.FailedPrecondition(err, message)
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
			if err := events.EmitDeduplicated(factory.OnWorkOrderPayloadType, payload, dedupKey); err != nil {
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

// EmitAutomaticTaskColumnCustomAutomations starts every automatic automation on one column.
// A task that is not in the column returns no events.
func EmitAutomaticTaskColumnCustomAutomations(tx *gorm.DB, factoryModel *models.Factory, order *models.FactoryWorkOrder, column string) ([]models.CanvasEvent, error) {
	if factoryModel == nil || order == nil {
		return nil, nil
	}
	if column != models.CanvasColumnKeyImplement && column != models.CanvasColumnKeyDone {
		return nil, nil
	}
	canvases, err := factoryModel.ListCanvases(tx)
	if err != nil {
		return nil, err
	}
	live := make([]models.Canvas, 0, len(canvases))
	ids := make([]uuid.UUID, 0, len(canvases))
	for i := range canvases {
		if canvases[i].LiveVersionID == nil || canvases[i].ColumnKey == nil || *canvases[i].ColumnKey != column {
			continue
		}
		live = append(live, canvases[i])
		ids = append(ids, canvases[i].ID)
	}
	if len(live) == 0 {
		return nil, nil
	}
	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(tx, ids)
	if err != nil {
		return nil, err
	}
	emitted := []models.CanvasEvent{}
	for i := range live {
		spec, ok := specs[live[i].ID]
		if !ok || models.FactoryAppTemplateID(spec.Nodes) != models.FactoryAppTemplateCustomAutomationID || !customAutomationSpecIsAutomatic(spec.Nodes) {
			continue
		}
		event, err := StartTaskColumnCustomAutomation(tx, factoryModel, &live[i], order, true)
		if err != nil {
			if code, _, ok := grpcerrors.HandlerStatus(err); ok && code == codes.FailedPrecondition {
				return emitted, nil
			}
			log.WithError(err).Warnf("failed to start %s custom automation %s for work order %s", column, live[i].ID, order.ID)
			continue
		}
		if event != nil {
			emitted = append(emitted, *event)
		}
	}
	return emitted, nil
}

func taskColumnAutomation(canvas *models.Canvas, factoryModel *models.Factory, order *models.FactoryWorkOrder) (string, string, bool) {
	if factoryModel == nil || canvas == nil || canvas.FactoryID == nil || *canvas.FactoryID != factoryModel.ID || canvas.ColumnKey == nil || order == nil {
		return "", "", false
	}
	switch *canvas.ColumnKey {
	case models.CanvasColumnKeyImplement:
		return models.CanvasColumnKeyImplement, "The task is not in Implement.", true
	case models.CanvasColumnKeyDone:
		return models.CanvasColumnKeyDone, "The task is not in Done.", true
	default:
		return "", "", false
	}
}

func taskColumnDedupKey(tx *gorm.DB, order *models.FactoryWorkOrder, column string) (string, error) {
	switch column {
	case models.CanvasColumnKeyImplement:
		return implementColumnDedupKey(tx, order)
	case models.CanvasColumnKeyDone:
		if order.State != models.FactoryWorkOrderStateClosed || order.Result == models.FactoryWorkOrderResultArchived {
			return "", errors.New("task is not in Done")
		}
		return order.ID.String() + "|" + order.UpdatedAt.UTC().Format(time.RFC3339Nano), nil
	default:
		return "", errors.New("task is not in this column")
	}
}

func implementColumnDedupKey(tx *gorm.DB, order *models.FactoryWorkOrder) (string, error) {
	if order.State != models.FactoryWorkOrderStateOpen {
		return "", errors.New("task is not in Implement")
	}
	inVerify, err := models.WorkOrderIsWaitingAfterPassedStage(tx, order.ID)
	if err != nil {
		return "", err
	}
	if inVerify {
		return "", errors.New("task is not in Implement")
	}
	grouped, err := models.ListWorkOrderLineDispatchesByWorkOrderIDs(tx, []uuid.UUID{order.ID})
	if err != nil {
		return "", err
	}
	dispatches := grouped[order.ID]
	if len(dispatches) == 0 {
		return "", errors.New("task is not in Implement")
	}
	current := dispatches[len(dispatches)-1]
	step, ok := currentImplementStep(current)
	if !ok || step != 0 {
		return "", errors.New("task is not in Implement")
	}
	return current.ID.String(), nil
}

func currentImplementStep(record models.FactoryWorkOrderLineDispatchRecord) (int, bool) {
	if record.QueueItem != nil {
		return record.QueueItem.StepIndex, true
	}
	if len(record.Executions) == 0 {
		return 0, false
	}
	active := -1
	furthest := record.Executions[0].FactoryWorkOrderExecution
	for i := range record.Executions {
		step := record.Executions[i].FactoryWorkOrderExecution
		if step.Status == models.FactoryWorkOrderExecutionStatusPending || step.Status == models.FactoryWorkOrderExecutionStatusRunning {
			active = step.StepIndex
		}
		if step.StepIndex > furthest.StepIndex || (step.StepIndex == furthest.StepIndex && step.UpdatedAt.After(furthest.UpdatedAt)) {
			furthest = step
		}
	}
	if active >= 0 {
		return active, true
	}
	return furthest.StepIndex, true
}

func customAutomationSpecIsAutomatic(nodes []models.Node) bool {
	for _, node := range nodes {
		template, _ := node.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
		if template["startMode"] == "automatic" {
			return true
		}
		kinds, _ := template["triggerKinds"].([]any)
		for _, kind := range kinds {
			if kind == "automatic" {
				return true
			}
		}
	}
	return false
}
