package models

import (
	"fmt"
	"strings"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	StageEventStatePending   = "pending"
	StageEventStateWaiting   = "waiting"
	StageEventStateProcessed = "processed"

	StageEventStateReasonApproval   = "approval"
	StageEventStateReasonTimeWindow = "time-window"
	StageEventStateReasonExecution  = "execution"
	StageEventStateReasonConnection = "connection"
	StageEventStateReasonCancelled  = "cancelled"
	StageEventStateReasonUnhealthy  = "unhealthy"
	StageEventStateReasonStuck      = "stuck"
	StageEventStateReasonTimeout    = "timeout"
	StageEventStateReasonEmpty      = ""
)

var (
	ErrEventAlreadyApprovedByRequester = fmt.Errorf("event already approved by requester")
	ErrEventAlreadyCancelled           = fmt.Errorf("event already cancelled")
	ErrEventCannotBeCancelled          = fmt.Errorf("event cannot be cancelled")
)

type StageEvent struct {
	ID          uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Name        string
	StageID     uuid.UUID
	EventID     uuid.UUID
	SourceID    uuid.UUID
	SourceName  string
	SourceType  string
	State       string
	StateReason string
	CreatedAt   *time.Time
	CancelledBy *uuid.UUID
	CancelledAt *time.Time
	Inputs      datatypes.JSONType[map[string]any]
}

func (e *StageEvent) UpdateState(state, reason string) error {
	return e.UpdateStateInTransaction(database.Conn(), state, reason)
}

func (e *StageEvent) UpdateStateInTransaction(tx *gorm.DB, state, reason string) error {
	return tx.Model(e).
		Clauses(clause.Returning{}).
		Update("state", state).
		Update("state_reason", reason).
		Error
}

func UpdateStageEventsInTransaction(tx *gorm.DB, ids []string, state, reason string) error {
	return tx.Table("stage_events").
		Where("id IN ?", ids).
		Update("state", state).
		Update("state_reason", reason).
		Error
}

func (e *StageEvent) Approve(requesterID uuid.UUID) error {
	now := time.Now()

	approval := StageEventApproval{
		StageEventID: e.ID,
		ApprovedAt:   &now,
		ApprovedBy:   &requesterID,
	}

	err := database.Conn().Create(&approval).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return ErrEventAlreadyApprovedByRequester
		}

		return err
	}

	return nil
}

func (e *StageEvent) Cancel(requesterID uuid.UUID) error {
	if e.StateReason == StageEventStateReasonCancelled {
		return ErrEventAlreadyCancelled
	}

	if e.State == StageEventStateProcessed {
		return ErrEventCannotBeCancelled
	}

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		execution, err := FindExecutionByStageEventID(e.ID)
		if err != nil && !strings.Contains(err.Error(), "record not found") {
			return err
		}

		if execution != nil && execution.State != ExecutionFinished && execution.State != ExecutionCancelled {
			execution.State = ExecutionCancelled
			err = tx.Save(execution).Error
			if err != nil {
				return err
			}
		}

		err = e.UpdateStateInTransaction(tx, StageEventStateProcessed, StageEventStateReasonCancelled)
		if err != nil {
			return err
		}

		now := time.Now()
		err = tx.Model(e).
			Clauses(clause.Returning{}).
			Update("cancelled_by", requesterID).
			Update("cancelled_at", now).
			Error
		if err != nil {
			return err
		}

		e.CancelledBy = &requesterID
		e.CancelledAt = &now

		return nil
	})
}

func (e *StageEvent) FindApprovals() ([]StageEventApproval, error) {
	var approvals []StageEventApproval
	err := database.Conn().
		Where("stage_event_id = ?", e.ID).
		Find(&approvals).
		Error

	if err != nil {
		return nil, err
	}

	return approvals, nil
}

func FindStageEventByID(id, stageID string) (*StageEvent, error) {
	var event StageEvent

	err := database.Conn().
		Where("id = ?", id).
		Where("stage_id = ?", stageID).
		First(&event).
		Error

	if err != nil {
		return nil, err
	}

	return &event, nil
}

func CreateStageEvent(stageID uuid.UUID, event *Event, state, stateReason string, inputs map[string]any, name string) (*StageEvent, error) {
	return CreateStageEventInTransaction(database.Conn(), stageID, event, state, stateReason, inputs, name)
}

func CreateStageEventInTransaction(tx *gorm.DB, stageID uuid.UUID, event *Event, state, stateReason string, inputs map[string]any, name string) (*StageEvent, error) {
	now := time.Now()
	stageEvent := StageEvent{
		StageID:     stageID,
		EventID:     event.ID,
		SourceID:    event.SourceID,
		SourceName:  event.SourceName,
		SourceType:  event.SourceType,
		State:       state,
		StateReason: stateReason,
		CreatedAt:   &now,
		Inputs:      datatypes.NewJSONType(inputs),
		Name:        name,
	}

	err := tx.Create(&stageEvent).
		Clauses(clause.Returning{}).
		Error

	if err != nil {
		return nil, err
	}

	return &stageEvent, nil
}

func FindOldestPendingStageEvent(stageID uuid.UUID) (*StageEvent, error) {
	var event StageEvent

	err := database.Conn().
		Where("state = ?", StageEventStatePending).
		Where("stage_id = ?", stageID).
		Order("created_at ASC").
		First(&event).
		Error

	if err != nil {
		return nil, err
	}

	return &event, nil
}

func FindStagesWithPendingEvents() ([]uuid.UUID, error) {
	var stageIDs []uuid.UUID

	err := database.Conn().
		Table("stage_events").
		Distinct("stage_id").
		Where("state = ?", StageEventStatePending).
		Find(&stageIDs).
		Error

	if err != nil {
		return nil, err
	}

	return stageIDs, nil
}

type StageEventWithConditions struct {
	ID         uuid.UUID
	StageID    uuid.UUID
	Conditions datatypes.JSONSlice[StageCondition]
}

func FindStageEventsWaitingForTimeWindow() ([]StageEventWithConditions, error) {
	var events []StageEventWithConditions

	err := database.Conn().
		Table("stage_events AS e").
		Joins("INNER JOIN stages AS s ON e.stage_id = s.id").
		Select("e.id, e.stage_id, s.conditions").
		Where("e.state = ?", StageEventStateWaiting).
		Where("e.state_reason = ?", StageEventStateReasonTimeWindow).
		Find(&events).
		Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func BulkListStageEventsByCanvasIDAndMultipleStages(canvasID uuid.UUID, stageIDs []uuid.UUID, limitPerStage int, before *time.Time, states []string, stateReasons []string, executionStates []string, executionResults []string) (map[string][]StageEvent, error) {
	if len(stageIDs) == 0 {
		return map[string][]StageEvent{}, nil
	}

	var events []StageEvent

	if len(states) == 0 {
		states = []string{
			StageEventStatePending,
			StageEventStateWaiting,
			StageEventStateProcessed,
		}
	}

	query := database.Conn().
		Where("stage_id IN ?", stageIDs).
		Where("state IN ?", states)

	if len(stateReasons) > 0 {
		query = query.Where("state_reason IN ?", stateReasons)
	}

	if len(executionStates) > 0 || len(executionResults) > 0 {
		query = query.
			Joins("INNER JOIN stage_executions AS ex ON ex.stage_event_id = se.id")
	}

	if len(executionStates) > 0 {
		query = query.Where("ex.state IN ?", executionStates)
	}

	if len(executionResults) > 0 {
		query = query.Where("ex.result IN ?", executionResults)
	}

	if before != nil {
		query = query.Where("created_at < ?", before)
	}

	query = query.Order("stage_id, created_at DESC")

	err := query.Find(&events).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string][]StageEvent)
	stageCounters := make(map[string]int)

	for _, event := range events {
		stageKey := event.StageID.String()

		if limitPerStage > 0 {
			if stageCounters[stageKey] >= limitPerStage {
				continue
			}
			stageCounters[stageKey]++
		}

		result[stageKey] = append(result[stageKey], event)
	}

	return result, nil
}
