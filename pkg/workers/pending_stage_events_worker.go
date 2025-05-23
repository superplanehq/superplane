package workers

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type PendingStageEventsWorker struct {
	nowFunc func() time.Time
}

func NewPendingStageEventsWorker(nowFunc func() time.Time) (*PendingStageEventsWorker, error) {
	if nowFunc == nil {
		return nil, fmt.Errorf("nowFunc is required")
	}

	return &PendingStageEventsWorker{nowFunc: nowFunc}, nil
}

func (w *PendingStageEventsWorker) Start() {
	for {
		err := w.Tick()
		if err != nil {
			log.Errorf("Error processing pending events: %v", err)
		}

		time.Sleep(time.Second)
	}
}

func (w *PendingStageEventsWorker) Tick() error {
	//
	// We need to process each stage with pending events separately.
	// So first, we find all the stages with pending events in their queue.
	//
	stageIDs, err := models.FindStagesWithPendingEvents()
	if err != nil {
		return fmt.Errorf("error listing pending stage events: %v", err)
	}

	//
	// We process each stage individually.
	//
	for _, stageID := range stageIDs {
		err := w.ProcessStage(stageID)
		if err != nil {
			return fmt.Errorf("error processing events for stage %s: %v", stageID, err)
		}
	}

	return nil
}

func (w *PendingStageEventsWorker) ProcessStage(stageID uuid.UUID) error {
	stage, err := models.FindStageByID(stageID.String())
	if err != nil {
		return fmt.Errorf("error finding stage")
	}

	//
	// For each stage, we are only interested in the oldest pending event.
	//
	event, err := models.FindOldestPendingStageEvent(stageID)
	if err != nil {
		return fmt.Errorf("error listing pending events for stage")
	}

	return w.ProcessEvent(stage, event)
}

func (w *PendingStageEventsWorker) ProcessEvent(stage *models.Stage, event *models.StageEvent) error {
	logger := logging.ForStageEvent(event)

	//
	// Check if another execution is already in progress.
	// TODO: this could probably be built into the query that we do above.
	//
	_, err := models.FindExecutionInState(event.StageID, []string{
		models.StageExecutionPending,
		models.StageExecutionStarted,
	})

	// TODO: move to waiting state too?
	if err == nil {
		logger.Infof("Another execution is already in progress - skipping %s", event.ID)
		return nil
	}

	//
	// Process all conditions
	//
	for _, condition := range stage.Conditions {
		proceed, err := w.checkCondition(logger, event, condition)
		if err != nil {
			return err
		}

		if !proceed {
			return nil
		}
	}

	//
	// If we get here, we can start an execution for this event.
	//
	var execution *models.StageExecution
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		execution, err = models.CreateStageExecutionInTransaction(tx, stage.ID, event.ID)
		if err != nil {
			return fmt.Errorf("error creating stage execution: %v", err)
		}

		logger.Infof("Created stage execution %s", execution.ID)

		if err := event.UpdateStateInTransaction(tx, models.StageEventStateWaiting, models.StageEventStateReasonExecution); err != nil {
			return fmt.Errorf("error updating event state: %v", err)
		}

		logger.Infof("Stage event %s processed", event.ID)
		return nil
	})

	if err != nil {
		return err
	}

	err = messages.NewExecutionCreatedMessage(stage.CanvasID.String(), execution).Publish()
	if err != nil {
		logging.ForStage(stage).Errorf("failed to publish execution created message: %v", err)
	}

	logging.ForStage(stage).Infof("Started execution %s", execution.ID)
	return nil
}

func (w *PendingStageEventsWorker) checkCondition(logger *log.Entry, event *models.StageEvent, condition models.StageCondition) (bool, error) {
	switch condition.Type {
	case models.StageConditionTypeApproval:
		return w.checkApprovalCondition(logger, event, condition.Approval)
	case models.StageConditionTypeTimeWindow:
		return w.checkTimeWindowCondition(logger, event, condition.TimeWindow)
	default:
		return false, fmt.Errorf("unknown condition type: %s", condition.Type)
	}
}

func (w *PendingStageEventsWorker) checkApprovalCondition(logger *log.Entry, event *models.StageEvent, condition *models.ApprovalCondition) (bool, error) {
	approvals, err := event.FindApprovals()
	if err != nil {
		return false, err
	}

	//
	// The event has the necessary amount of approvals,
	// so we can proceed to the next condition.
	//
	if len(approvals) >= int(condition.Count) {
		logger.Infof("Approval condition met for event %s", event.ID)
		return true, nil
	}

	logger.Infof(
		"Approval condition not met for event %s - %d/%d",
		event.ID,
		len(approvals),
		condition.Count,
	)

	//
	// The event does not have the necessary amount of approvals,
	// so we move it to the waiting state, and do not proceed to the next condition.
	//
	return false, event.UpdateState(
		models.StageEventStateWaiting,
		models.StageEventStateReasonApproval,
	)
}

func (w *PendingStageEventsWorker) checkTimeWindowCondition(logger *log.Entry, event *models.StageEvent, condition *models.TimeWindowCondition) (bool, error) {
	now := w.nowFunc()
	err := condition.Evaluate(&now)

	//
	// If the current time is within the allowed time window, we proceed.
	//
	if err == nil {
		logger.Infof("Time window condition met for event %s", event.ID)
		return true, nil
	}

	logger.Infof("Time window condition not met for event %s - %s", event.ID, err.Error())

	//
	// The current time is not within the time window allowed,
	// so we move it to the waiting state, and do not proceed to the next condition.
	//
	return false, event.UpdateState(
		models.StageEventStateWaiting,
		models.StageEventStateReasonTimeWindow,
	)
}
