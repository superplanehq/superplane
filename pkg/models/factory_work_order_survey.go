package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	FactoryWorkOrderSurveyPending   = "pending"
	FactoryWorkOrderSurveyAnswered  = "answered"
	FactoryWorkOrderSurveyTimedOut  = "timed_out"
	FactoryWorkOrderSurveyCancelled = "cancelled"

	DefaultWorkOrderSurveyTimeoutSeconds = 3600
	MinWorkOrderSurveyTimeoutSeconds     = 60
	MaxWorkOrderSurveyTimeoutSeconds     = 18 * 60 * 60
	MaxWorkOrderSurveyQuestions          = 10
)

var (
	ErrFactoryWorkOrderSurveyInvalid    = errors.New("invalid work order survey")
	ErrFactoryWorkOrderSurveyNotFound   = errors.New("work order survey not found")
	ErrFactoryWorkOrderSurveyConflict   = errors.New("work order already has a pending survey")
	ErrFactoryWorkOrderSurveyNotPending = errors.New("work order survey is not pending")
)

const factoryWorkOrderSurveyPendingUnique = "idx_factory_work_order_surveys_one_pending"

type WorkOrderSurveyQuestion struct {
	ID            string   `json:"id"`
	Prompt        string   `json:"prompt"`
	Options       []string `json:"options,omitempty"`
	AllowFreeText bool     `json:"allow_free_text,omitempty"`
}

type WorkOrderSurveyAnswer struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type FactoryWorkOrderSurvey struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	FactoryID        uuid.UUID
	WorkOrderID      uuid.UUID
	CanvasRunID      uuid.UUID
	ExecutionID      *uuid.UUID
	Status           string
	Questions        datatypes.JSONSlice[WorkOrderSurveyQuestion]
	Answers          datatypes.JSONSlice[WorkOrderSurveyAnswer]
	TimeoutSeconds   int
	ExpiresAt        time.Time
	CreatedAt        time.Time
	AnsweredAt       *time.Time
	AnsweredByUserID *uuid.UUID
}

func (FactoryWorkOrderSurvey) TableName() string {
	return "factory_work_order_surveys"
}

type FactoryWorkOrderSurveyParams struct {
	CanvasRunID    uuid.UUID
	ExecutionID    *uuid.UUID
	TimeoutSeconds int
	Questions      []WorkOrderSurveyQuestion
}

func (o *FactoryWorkOrder) CreateSurvey(tx *gorm.DB, params FactoryWorkOrderSurveyParams) (*FactoryWorkOrderSurvey, bool, error) {
	normalized, err := normalizeSurveyParams(params)
	if err != nil {
		return nil, false, err
	}

	pending, err := o.PendingSurvey(tx)
	if err == nil {
		if pending.CanvasRunID == normalized.CanvasRunID {
			return pending, false, nil
		}
		return nil, false, ErrFactoryWorkOrderSurveyConflict
	}
	if !errors.Is(err, ErrFactoryWorkOrderSurveyNotFound) {
		return nil, false, err
	}

	now := time.Now()
	survey := &FactoryWorkOrderSurvey{
		ID:             uuid.New(),
		OrganizationID: o.OrganizationID,
		FactoryID:      o.FactoryID,
		WorkOrderID:    o.ID,
		CanvasRunID:    normalized.CanvasRunID,
		ExecutionID:    normalized.ExecutionID,
		Status:         FactoryWorkOrderSurveyPending,
		Questions:      normalized.Questions,
		TimeoutSeconds: normalized.TimeoutSeconds,
		ExpiresAt:      now.Add(time.Duration(normalized.TimeoutSeconds) * time.Second),
		CreatedAt:      now,
	}
	if err := tx.Create(survey).Error; err != nil {
		if isPendingSurveyConflict(err) {
			return nil, false, ErrFactoryWorkOrderSurveyConflict
		}
		return nil, false, err
	}
	return survey, true, nil
}

func (o *FactoryWorkOrder) PendingSurvey(tx *gorm.DB) (*FactoryWorkOrderSurvey, error) {
	var survey FactoryWorkOrderSurvey
	err := tx.Where("work_order_id = ? AND status = ?", o.ID, FactoryWorkOrderSurveyPending).First(&survey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFactoryWorkOrderSurveyNotFound
	}
	if err != nil {
		return nil, err
	}
	return &survey, nil
}

func ListPendingWorkOrderSurveys(tx *gorm.DB, workOrderIDs []uuid.UUID) (map[uuid.UUID]*FactoryWorkOrderSurvey, error) {
	byOrder := make(map[uuid.UUID]*FactoryWorkOrderSurvey, len(workOrderIDs))
	if len(workOrderIDs) == 0 {
		return byOrder, nil
	}

	var surveys []FactoryWorkOrderSurvey
	if err := tx.Where("work_order_id IN ? AND status = ?", workOrderIDs, FactoryWorkOrderSurveyPending).Find(&surveys).Error; err != nil {
		return nil, err
	}
	for i := range surveys {
		survey := surveys[i]
		byOrder[survey.WorkOrderID] = &survey
	}
	return byOrder, nil
}

func FindWorkOrderSurvey(tx *gorm.DB, organizationID, id uuid.UUID) (*FactoryWorkOrderSurvey, error) {
	var survey FactoryWorkOrderSurvey
	err := tx.Where("organization_id = ? AND id = ?", organizationID, id).First(&survey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFactoryWorkOrderSurveyNotFound
	}
	if err != nil {
		return nil, err
	}
	return &survey, nil
}

func (s *FactoryWorkOrderSurvey) Answer(tx *gorm.DB, actor uuid.UUID, answers []WorkOrderSurveyAnswer) error {
	if s.Status != FactoryWorkOrderSurveyPending {
		return ErrFactoryWorkOrderSurveyNotPending
	}

	resolved, err := resolveSurveyAnswers(s.Questions, answers)
	if err != nil {
		return err
	}

	now := time.Now()
	s.Status = FactoryWorkOrderSurveyAnswered
	s.Answers = resolved
	s.AnsweredAt = &now
	s.AnsweredByUserID = &actor
	if err := tx.Model(s).Select("Status", "Answers", "AnsweredAt", "AnsweredByUserID").Updates(s).Error; err != nil {
		return err
	}

	order := FactoryWorkOrder{ID: s.WorkOrderID, OrganizationID: s.OrganizationID, FactoryID: s.FactoryID}
	return order.recordEvent(tx, factory.EventTypeOrderSurveyAnswered, factory.WorkOrderSurveyAnswered{
		Order:     order.Ref(),
		SurveyID:  s.ID,
		User:      &factory.UserRef{ID: actor},
		Questions: surveyQuestionsToEvent(s.Questions),
		Answers:   surveyAnswersToEvent(resolved),
	})
}

func (s *FactoryWorkOrderSurvey) MarkTimedOut(tx *gorm.DB) error {
	return s.finishWithoutAnswer(tx, FactoryWorkOrderSurveyTimedOut)
}

func (s *FactoryWorkOrderSurvey) Cancel(tx *gorm.DB) error {
	return s.finishWithoutAnswer(tx, FactoryWorkOrderSurveyCancelled)
}

func (s *FactoryWorkOrderSurvey) ExpireIfDue(tx *gorm.DB, now time.Time) error {
	if s.Status != FactoryWorkOrderSurveyPending {
		return nil
	}
	if now.Before(s.ExpiresAt) {
		return nil
	}
	return s.MarkTimedOut(tx)
}

func (o *FactoryWorkOrder) CancelPendingSurvey(tx *gorm.DB) error {
	survey, err := o.PendingSurvey(tx)
	if errors.Is(err, ErrFactoryWorkOrderSurveyNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return survey.Cancel(tx)
}

func CancelPendingWorkOrderSurveysForRun(tx *gorm.DB, canvasRunID uuid.UUID) error {
	if canvasRunID == uuid.Nil {
		return nil
	}
	var surveys []FactoryWorkOrderSurvey
	if err := tx.Where("canvas_run_id = ? AND status = ?", canvasRunID, FactoryWorkOrderSurveyPending).Find(&surveys).Error; err != nil {
		return err
	}
	for i := range surveys {
		if err := surveys[i].Cancel(tx); err != nil && !errors.Is(err, ErrFactoryWorkOrderSurveyNotPending) {
			return err
		}
	}
	return nil
}

func (s *FactoryWorkOrderSurvey) finishWithoutAnswer(tx *gorm.DB, status string) error {
	if s.Status != FactoryWorkOrderSurveyPending {
		return ErrFactoryWorkOrderSurveyNotPending
	}
	now := time.Now()
	s.Status = status
	s.AnsweredAt = &now
	return tx.Model(s).Select("Status", "AnsweredAt").Updates(s).Error
}

func normalizeSurveyParams(params FactoryWorkOrderSurveyParams) (FactoryWorkOrderSurveyParams, error) {
	if params.CanvasRunID == uuid.Nil {
		return params, fmt.Errorf("%w: canvas run id is required", ErrFactoryWorkOrderSurveyInvalid)
	}
	if len(params.Questions) == 0 {
		return params, fmt.Errorf("%w: at least one question is required", ErrFactoryWorkOrderSurveyInvalid)
	}
	if len(params.Questions) > MaxWorkOrderSurveyQuestions {
		return params, fmt.Errorf("%w: at most %d questions", ErrFactoryWorkOrderSurveyInvalid, MaxWorkOrderSurveyQuestions)
	}

	seen := make(map[string]struct{}, len(params.Questions))
	normalized := make([]WorkOrderSurveyQuestion, 0, len(params.Questions))
	for _, question := range params.Questions {
		id := strings.TrimSpace(question.ID)
		prompt := strings.TrimSpace(question.Prompt)
		if id == "" || prompt == "" {
			return params, fmt.Errorf("%w: each question needs an id and a prompt", ErrFactoryWorkOrderSurveyInvalid)
		}
		if _, exists := seen[id]; exists {
			return params, fmt.Errorf("%w: duplicate question id %q", ErrFactoryWorkOrderSurveyInvalid, id)
		}
		seen[id] = struct{}{}
		options := make([]string, 0, len(question.Options))
		for _, option := range question.Options {
			option = strings.TrimSpace(option)
			if option != "" {
				options = append(options, option)
			}
		}
		normalized = append(normalized, WorkOrderSurveyQuestion{
			ID:            id,
			Prompt:        prompt,
			Options:       options,
			AllowFreeText: question.AllowFreeText || len(options) == 0,
		})
	}

	timeout := params.TimeoutSeconds
	if timeout <= 0 {
		timeout = DefaultWorkOrderSurveyTimeoutSeconds
	}
	if timeout < MinWorkOrderSurveyTimeoutSeconds || timeout > MaxWorkOrderSurveyTimeoutSeconds {
		return params, fmt.Errorf("%w: timeout must be between %d and %d seconds", ErrFactoryWorkOrderSurveyInvalid, MinWorkOrderSurveyTimeoutSeconds, MaxWorkOrderSurveyTimeoutSeconds)
	}

	params.Questions = normalized
	params.TimeoutSeconds = timeout
	return params, nil
}

func resolveSurveyAnswers(questions []WorkOrderSurveyQuestion, answers []WorkOrderSurveyAnswer) ([]WorkOrderSurveyAnswer, error) {
	byID := make(map[string]string, len(answers))
	known := make(map[string]struct{}, len(questions))
	for _, question := range questions {
		known[question.ID] = struct{}{}
	}
	for _, answer := range answers {
		id := strings.TrimSpace(answer.ID)
		if _, ok := known[id]; !ok {
			return nil, fmt.Errorf("%w: unknown question %q", ErrFactoryWorkOrderSurveyInvalid, id)
		}
		value := strings.TrimSpace(answer.Value)
		if value == "" {
			value = "skipped"
		}
		byID[id] = value
	}

	resolved := make([]WorkOrderSurveyAnswer, 0, len(questions))
	for _, question := range questions {
		value, ok := byID[question.ID]
		if !ok {
			value = "skipped"
		}
		resolved = append(resolved, WorkOrderSurveyAnswer{ID: question.ID, Value: value})
	}
	return resolved, nil
}

func surveyQuestionsToEvent(questions []WorkOrderSurveyQuestion) []factory.SurveyQuestion {
	out := make([]factory.SurveyQuestion, 0, len(questions))
	for _, question := range questions {
		out = append(out, factory.SurveyQuestion{
			ID:            question.ID,
			Prompt:        question.Prompt,
			Options:       question.Options,
			AllowFreeText: question.AllowFreeText,
		})
	}
	return out
}

func surveyAnswersToEvent(answers []WorkOrderSurveyAnswer) []factory.SurveyAnswer {
	out := make([]factory.SurveyAnswer, 0, len(answers))
	for _, answer := range answers {
		out = append(out, factory.SurveyAnswer{ID: answer.ID, Value: answer.Value})
	}
	return out
}

func isPendingSurveyConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.ConstraintName == factoryWorkOrderSurveyPendingUnique
}
