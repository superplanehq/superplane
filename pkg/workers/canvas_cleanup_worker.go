package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/workers/cleaners"
)

type CanvasCleanupWorker struct {
	semaphore      *semaphore.Weighted
	logger         *log.Entry
	maxRunsPerTick int
	sessionCleaner agents.ProviderSessionCleaner
	gitProvider    git.Provider
}

func NewCanvasCleanupWorker(gitProvider git.Provider, providers ...agents.Provider) *CanvasCleanupWorker {
	w := &CanvasCleanupWorker{
		semaphore:      semaphore.NewWeighted(25),
		logger:         log.WithFields(log.Fields{"worker": "CanvasCleanupWorker"}),
		maxRunsPerTick: 50,
		gitProvider:    gitProvider,
	}

	if len(providers) > 0 {
		if cleaner, ok := providers[0].(agents.ProviderSessionCleaner); ok {
			w.sessionCleaner = cleaner
		}
	}

	return w
}

func (w *CanvasCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickStart := time.Now()
			canvases, err := models.ListDeletedCanvases()
			if err != nil {
				w.logger.Errorf("Error finding deleted workflows: %v", err)
				continue
			}

			telemetry.RecordWorkflowCleanupWorkerCanvasesCount(context.Background(), len(canvases))

			for _, canvas := range canvases {
				if deletedResourceWithinGracePeriod(canvas.DeletedAt.Time, tickStart) {
					continue
				}

				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(canvas models.Canvas) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessCanvas(canvas); err != nil {
						w.logger.Errorf("Error processing canvas %s: %v", canvas.ID, err)
					}
				}(canvas)
			}

			telemetry.RecordWorkflowCleanupWorkerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

func (w *CanvasCleanupWorker) LockAndProcessCanvas(canvas models.Canvas) error {
	if deletedResourceWithinGracePeriod(canvas.DeletedAt.Time, time.Now()) {
		return nil
	}

	var sessionsToClean []models.AgentSession
	var repositoriesToClean []models.Repository
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		lockedCanvas, err := models.LockCanvas(tx, canvas.ID)
		if err != nil {
			w.logger.Infof("Canvas %s already being processed - skipping", canvas.ID)
			return nil
		}

		w.logger.Infof("Processing deleted canvas %s", lockedCanvas.ID)
		sessions, repositories, err := w.processCanvas(tx, *lockedCanvas)
		if err != nil {
			return err
		}

		sessionsToClean = sessions
		repositoriesToClean = repositories
		return nil
	})
	if err != nil {
		return err
	}

	ctx := context.Background()
	w.cleanupProviderSessions(ctx, sessionsToClean)
	w.cleanupGitRepositories(ctx, repositoriesToClean)
	return nil
}

func (w *CanvasCleanupWorker) cleanupProviderSessions(ctx context.Context, sessions []models.AgentSession) {
	if w.sessionCleaner == nil || len(sessions) == 0 {
		return
	}

	for _, session := range sessions {
		if session.Provider != w.sessionCleaner.Name() {
			w.logger.WithFields(log.Fields{
				"session_id":       session.ID,
				"session_provider": session.Provider,
				"cleaner_provider": w.sessionCleaner.Name(),
			}).Warn("Skipping provider cleanup for agent session with mismatched provider")
			continue
		}

		cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := w.sessionCleaner.DeleteSession(cleanupCtx, session.ProviderSessionID)
		cancel()
		if err != nil {
			w.logger.WithFields(log.Fields{
				"session_id":          session.ID,
				"provider":            session.Provider,
				"provider_session_id": session.ProviderSessionID,
			}).WithError(err).Warn("Failed to cleanup provider agent session")
		}
	}
}

func (w *CanvasCleanupWorker) cleanupGitRepositories(ctx context.Context, repositories []models.Repository) {
	if w.gitProvider == nil || len(repositories) == 0 {
		return
	}

	for _, repository := range repositories {
		if repository.Provider != w.gitProvider.Name() {
			w.logger.WithFields(log.Fields{
				"repository_id":       repository.ID,
				"repository_provider": repository.Provider,
				"git_provider":        w.gitProvider.Name(),
			}).Warn("Skipping repository cleanup for repository with mismatched provider")
			continue
		}

		cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := w.gitProvider.DeleteRepository(cleanupCtx, repository.RepoID)
		cancel()
		if err != nil {
			w.logger.WithFields(log.Fields{
				"repository_id": repository.ID,
				"provider":      repository.Provider,
				"repo_id":       repository.RepoID,
			}).WithError(err).Warn("Failed to cleanup git repository")
		}
	}
}

func (w *CanvasCleanupWorker) processCanvas(tx *gorm.DB, canvas models.Canvas) ([]models.AgentSession, []models.Repository, error) {
	if !canvas.DeletedAt.Valid {
		w.logger.Infof("Skipping non-deleted canvas %s", canvas.ID)
		return nil, nil, nil
	}

	runCleaner, err := cleaners.NewRunCleaner(tx, cleaners.RunCleanerOptions{
		Mode:   cleaners.RunCleanerModeCanvasTeardown,
		Canvas: &canvas,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create run cleaner: %w", err)
	}

	totalRunsDeleted := 0
	for totalRunsDeleted < w.maxRunsPerTick {
		deleted, err := runCleaner.CleanBatch(w.maxRunsPerTick - totalRunsDeleted)
		if err != nil {
			return nil, nil, fmt.Errorf("clean workflow runs: %w", err)
		}

		if deleted == 0 {
			break
		}

		totalRunsDeleted += deleted
	}

	remainingRuns, err := canvas.CountRuns(tx)
	if err != nil {
		return nil, nil, fmt.Errorf("count remaining workflow runs: %w", err)
	}

	if remainingRuns > 0 {
		w.logger.Infof("Partially cleaned runs from canvas %s (deleted %d runs, %d remaining)", canvas.ID, totalRunsDeleted, remainingRuns)
		return nil, nil, nil
	}

	if err := canvas.DeleteRemainingResources(tx); err != nil {
		return nil, nil, fmt.Errorf("delete remaining workflow resources: %w", err)
	}

	if err := tx.Unscoped().Where("workflow_id = ?", canvas.ID).Delete(&models.CanvasNode{}).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to delete canvas nodes: %w", err)
	}

	sessions, err := models.ListAgentSessionsForCanvasInTransaction(tx, canvas.OrganizationID, canvas.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("list canvas agent sessions: %w", err)
	}

	var repositories []models.Repository
	repository, err := models.FindRepositoryInTransaction(tx, canvas.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, fmt.Errorf("find canvas repository: %w", err)
	}

	if repository != nil {
		repositories = append(repositories, *repository)
	}

	if err := models.DeleteAgentSessionsForCanvasInTransaction(tx, canvas.OrganizationID, canvas.ID); err != nil {
		return nil, nil, fmt.Errorf("delete canvas agent sessions: %w", err)
	}

	if err := tx.Unscoped().Delete(&canvas).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to delete canvas: %w", err)
	}

	w.logger.Infof("Successfully cleaned up canvas %s (deleted %d runs)", canvas.ID, totalRunsDeleted)
	return sessions, repositories, nil
}
