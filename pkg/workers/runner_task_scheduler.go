package workers

import (
	"context"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	runnercontrol "github.com/superplanehq/superplane/pkg/runners/control"
)

const runnerTaskSchedulerBatchSize = 100

// RunnerTaskScheduler schedules organizations independently from runner sockets.
// Database state is authoritative; every scan observes committed creations,
// cancellations, completions, runner cleanup, and billing plan changes.
type RunnerTaskScheduler struct {
	interval time.Duration
	after    uuid.UUID
}

func NewRunnerTaskScheduler(interval time.Duration) *RunnerTaskScheduler {
	return &RunnerTaskScheduler{interval: interval}
}

func (w *RunnerTaskScheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		if err := w.Process(ctx); err != nil {
			log.WithError(err).Error("Failed to schedule runner tasks")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *RunnerTaskScheduler) Process(ctx context.Context) error {
	db := database.DB(ctx)
	ids, err := models.ListPendingRunnerTaskOrganizations(db, w.after, runnerTaskSchedulerBatchSize)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		w.after = id
		scheduleCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		tasks, err := models.ScheduleOrganizationRunnerTasks(database.DB(scheduleCtx), id, runnerTaskSchedulerBatchSize)
		cancel()
		if err != nil {
			log.WithError(err).WithField("organization_id", id).Error("Failed to schedule organization runner tasks")
			continue
		}
		fleets := make(map[uuid.UUID]struct{})
		for i := range tasks {
			fleets[tasks[i].FleetID] = struct{}{}
			messages.PublishPlanningBoardStatusForTask(db, &tasks[i])
		}
		// Notifications follow commit and use fleet UUIDs, as the gateway expects.
		for fleetID := range fleets {
			if err := runnercontrol.Publish(runnercontrol.Notification{FleetID: fleetID.String()}); err != nil {
				log.WithError(err).WithField("fleet_id", fleetID).Warn("Failed to notify idle runners")
			}
		}
	}
	if len(ids) < runnerTaskSchedulerBatchSize {
		w.after = uuid.Nil
	}
	return nil
}
