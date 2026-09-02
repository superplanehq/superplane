package workers

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	planningSessionCleanupEvery = 15 * time.Second
	planningSessionCleanupLimit = 50
)

// PlanningSessionCleanupWorker ends planning sessions whose heartbeat stopped
// and cancels the hidden canvas run so the fleet machine stops.
type PlanningSessionCleanupWorker struct {
	logger *log.Entry
	now    func() time.Time
}

func NewPlanningSessionCleanupWorker() *PlanningSessionCleanupWorker {
	return &PlanningSessionCleanupWorker{
		logger: log.WithFields(log.Fields{"worker": "PlanningSessionCleanupWorker"}),
		now:    time.Now,
	}
}

func (w *PlanningSessionCleanupWorker) Start(ctx context.Context) {
	w.tick(ctx)

	ticker := time.NewTicker(planningSessionCleanupEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *PlanningSessionCleanupWorker) tick(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	db := database.DB(ctx)
	sessions, err := models.ListStaleOpenPlanningSessions(db, w.now(), planningSessionCleanupLimit)
	if err != nil {
		w.logger.WithError(err).Error("failed to list stale planning sessions")
		return
	}

	for i := range sessions {
		session := &sessions[i]
		if err := session.End(db); err != nil {
			w.logger.WithError(err).Warnf("failed to end stale planning session %s", session.ID)
			continue
		}
		if session.CanvasID == nil || session.CanvasRunID == nil {
			continue
		}
		canvas, err := models.FindCanvasInTransaction(db, session.OrganizationID, *session.CanvasID)
		if err != nil {
			w.logger.WithError(err).Warnf("failed to load planning session canvas %s", session.CanvasID)
			continue
		}
		cancelCtx := authentication.SetUserIdInMetadata(ctx, session.CreatedByUserID.String())
		if _, err := canvases.CancelRun(cancelCtx, db, canvas, *session.CanvasRunID); err != nil {
			w.logger.WithError(err).Warnf("failed to cancel planning session run %s", session.CanvasRunID)
		}
	}
}
