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
)

var (
	ErrEventAlreadyApprovedByRequester = fmt.Errorf("event already approved by requester")
)

type StageEvent struct {
	ID          uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	StageID     uuid.UUID
	EventID     uuid.UUID
	SourceID    uuid.UUID
	SourceName  string
	SourceType  string
	State       string
	StateReason string
	CreatedAt   *time.Time
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

func CreateStageEvent(stageID uuid.UUID, event *Event, state, stateReason string, inputs map[string]any) (*StageEvent, error) {
	return CreateStageEventInTransaction(database.Conn(), stageID, event, state, stateReason, inputs)
}

func CreateStageEventInTransaction(tx *gorm.DB, stageID uuid.UUID, event *Event, state, stateReason string, inputs map[string]any) (*StageEvent, error) {
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
