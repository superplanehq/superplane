package workers

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/storedfiles"
)

type FileCleanupWorker struct {
	logger              *log.Entry
	maxResourcesPerTick int
}

func NewFileCleanupWorker() *FileCleanupWorker {
	return &FileCleanupWorker{
		logger:              log.WithFields(log.Fields{"worker": "FileCleanupWorker"}),
		maxResourcesPerTick: 200,
	}
}

func (w *FileCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.cleanupStaleUploads(); err != nil {
				w.logger.Errorf("Error cleaning stale file uploads: %v", err)
			}
		}
	}
}

func (w *FileCleanupWorker) cleanupStaleUploads() error {
	before := time.Now().Add(-models.StalePendingFileAge)
	files, err := models.ListStalePendingFiles(database.Conn(), before, w.maxResourcesPerTick)
	if err != nil {
		return err
	}
	provider := blob.Current()
	ctx := context.Background()
	for i := range files {
		if err := storedfiles.DeleteObjectAndRow(ctx, database.Conn(), provider, &files[i]); err != nil {
			return fmt.Errorf("delete stale file %s: %w", files[i].ID, err)
		}
	}
	if len(files) > 0 {
		w.logger.Infof("Deleted %d stale pending files", len(files))
	}
	return nil
}
