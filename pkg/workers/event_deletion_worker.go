package workers

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	WorkerDefaultInterval         = 24 * time.Hour
	DefaultEventRetentionDuration = 3 * 30 * 24 * time.Hour // ~3 months
)

type EventDeletionWorker struct {
	RetentionDuration time.Duration
}

func NewEventDeletionWorker() *EventDeletionWorker {
	return &EventDeletionWorker{
		RetentionDuration: DefaultEventRetentionDuration,
	}
}

func (w *EventDeletionWorker) Start() {
	for {
		err := w.Tick()
		if err != nil {
			log.Errorf("Error processing event deletion: %v", err)
		}

		// Run once per day
		time.Sleep(WorkerDefaultInterval)
	}
}

func (w *EventDeletionWorker) Tick() error {
	log.Info("Starting event deletion worker")

	states := []string{models.EventStateRejected}

	deletedCount, err := models.DeleteOldEvents(w.RetentionDuration, states)
	if err != nil {
		log.Errorf("Error deleting old events: %v", err)
		return err
	}

	if deletedCount > 0 {
		log.Infof("Successfully deleted %d old rejected events older than %v", deletedCount, w.RetentionDuration)
	} else {
		log.Info("No old rejected events to delete")
	}

	return nil
}
