package models

import (
	"fmt"
	"slices"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	RunTemplateTypeSemaphore     = "semaphore"
	StageConditionTypeApproval   = "approval"
	StageConditionTypeTimeWindow = "time-window"
)

type Stage struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;"`
	OrganizationID uuid.UUID
	CanvasID       uuid.UUID
	Name           string
	CreatedAt      *time.Time
	CreatedBy      uuid.UUID

	Use         datatypes.JSONType[StageTagUsageDefinition]
	Conditions  datatypes.JSONSlice[StageCondition]
	RunTemplate datatypes.JSONType[RunTemplate]
}

type StageTagUsageDefinition struct {
	From []string             `json:"from"`
	Tags []StageTagDefinition `json:"tags"`
}

type StageTagDefinition struct {
	Name      string `json:"name"`
	ValueFrom string `json:"value_from"`
}

type StageCondition struct {
	Type       string               `json:"type"`
	Approval   *ApprovalCondition   `json:"approval,omitempty"`
	TimeWindow *TimeWindowCondition `json:"time,omitempty"`
}

type TimeWindowCondition struct {
	Start    string   `json:"start"`
	End      string   `json:"end"`
	WeekDays []string `json:"week_days"`
}

func NewTimeWindowCondition(start, end string, days []string) (*TimeWindowCondition, error) {
	if err := validateTime(start); err != nil {
		return nil, fmt.Errorf("invalid start")
	}

	if err := validateTime(end); err != nil {
		return nil, fmt.Errorf("invalid end")
	}

	if len(days) == 0 {
		return nil, fmt.Errorf("missing week day list")
	}

	if err := validateWeekDays(days); err != nil {
		return nil, err
	}

	return &TimeWindowCondition{
		Start:    start,
		End:      end,
		WeekDays: days,
	}, nil
}

// We only need HH:mm precision, so we use time.TimeOnly format
// but without the seconds part.
// See: https://pkg.go.dev/time#pkg-constants.
var layout = "15:04"

// Copied from Golang's time package
var longDayNames = []string{
	"Sunday",
	"Monday",
	"Tuesday",
	"Wednesday",
	"Thursday",
	"Friday",
	"Saturday",
}

func validateTime(t string) error {
	_, err := time.Parse(layout, t)
	return err
}

func validateWeekDays(days []string) error {
	for _, day := range days {
		if !slices.Contains(longDayNames, day) {
			return fmt.Errorf("invalid day %s", day)
		}
	}

	return nil
}

func (c *TimeWindowCondition) Evaluate(t *time.Time) error {
	weekDay := t.Weekday().String()
	if !slices.Contains(c.WeekDays, weekDay) {
		return fmt.Errorf("current day - %s - is outside week days allowed - %v", weekDay, c.WeekDays)
	}

	hourAndMinute := fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
	now, err := time.Parse(layout, hourAndMinute)
	if err != nil {
		return err
	}

	if !c.inTimeWindow(now) {
		return fmt.Errorf("%s is not in time window %s-%s", hourAndMinute, c.Start, c.End)
	}

	return nil
}

func (c *TimeWindowCondition) inTimeWindow(now time.Time) bool {
	start, _ := time.Parse(layout, c.Start)
	end, _ := time.Parse(layout, c.End)

	if start.Before(end) {
		return (now.After(start) || now.Equal(start)) && now.Before(end)
	}

	return (now.After(start) || now.Equal(start)) || now.Before(end)
}

type ApprovalCondition struct {
	Count int `json:"count"`
}

type RunTemplate struct {
	Type string `json:"type"`

	//
	// Triggers a workflow on an existing Semaphore project/task.
	//
	Semaphore *SemaphoreRunTemplate `json:"semaphore_workflow,omitempty"`
}

type SemaphoreRunTemplate struct {
	ProjectID    string            `json:"project_id"`
	Branch       string            `json:"branch"`
	PipelineFile string            `json:"pipeline_file"`
	Parameters   map[string]string `json:"parameters"`
	TaskID       string            `json:"task_id"`
}

func FindStageByID(id uuid.UUID) (*Stage, error) {
	return FindStageByIDInTransaction(database.Conn(), id)
}

func FindStageByIDInTransaction(tx *gorm.DB, id uuid.UUID) (*Stage, error) {
	var stage Stage

	err := tx.
		Where("id = ?", id).
		First(&stage).
		Error

	if err != nil {
		return nil, err
	}

	return &stage, nil
}

func FindStage(id, orgID, canvasID uuid.UUID) (*Stage, error) {
	var stage Stage

	err := database.Conn().
		Where("organization_id = ?", orgID).
		Where("canvas_id = ?", canvasID).
		Where("id = ?", id).
		First(&stage).
		Error

	if err != nil {
		return nil, err
	}

	return &stage, nil
}

func (s *Stage) ApprovalsRequired() int {
	for _, condition := range s.Conditions {
		if condition.Type == StageConditionTypeApproval {
			return condition.Approval.Count
		}
	}

	return 0
}

func (s *Stage) HasApprovalCondition() bool {
	for _, condition := range s.Conditions {
		if condition.Type == StageConditionTypeApproval {
			return true
		}
	}

	return false
}

func (s *Stage) ListPendingEvents() ([]StageEvent, error) {
	return s.ListEvents([]string{StageEventStatePending}, []string{})
}

func (s *Stage) ListEvents(states, stateReasons []string) ([]StageEvent, error) {
	return s.ListEventsInTransaction(database.Conn(), states, stateReasons)
}

func (s *Stage) ListEventsInTransaction(tx *gorm.DB, states, stateReasons []string) ([]StageEvent, error) {
	var events []StageEvent
	query := tx.
		Where("stage_id = ?", s.ID).
		Where("state IN ?", states)

	if len(stateReasons) > 0 {
		query.Where("state_reason IN ?", stateReasons)
	}

	err := query.Order("created_at DESC").Find(&events).Error
	if err != nil {
		return nil, err
	}

	return events, nil
}

func (s *Stage) FindExecutionByID(id uuid.UUID) (*StageExecution, error) {
	var execution StageExecution

	err := database.Conn().
		Where("id = ?", id).
		Where("stage_id = ?", s.ID).
		First(&execution).
		Error

	if err != nil {
		return nil, err
	}

	return &execution, nil
}

func ListStagesByIDs(ids []uuid.UUID) ([]Stage, error) {
	var stages []Stage

	err := database.Conn().
		Where("id IN ?", ids).
		Find(&stages).
		Error

	if err != nil {
		return nil, err
	}

	return stages, nil
}
