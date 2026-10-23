package workers

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

func (w *RunFinalizer) startTaskAdmissionLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.sweepQueuedTasks(ctx)
		}
	}
}

func (w *RunFinalizer) sweepQueuedTasks(ctx context.Context) {
	db := database.DB(ctx)
	organizations, err := models.ListQueuedTaskOrganizationIDs(db)
	if err != nil {
		w.logger.WithError(err).Error("Failed to list queued task organizations")
		return
	}
	for _, organizationID := range organizations {
		if ctx.Err() != nil {
			return
		}
		logger := w.logger.WithField("organization_id", organizationID)
		var admitted []*models.FactoryLineStepResult
		err := db.Transaction(func(tx *gorm.DB) error {
			locked, err := models.TryLockOrganizationTaskAdmission(tx, organizationID)
			if err != nil || !locked {
				return err
			}
			admitted, err = models.AdmitQueuedForOrganization(tx, organizationID)
			return err
		})
		if err != nil {
			logger.WithError(err).Error("Failed to admit queued tasks")
			continue
		}
		publishTaskAdmissions(logger, admitted)
	}
}

func publishTaskAdmissions(logger *log.Entry, admitted []*models.FactoryLineStepResult) {
	for _, result := range admitted {
		var factoryID, orderID string
		if result.Execution != nil {
			factoryID, orderID = result.Execution.FactoryID.String(), result.Execution.WorkOrderID.String()
		} else if result.QueueItem != nil {
			factoryID, orderID = result.QueueItem.FactoryID.String(), result.QueueItem.WorkOrderID.String()
		}
		logger := logger.WithFields(log.Fields{"factory_id": factoryID, "order_id": orderID})
		if result.Run != nil {
			if err := messages.NewCanvasRunMessage(result.Run.WorkflowID.String(), result.Run.ID.String()).PublishPending(); err != nil {
				logger.WithError(err).WithField("run_id", result.Run.ID).Warn("Failed to publish admitted run")
			}
		}
		if err := messages.PublishFactoryWorkOrderUpdated(factoryID, orderID, factoryevents.EventTypeLineStepExecutionCreated); err != nil {
			logger.WithError(err).Warn("Failed to publish admitted work order")
		}
	}
}
