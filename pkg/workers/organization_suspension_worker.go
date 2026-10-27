package workers

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/runners/control"
	"gorm.io/gorm"
)

type OrganizationSuspensionWorker struct{ agents *agents.Service }

func NewOrganizationSuspensionWorker(service *agents.Service) *OrganizationSuspensionWorker {
	return &OrganizationSuspensionWorker{agents: service}
}

func (w *OrganizationSuspensionWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		w.processPending(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *OrganizationSuspensionWorker) processPending(ctx context.Context) {
	events, err := models.ListPendingOrganizationSuspensions(database.DB(ctx))
	if err != nil {
		log.WithError(err).Error("list pending organization suspensions")
		return
	}
	for _, event := range events {
		if err := w.process(ctx, event); err != nil {
			log.WithError(err).WithField("organization_id", event.OrganizationID).Error("cancel suspended organization work")
		}
	}
}

// The pending event lock serializes competing replicas and prevents an old
// cancellation batch from acting on work created after reactivation.
func (w *OrganizationSuspensionWorker) process(ctx context.Context, event models.OrganizationSuspensionEvent) error {
	db := database.DB(ctx)
	notifier := &RunCancellationNotifier{}
	var notifications []control.Notification
	err := db.Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockPendingOrganizationSuspension(tx, event.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		organization := models.Organization{ID: locked.OrganizationID}
		if err := organization.DiscardSuspendedWork(tx); err != nil {
			return err
		}
		runs, tasks, sessions, err := organization.SuspensionWork(tx)
		if err != nil {
			return err
		}
		for _, run := range runs {
			result, err := run.RequestCancellation(tx, nil)
			if err != nil {
				return err
			}
			notifier.RecordRuns([]*models.RunCancellationResult{result})
		}
		for _, task := range tasks {
			if err := task.RequestCancel(tx, time.Now()); err != nil {
				return err
			}
			if task.RunnerID != nil {
				notifications = append(notifications, control.Notification{RunnerID: task.RunnerID.String()})
			}
		}
		// Keep the suspension event pending until provider interrupts return.
		// Reactivation cannot race with an interrupt from an older suspension.
		for _, session := range sessions {
			if w.agents == nil {
				return errors.New("agent service is required to stop active sessions")
			}
			if err := w.agents.InterruptSession(ctx, session.OrganizationID, session.UserID, session.ID); err != nil {
				return err
			}
		}
		return organization.CompleteSuspensionCancellation(tx, event.ID)
	})
	if err != nil {
		return err
	}
	notifier.Publish()
	for _, notification := range notifications {
		if err := control.Publish(notification); err != nil {
			return err
		}
	}
	return nil
}
