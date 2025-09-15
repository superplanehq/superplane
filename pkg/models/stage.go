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
	ExecutorTypeSemaphore = "semaphore"
	ExecutorTypeHTTP      = "http"

	StageConditionTypeApproval   = "approval"
	StageConditionTypeTimeWindow = "time-window"
)

type Stage struct {
	ID          uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	CanvasID    uuid.UUID
	Name        string
	Description string
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
	CreatedBy   uuid.UUID
	UpdatedBy   uuid.UUID
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	ExecutorType string
	ExecutorSpec datatypes.JSON
	ExecutorName string
	ResourceID   *uuid.UUID

	Conditions    datatypes.JSONSlice[StageCondition]
	Inputs        datatypes.JSONSlice[InputDefinition]
	InputMappings datatypes.JSONSlice[InputMapping]
	Outputs       datatypes.JSONSlice[OutputDefinition]
	Secrets       datatypes.JSONSlice[ValueDefinition]
}

type InputDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type OutputDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type InputMapping struct {
	When   *InputMappingWhen `json:"when"`
	Values []ValueDefinition `json:"values"`
}

type InputMappingWhen struct {
	TriggeredBy *WhenTriggeredBy `json:"triggered_by"`
}

type WhenTriggeredBy struct {
	Connection string `json:"connection"`
}

type ValueDefinition struct {
	Name      string               `json:"name"`
	ValueFrom *ValueDefinitionFrom `json:"value_from"`
	Value     *string              `json:"value"`
}

type ValueDefinitionFrom struct {
	EventData     *ValueDefinitionFromEventData     `json:"event_data,omitempty"`
	LastExecution *ValueDefinitionFromLastExecution `json:"last_execution,omitempty"`
	Secret        *ValueDefinitionFromSecret        `json:"secret,omitempty"`
}

type ValueDefinitionFromEventData struct {
	Connection string `json:"connection"`
	Expression string `json:"expression"`
}

type ValueDefinitionFromLastExecution struct {
	Results []string `json:"results"`
}

type ValueDefinitionFromSecret struct {
	DomainType string `json:"domain_type"`
	Name       string `json:"name"`
	Key        string `json:"key"`
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

func ListStages(canvasID string) ([]Stage, error) {
	var stages []Stage

	err := database.Conn().
		Where("canvas_id = ?", canvasID).
		Order("name ASC").
		Find(&stages).
		Error

	if err != nil {
		return nil, err
	}

	return stages, nil
}

// NOTE: we are not querying scoped by canvas here,
// so this should be used only in the workers.
func FindUnscopedStage(id string) (*Stage, error) {
	var stage Stage

	err := database.Conn().
		Where("id = ?", id).
		First(&stage).
		Error

	if err != nil {
		return nil, err
	}

	return &stage, nil
}

func ListUnscopedSoftDeletedStages(limit int) ([]Stage, error) {
	var stages []Stage

	err := database.Conn().
		Unscoped().
		Where("deleted_at is not null").
		Limit(limit).
		Find(&stages).
		Error

	if err != nil {
		return nil, err
	}

	return stages, nil
}

func FindStageByID(canvasID string, id string) (*Stage, error) {
	return FindStageByIDInTransaction(database.Conn(), canvasID, id)
}

func FindStageByIDInTransaction(tx *gorm.DB, canvasID string, id string) (*Stage, error) {
	var stage Stage

	err := tx.
		Where("id = ?", id).
		Where("canvas_id = ?", canvasID).
		First(&stage).
		Error

	if err != nil {
		return nil, err
	}

	return &stage, nil
}

func FindStageByName(canvasID string, name string) (*Stage, error) {
	var stage Stage

	err := database.Conn().
		Where("canvas_id = ?", canvasID).
		Where("name = ?", name).
		First(&stage).
		Error

	if err != nil {
		return nil, err
	}

	return &stage, nil
}

func (s *Stage) GetResource() (*Resource, error) {
	var resource Resource

	err := database.Conn().
		Where("id = ?", s.ResourceID).
		First(&resource).
		Error

	if err != nil {
		return nil, err
	}

	return &resource, nil
}

func (s *Stage) GetIntegrationResource() (*IntegrationResource, error) {
	var r IntegrationResource

	err := database.Conn().
		Table("resources").
		Joins("INNER JOIN integrations ON integrations.id = resources.integration_id").
		Select("resources.name as name, resources.type as type, integrations.name as integration_name, integrations.domain_type as domain_type").
		Where("resources.id = ?", s.ResourceID).
		First(&r).
		Error

	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (s *Stage) FindIntegration() (*Integration, error) {
	var integration Integration

	err := database.Conn().
		Table("resources").
		Joins("INNER JOIN integrations ON integrations.id = resources.integration_id").
		Where("resources.id = ?", s.ResourceID).
		Select("integrations.*").
		First(&integration).
		Error

	if err != nil {
		return nil, err
	}

	return &integration, nil
}

func (s *Stage) AddConnection(tx *gorm.DB, connection Connection) error {
	connection.CanvasID = s.CanvasID
	connection.TargetID = s.ID
	connection.TargetType = ConnectionTargetTypeStage
	return tx.Create(&connection).Error
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

func (s *Stage) MissingRequiredOutputs(outputs map[string]any) []string {
	missing := []string{}
	for _, outputDef := range s.Outputs {
		if !outputDef.Required {
			continue
		}

		if _, ok := outputs[outputDef.Name]; !ok {
			missing = append(missing, outputDef.Name)
		}
	}

	return missing
}

func (s *Stage) HasOutputDefinition(name string) bool {
	for _, outputDefinition := range s.Outputs {
		if outputDefinition.Name == name {
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

func (s *Stage) FilterEvents(states, stateReasons []string, limit int, before *time.Time) ([]StageEvent, error) {
	var events []StageEvent
	query := database.Conn().
		Preload("Event").
		Where("stage_id = ?", s.ID).
		Where("state IN ?", states)

	if len(stateReasons) > 0 {
		query = query.Where("state_reason IN ?", stateReasons)
	}

	if before != nil {
		query = query.Where("created_at < ?", before)
	}

	err := query.Order("created_at DESC").Limit(limit).Find(&events).Error
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

func (s *Stage) FindLastExecutionInputs(tx *gorm.DB, results []string) (map[string]any, error) {
	var event StageEvent

	err := tx.
		Table("stage_events AS e").
		Select("e.*").
		Joins("INNER JOIN stage_executions AS ex ON ex.stage_event_id = e.id").
		Where("e.stage_id = ?", s.ID).
		Where("ex.state = ?", ExecutionFinished).
		Where("ex.result IN ?", results).
		Order("ex.finished_at DESC").
		Limit(1).
		First(&event).
		Error

	if err != nil {
		return nil, err
	}

	return event.Inputs.Data(), nil
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

func (s *Stage) Delete() error {
	deletedName := fmt.Sprintf("%s-deleted-%d", s.Name, time.Now().Unix())

	return database.Conn().Model(s).
		Where("id = ?", s.ID).
		Update("name", deletedName).
		Update("deleted_at", time.Now()).
		Error
}

func (s *Stage) HardDeleteInTransaction(tx *gorm.DB) error {
	return tx.Unscoped().Delete(s).Error
}

func (s *Stage) DeleteStageExecutionsInTransaction(tx *gorm.DB) error {
	// Delete execution resources for all executions of this stage
	if err := tx.Unscoped().
		Where("execution_id IN (SELECT id FROM stage_executions WHERE stage_id = ?)", s.ID).
		Delete(&ExecutionResource{}).Error; err != nil {
		return fmt.Errorf("failed to delete execution resources: %v", err)
	}

	// Delete stage executions
	if err := tx.Unscoped().Where("stage_id = ?", s.ID).Delete(&StageExecution{}).Error; err != nil {
		return fmt.Errorf("failed to delete stage executions: %v", err)
	}

	return nil
}

func (s *Stage) DeleteStageEventsInTransaction(tx *gorm.DB) error {
	// Delete events associated with stage events
	if err := tx.Unscoped().
		Where("id IN (SELECT event_id FROM stage_events WHERE stage_id = ?)", s.ID).
		Delete(&Event{}).Error; err != nil {
		return fmt.Errorf("failed to delete events: %v", err)
	}

	if err := tx.Unscoped().Where("stage_id = ?", s.ID).Delete(&StageEvent{}).Error; err != nil {
		return fmt.Errorf("failed to delete stage events: %v", err)
	}

	return nil
}

func (s *Stage) ListExecutionsWithLimitAndBefore(states []string, results []string, limit int, before *time.Time) ([]StageExecution, error) {
	var executions []StageExecution
	query := database.Conn().
		Preload("StageEvent", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Event")
		}).
		Where("stage_id = ?", s.ID)

	if len(states) > 0 {
		query = query.Where("state IN ?", states)
	}

	if len(results) > 0 {
		query = query.Where("result IN ?", results)
	}

	if before != nil {
		query = query.Where("created_at < ?", before)
	}

	err := query.Order("created_at DESC").Limit(limit).Find(&executions).Error
	if err != nil {
		return nil, err
	}

	return executions, nil
}

func (s *Stage) DeleteConnectionsInTransaction(tx *gorm.DB) error {
	if err := tx.Unscoped().Where("target_id = ? AND target_type = ?", s.ID, ConnectionTargetTypeStage).Delete(&Connection{}).Error; err != nil {
		return fmt.Errorf("failed to delete connections: %v", err)
	}
	return nil
}

func (s *Stage) CountExecutions(states []string, results []string) (int64, error) {
	query := database.Conn().
		Model(&StageExecution{}).
		Where("stage_id = ?", s.ID)

	if len(states) > 0 {
		query = query.Where("state IN ?", states)
	}

	if len(results) > 0 {
		query = query.Where("result IN ?", results)
	}

	var count int64
	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Stage) CountEvents(states, stateReasons []string) (int64, error) {
	query := database.Conn().
		Model(&StageEvent{}).
		Where("stage_id = ?", s.ID)

	if len(states) > 0 {
		query = query.Where("state IN ?", states)
	}

	if len(stateReasons) > 0 {
		query = query.Where("state_reason IN ?", stateReasons)
	}

	var count int64
	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

type StageStatusInfo struct {
	StageID       uuid.UUID
	LastExecution *StageExecution
	QueueTotal    int
	QueueItems    []StageEvent
}

func GetStagesStatusInfo(stages []Stage) (map[uuid.UUID]*StageStatusInfo, error) {
	statusMap := make(map[uuid.UUID]*StageStatusInfo)

	if len(stages) == 0 {
		return statusMap, nil
	}

	stageIDs := make([]uuid.UUID, len(stages))
	for i, stage := range stages {
		stageIDs[i] = stage.ID
		statusMap[stage.ID] = &StageStatusInfo{
			StageID:    stage.ID,
			QueueItems: []StageEvent{},
		}
	}

	lastExecutions, err := getLastExecutionsForStages(stageIDs)
	if err != nil {
		return nil, err
	}

	queueCounts, queueItems, err := getQueueInfoForStages(stageIDs)
	if err != nil {
		return nil, err
	}

	for stageID, execution := range lastExecutions {
		if status, exists := statusMap[stageID]; exists {
			status.LastExecution = execution
		}
	}

	for stageID, count := range queueCounts {
		if status, exists := statusMap[stageID]; exists {
			status.QueueTotal = count
		}
	}

	for stageID, items := range queueItems {
		if status, exists := statusMap[stageID]; exists {
			status.QueueItems = items
		}
	}

	return statusMap, nil
}

func getLastExecutionsForStages(stageIDs []uuid.UUID) (map[uuid.UUID]*StageExecution, error) {
	executions := make(map[uuid.UUID]*StageExecution)

	var results []StageExecution

	err := database.Conn().
		Preload("StageEvent", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Event")
		}).
		Raw(`
			SELECT * FROM stage_executions 
			WHERE id IN (
				SELECT DISTINCT ON (stage_id) id
				FROM stage_executions 
				WHERE stage_id IN ?
				ORDER BY stage_id, created_at DESC
			)
		`, stageIDs).
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	for _, result := range results {
		executions[result.StageID] = &result
	}

	return executions, nil
}

func getQueueInfoForStages(stageIDs []uuid.UUID) (map[uuid.UUID]int, map[uuid.UUID][]StageEvent, error) {
	counts := make(map[uuid.UUID]int)
	items := make(map[uuid.UUID][]StageEvent)

	var queueEvents []struct {
		StageID     uuid.UUID
		ID          uuid.UUID
		Name        string
		EventID     uuid.UUID
		SourceID    uuid.UUID
		SourceName  string
		SourceType  string
		State       string
		StateReason string
		CreatedAt   *time.Time
	}

	err := database.Conn().
		Raw(`
			SELECT 
				stage_id,
				id,
				name,
				event_id,
				source_id,
				source_name,
				source_type,
				state,
				state_reason,
				created_at
			FROM stage_events
			WHERE stage_id IN ? 
				AND state IN (?, ?)
			ORDER BY stage_id, created_at ASC
		`, stageIDs, StageEventStatePending, StageEventStateWaiting).
		Find(&queueEvents).Error

	if err != nil {
		return nil, nil, err
	}

	for _, event := range queueEvents {
		counts[event.StageID]++

		stageEvent := StageEvent{
			ID:          event.ID,
			Name:        event.Name,
			StageID:     event.StageID,
			EventID:     event.EventID,
			SourceID:    event.SourceID,
			SourceName:  event.SourceName,
			SourceType:  event.SourceType,
			State:       event.State,
			StateReason: event.StateReason,
			CreatedAt:   event.CreatedAt,
		}

		if _, exists := items[event.StageID]; !exists {
			items[event.StageID] = []StageEvent{}
		}
		items[event.StageID] = append(items[event.StageID], stageEvent)
	}

	return counts, items, nil
}
