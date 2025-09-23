package workers

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type EventSourceScheduleWorker struct {
	nowFunc func() time.Time
}

func NewEventSourceScheduleWorker(nowFunc func() time.Time) (*EventSourceScheduleWorker, error) {
	if nowFunc == nil {
		nowFunc = time.Now
	}

	return &EventSourceScheduleWorker{nowFunc: nowFunc}, nil
}

func (w *EventSourceScheduleWorker) Start() {
	for {
		err := w.Tick()
		if err != nil {
			log.Errorf("Error processing scheduled event sources: %v", err)
		}

		time.Sleep(time.Minute)
	}
}

func (w *EventSourceScheduleWorker) Tick() error {
	eventSources, err := models.ListDueScheduledEventSources()
	if err != nil {
		return fmt.Errorf("failed to list due scheduled event sources: %v", err)
	}

	for _, eventSource := range eventSources {
		err := w.ProcessScheduledEventSource(eventSource)
		if err != nil {
			log.Errorf("Error processing scheduled event source %s: %v", eventSource.ID, err)
		}
	}

	return nil
}

func (w *EventSourceScheduleWorker) ProcessScheduledEventSource(eventSource models.EventSource) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		event, err := models.CreateEventInTransaction(
			tx,
			eventSource.ID,
			eventSource.CanvasID,
			eventSource.Name,
			models.SourceTypeEventSource,
			"scheduled",
			[]byte(`{}`),
			[]byte(`{}`),
		)
		if err != nil {
			return fmt.Errorf("failed to create scheduled event: %v", err)
		}

		if eventSource.Schedule == nil {
			return fmt.Errorf("event source %s has no schedule configured", eventSource.ID)
		}

		scheduleData := eventSource.Schedule.Data()
		next, err := scheduleData.CalculateNextTrigger(w.nowFunc())
		if err != nil {
			return fmt.Errorf("failed to calculate next trigger: %v", err)
		}

		err = eventSource.UpdateNextTriggerInTransaction(tx, *next)
		if err != nil {
			return fmt.Errorf("failed to update next trigger: %v", err)
		}

		err = messages.NewEventCreatedMessage(eventSource.CanvasID.String(), event).Publish()
		if err != nil {
			log.Errorf("Failed to publish event created message for scheduled event %s: %v", event.ID, err)
		}

		logging.ForEventSource(&eventSource).Infof("New event %s - next trigger: %v", event.ID, next)
		return nil
	})
}
