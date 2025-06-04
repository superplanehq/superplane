package models

import (
	"context"
	"fmt"
	"slices"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/secrets"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	ExecutorSpecTypeSemaphore = "semaphore"
	ExecutorSpecTypeHTTP      = "http"

	StageConditionTypeApproval   = "approval"
	StageConditionTypeTimeWindow = "time-window"
)

type Stage struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;"`
	CanvasID  uuid.UUID
	Name      string
	CreatedAt *time.Time
	UpdatedAt *time.Time
	CreatedBy uuid.UUID
	UpdatedBy uuid.UUID

	Conditions    datatypes.JSONSlice[StageCondition]
	ExecutorSpec  datatypes.JSONType[ExecutorSpec]
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
	EventData     *ValueDefinitionFromEventData     `json:"event_data"`
	LastExecution *ValueDefinitionFromLastExecution `json:"last_execution"`
	Secret        *ValueDefinitionFromSecret        `json:"secret"`
}

type ValueDefinitionFromEventData struct {
	Connection string `json:"connection"`
	Expression string `json:"expression"`
}

type ValueDefinitionFromLastExecution struct {
	Results []string `json:"results"`
}

type ValueDefinitionFromSecret struct {
	Name string `json:"name"`
	Key  string `json:"key"`
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

type ExecutorSpec struct {
	Type      string                 `json:"type"`
	Semaphore *SemaphoreExecutorSpec `json:"semaphore,omitempty"`
	HTTP      *HTTPExecutorSpec      `json:"http,omitempty"`
}

type SemaphoreExecutorSpec struct {
	APIToken        string            `json:"api_token"`
	OrganizationURL string            `json:"organization_url"`
	ProjectID       string            `json:"project_id"`
	Branch          string            `json:"branch"`
	PipelineFile    string            `json:"pipeline_file"`
	Parameters      map[string]string `json:"parameters"`
	TaskID          string            `json:"task_id"`
}

type HTTPExecutorSpec struct {
	URL            string              `json:"url"`
	Payload        map[string]string   `json:"payload"`
	Headers        map[string]string   `json:"headers"`
	ResponsePolicy *HTTPResponsePolicy `json:"success_policy"`
}

type HTTPResponsePolicy struct {
	StatusCodes []uint32 `json:"status_codes"`
}

func FindStageByID(id string) (*Stage, error) {
	return FindStageByIDInTransaction(database.Conn(), id)
}

func FindStageByIDInTransaction(tx *gorm.DB, id string) (*Stage, error) {
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

func FindStage(id, canvasID uuid.UUID) (*Stage, error) {
	var stage Stage

	err := database.Conn().
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
		Where("ex.state = ?", StageExecutionFinished).
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

func (s *Stage) FindSecrets(encryptor crypto.Encryptor) (map[string]string, error) {
	secretMap := map[string]string{}
	for _, secretDef := range s.Secrets {
		secretName := secretDef.ValueFrom.Secret.Name
		secret, err := FindSecretByName(s.CanvasID.String(), secretName)
		if err != nil {
			return nil, fmt.Errorf("error finding secret %s: %v", secretName, err)
		}

		provider, err := secrets.NewProvider(secret.Provider, secrets.Options{
			CanvasID:   s.CanvasID,
			Encryptor:  encryptor,
			SecretName: secret.Name,
			SecretData: secret.Data,
		})

		if err != nil {
			return nil, fmt.Errorf("error initializing secret provider for %s: %v", secretName, err)
		}

		values, err := provider.Get(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("error getting secret values for %s: %v", secretName, err)
		}

		secretMap[secretDef.Name] = values[secretDef.ValueFrom.Secret.Key]
	}

	return secretMap, nil
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
