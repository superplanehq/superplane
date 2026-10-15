package workers

import (
	"context"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const systemSkillSyncBatchSize = 100
const systemSkillSyncInterval = time.Hour

// SystemSkillSyncWorker applies new supplied instructions without replacing user edits.
type SystemSkillSyncWorker struct{ logger *log.Entry }

func NewSystemSkillSyncWorker() *SystemSkillSyncWorker {
	return &SystemSkillSyncWorker{logger: log.WithField("worker", "SystemSkillSyncWorker")}
}

func (w *SystemSkillSyncWorker) Start(ctx context.Context) {
	w.tick(ctx)
	ticker := time.NewTicker(systemSkillSyncInterval)
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

// Process scans every eligible organization. Concurrent workers may safely overlap.
func (w *SystemSkillSyncWorker) Process(ctx context.Context, db *gorm.DB) (int, error) {
	db = db.WithContext(ctx)
	after := uuid.Nil
	count := 0
	for {
		resources, err := models.ListSystemSkillsForSync(db, after, systemSkillSyncBatchSize)
		if err != nil {
			return count, err
		}
		if len(resources) == 0 {
			return count, nil
		}
		for i := range resources {
			resource := &resources[i]
			after = resource.ID
			template, ok := runner.SystemSkillTemplateByID(resource.Config.Data().SystemSkill.TemplateID)
			if !ok {
				continue
			}
			updated, err := resource.UpdateSystemSkillIfUnmodified(db, template.Version, template.Markdown(resource.Name))
			if err != nil {
				w.logger.WithError(err).WithField("resource_id", resource.ID).Error("Failed to update system skill")
				continue
			}
			if updated {
				count++
			}
		}
	}
}

func (w *SystemSkillSyncWorker) tick(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	count, err := w.Process(ctx, database.Conn())
	if err != nil {
		w.logger.WithError(err).Error("System skill sync failed")
		return
	}
	if count > 0 {
		w.logger.WithField("updated", count).Info("Updated system skills")
	}
}
