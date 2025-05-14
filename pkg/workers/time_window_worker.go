package workers

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
)

type TimeWindowWorker struct {
	nowFunc func() time.Time
}

func NewTimeWindowWorker(nowFunc func() time.Time) (*TimeWindowWorker, error) {
	if nowFunc == nil {
		return nil, fmt.Errorf("nowFunc is required")
	}

	return &TimeWindowWorker{nowFunc: nowFunc}, nil
}

func (w *TimeWindowWorker) Start() {
	for {
		err := w.Tick()
		if err != nil {
			log.Errorf("Error processing events: %v", err)
		}

		time.Sleep(time.Minute)
	}
}

func (w *TimeWindowWorker) Tick() error {
	events, err := models.FindStageEventsWaitingForTimeWindow()
	if err != nil {
		return err
	}

	for _, event := range events {
		err := w.ProcessEvent(event)
		if err != nil {
			log.Errorf("Error processing event %s: %v", event.ID, err)
		}
	}

	return nil
}

func (w *TimeWindowWorker) ProcessEvent(e models.StageEventWithConditions) error {
	condition, err := w.findTimeWindowCondition(e.Conditions)
	if err != nil {
		return err
	}

	now := w.nowFunc()
	err = condition.Evaluate(&now)
	if err != nil {
		log.Infof("Event %s is not within time window - %v", e.ID, err.Error())
		return nil
	}

	event, err := models.FindStageEventByID(e.ID.String(), e.StageID.String())
	if err != nil {
		return err
	}

	log.Infof("Event %s is within time window - moving to pending state", e.ID)
	return event.UpdateState(models.StageEventStatePending, "")
}

func (w *TimeWindowWorker) findTimeWindowCondition(conditions []models.StageCondition) (*models.TimeWindowCondition, error) {
	for _, condition := range conditions {
		if condition.Type == models.StageConditionTypeTimeWindow {
			return condition.TimeWindow, nil
		}
	}

	return nil, fmt.Errorf("time window condition not found")
}
