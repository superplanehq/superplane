package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	PlanningCanvasName        = "Create with an Agent"
	PlanningCanvasLegacyName  = "__planning_session"
	PlanningCanvasDescription = "Starts the machine when you create a task with an agent."

	PlanningSessionStateRunning = "running"
	PlanningSessionStateEnded   = "ended"

	PlanningSessionMessageKindText   = "text"
	PlanningSessionMessageKindSurvey = "survey"
	PlanningSessionMessageRoleUser   = "user"
	PlanningSessionMessageRoleAgent  = "agent"

	PlanningWaitIdle     = ""
	PlanningWaitPending  = "pending"
	PlanningWaitResolved = "resolved"

	PlanningWaitKindMessage = "message"
	PlanningWaitKindCreated = "created"
	PlanningWaitKindSkipped = "skipped"
	PlanningWaitKindEnded   = "ended"

	PlanningSessionHeartbeatStale = 5 * time.Minute

	maxPlanningSurveyQuestions = 5
	maxPlanningSurveyOptions   = 6
)

var (
	ErrFactoryPlanningSessionInvalid  = errors.New("invalid planning session")
	ErrFactoryPlanningSessionNotFound = errors.New("planning session not found")
	ErrFactoryPlanningSessionEnded    = errors.New("planning session has ended")
	ErrFactoryPlanningSessionNoDraft  = errors.New("planning session has no draft")
	ErrFactoryPlanningWaitIdle        = errors.New("planning session is not waiting")
)

type PlanningSessionDraft struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

type PlanningSessionSurveyQuestion struct {
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"`
}

type PlanningSessionSurvey struct {
	Questions []PlanningSessionSurveyQuestion `json:"questions,omitempty"`
}

type PlanningSessionMessage struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Role      string    `json:"role"`
	Text      string    `json:"text,omitempty"`
	Delivered bool      `json:"delivered,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type PlanningWaitResult struct {
	Kind         string `json:"kind,omitempty"`
	Text         string `json:"text,omitempty"`
	WorkOrderID  string `json:"work_order_id,omitempty"`
	WorkOrderKey string `json:"work_order_key,omitempty"`
}

type StartPlanningSessionParams struct {
	CreatedByUserID uuid.UUID
	Repository      string
	CanvasID        uuid.UUID
	Entrypoint      string
}

type FactoryPlanningSession struct {
	ID                  uuid.UUID
	OrganizationID      uuid.UUID
	FactoryID           uuid.UUID
	CreatedByUserID     uuid.UUID
	Repository          string
	State               string
	CanvasID            *uuid.UUID
	CanvasRunID         *uuid.UUID
	Messages            datatypes.JSONSlice[PlanningSessionMessage]
	PendingDraft        datatypes.JSONType[PlanningSessionDraft]
	CreatedWorkOrderIDs datatypes.JSONSlice[string]
	WaitState           string
	WaitResult          datatypes.JSONType[PlanningWaitResult]
	HeartbeatAt         time.Time
	EndedAt             *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (FactoryPlanningSession) TableName() string {
	return "factory_planning_sessions"
}

func (f *Factory) StartPlanningSession(tx *gorm.DB, params StartPlanningSessionParams) (*FactoryPlanningSession, error) {
	repository := strings.TrimSpace(params.Repository)
	if repository == "" || params.CreatedByUserID == uuid.Nil || params.CanvasID == uuid.Nil || strings.TrimSpace(params.Entrypoint) == "" {
		return nil, ErrFactoryPlanningSessionInvalid
	}

	node, err := FindCanvasNode(tx, params.CanvasID, params.Entrypoint)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: entrypoint not found", ErrFactoryPlanningSessionInvalid)
		}
		return nil, err
	}
	if node.Ref.Data().Trigger == nil || node.Ref.Data().Trigger.Name != "onRun" {
		return nil, fmt.Errorf("%w: entrypoint must be onRun", ErrFactoryPlanningSessionInvalid)
	}

	liveVersion, err := FindLiveCanvasVersionInTransaction(tx, params.CanvasID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	run := &CanvasRun{
		ID:         uuid.New(),
		WorkflowID: params.CanvasID,
		NodeID:     params.Entrypoint,
		VersionID:  liveVersion.ID,
		Callbacks: datatypes.JSONSlice[core.RunCallback]{
			{When: core.RunCallbackWhenPending, On: core.RunCallbackOnEntry, Hook: "onMessage"},
		},
		Input: NewJSONValue(map[string]any{
			"planning_session": map[string]any{
				"factory_id": f.ID.String(),
				"repository": repository,
			},
		}),
		State:     CanvasRunStatePending,
		CreatedAt: &now,
		UpdatedAt: &now,
	}
	if err := tx.Create(run).Error; err != nil {
		return nil, err
	}

	canvasID := params.CanvasID
	session := &FactoryPlanningSession{
		ID:              uuid.New(),
		OrganizationID:  f.OrganizationID,
		FactoryID:       f.ID,
		CreatedByUserID: params.CreatedByUserID,
		Repository:      repository,
		State:           PlanningSessionStateRunning,
		CanvasID:        &canvasID,
		CanvasRunID:     &run.ID,
		Messages:        datatypes.JSONSlice[PlanningSessionMessage]{},
		HeartbeatAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := tx.Create(session).Error; err != nil {
		return nil, err
	}
	return session, nil
}

func (f *Factory) EndOpenPlanningSessions(tx *gorm.DB, createdBy uuid.UUID) ([]FactoryPlanningSession, error) {
	if createdBy == uuid.Nil {
		return nil, ErrFactoryPlanningSessionInvalid
	}
	var sessions []FactoryPlanningSession
	err := tx.Where(
		"organization_id = ? AND factory_id = ? AND created_by_user_id = ? AND state <> ?",
		f.OrganizationID,
		f.ID,
		createdBy,
		PlanningSessionStateEnded,
	).Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	ended := make([]FactoryPlanningSession, 0, len(sessions))
	for i := range sessions {
		if err := sessions[i].End(tx); err != nil {
			return nil, err
		}
		ended = append(ended, sessions[i])
	}
	return ended, nil
}

func FindPlanningSession(tx *gorm.DB, organizationID, factoryID, id uuid.UUID) (*FactoryPlanningSession, error) {
	var session FactoryPlanningSession
	err := tx.Where("organization_id = ? AND factory_id = ? AND id = ?", organizationID, factoryID, id).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFactoryPlanningSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func FindPlanningSessionByRun(tx *gorm.DB, canvasRunID uuid.UUID) (*FactoryPlanningSession, error) {
	var session FactoryPlanningSession
	err := tx.Where("canvas_run_id = ?", canvasRunID).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFactoryPlanningSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func ListStaleOpenPlanningSessions(tx *gorm.DB, now time.Time, limit int) ([]FactoryPlanningSession, error) {
	if limit < 1 {
		limit = 50
	}
	var sessions []FactoryPlanningSession
	cutoff := now.Add(-PlanningSessionHeartbeatStale)
	err := tx.
		Where("state <> ? AND heartbeat_at < ?", PlanningSessionStateEnded, cutoff).
		Order("heartbeat_at ASC").
		Limit(limit).
		Find(&sessions).Error
	return sessions, err
}

func IsPlanningCanvasName(name string) bool {
	return name == PlanningCanvasName || name == PlanningCanvasLegacyName
}

func FindPlanningCanvas(tx *gorm.DB, organizationID, factoryID uuid.UUID) (*Canvas, error) {
	return findPlanningCanvasByName(tx, organizationID, factoryID, PlanningCanvasName)
}

func findPlanningCanvasByName(tx *gorm.DB, organizationID, factoryID uuid.UUID, name string) (*Canvas, error) {
	var canvas Canvas
	err := tx.Where("organization_id = ? AND factory_id = ? AND name = ?", organizationID, factoryID, name).First(&canvas).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound
	}
	return &canvas, err
}

func (s *FactoryPlanningSession) Heartbeat(tx *gorm.DB) error {
	if s.State == PlanningSessionStateEnded {
		return ErrFactoryPlanningSessionEnded
	}
	now := time.Now()
	s.HeartbeatAt = now
	s.UpdatedAt = now
	return tx.Model(s).Select("HeartbeatAt", "UpdatedAt").Updates(s).Error
}

func (s *FactoryPlanningSession) End(tx *gorm.DB) error {
	if s.State == PlanningSessionStateEnded {
		return nil
	}
	now := time.Now()
	s.State = PlanningSessionStateEnded
	s.EndedAt = &now
	s.UpdatedAt = now
	if s.WaitState == PlanningWaitPending {
		s.WaitState = PlanningWaitResolved
		s.WaitResult = datatypes.NewJSONType(PlanningWaitResult{Kind: PlanningWaitKindEnded})
	}
	return tx.Model(s).Select("State", "EndedAt", "UpdatedAt", "WaitState", "WaitResult").Updates(s).Error
}

func (s *FactoryPlanningSession) EndIfStale(tx *gorm.DB, now time.Time) (bool, error) {
	if s.State == PlanningSessionStateEnded {
		return false, nil
	}
	if now.Sub(s.HeartbeatAt) < PlanningSessionHeartbeatStale {
		return false, nil
	}
	return true, s.End(tx)
}

func (s *FactoryPlanningSession) BeginWait(tx *gorm.DB) error {
	if err := s.guardOpen(); err != nil {
		return err
	}
	if s.WaitState == PlanningWaitPending || s.WaitState == PlanningWaitResolved {
		return nil
	}
	if text := s.nextUndeliveredUserText(); text != "" {
		s.deliverUserText(text)
		return s.saveMessagesAndWait(tx)
	}
	s.WaitState = PlanningWaitPending
	s.WaitResult = datatypes.NewJSONType(PlanningWaitResult{})
	s.UpdatedAt = time.Now()
	return tx.Model(s).Select("WaitState", "WaitResult", "UpdatedAt").Updates(s).Error
}

func (s *FactoryPlanningSession) ConsumeWait(tx *gorm.DB) (PlanningWaitResult, error) {
	if s.WaitState != PlanningWaitResolved {
		return PlanningWaitResult{}, ErrFactoryPlanningWaitIdle
	}
	result := s.WaitResult.Data()
	s.WaitState = PlanningWaitIdle
	s.WaitResult = datatypes.NewJSONType(PlanningWaitResult{})
	s.UpdatedAt = time.Now()
	if err := tx.Model(s).Select("WaitState", "WaitResult", "UpdatedAt").Updates(s).Error; err != nil {
		return PlanningWaitResult{}, err
	}
	return result, nil
}

func (s *FactoryPlanningSession) SendUserMessage(tx *gorm.DB, text string) error {
	if err := s.guardOpen(); err != nil {
		return err
	}
	body := strings.TrimSpace(text)
	if body == "" {
		return fmt.Errorf("%w: message is required", ErrFactoryPlanningSessionInvalid)
	}
	s.appendMessage(PlanningSessionMessage{
		ID:        uuid.NewString(),
		Kind:      PlanningSessionMessageKindText,
		Role:      PlanningSessionMessageRoleUser,
		Text:      body,
		CreatedAt: time.Now(),
	})
	s.clearPendingSurvey()
	if s.WaitState == PlanningWaitPending {
		s.deliverUserText(body)
	}
	return s.saveMessagesAndWait(tx)
}

func (s *FactoryPlanningSession) ProposeSurvey(tx *gorm.DB, survey PlanningSessionSurvey) error {
	if err := s.guardOpen(); err != nil {
		return err
	}
	normalized, err := normalizePlanningSurvey(survey)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	s.clearPendingSurvey()
	s.appendMessage(PlanningSessionMessage{
		ID:        uuid.NewString(),
		Kind:      PlanningSessionMessageKindSurvey,
		Role:      PlanningSessionMessageRoleAgent,
		Text:      string(payload),
		CreatedAt: time.Now(),
	})
	s.UpdatedAt = time.Now()
	return tx.Model(s).Select("Messages", "UpdatedAt").Updates(s).Error
}

func (s *FactoryPlanningSession) CurrentSurvey() PlanningSessionSurvey {
	for i := len(s.Messages) - 1; i >= 0; i-- {
		if s.Messages[i].Kind != PlanningSessionMessageKindSurvey {
			continue
		}
		var survey PlanningSessionSurvey
		if err := json.Unmarshal([]byte(s.Messages[i].Text), &survey); err != nil {
			return PlanningSessionSurvey{}
		}
		return survey
	}
	return PlanningSessionSurvey{}
}

func (s *FactoryPlanningSession) ProposeDraft(tx *gorm.DB, draft PlanningSessionDraft) error {
	if err := s.guardOpen(); err != nil {
		return err
	}
	title := strings.TrimSpace(draft.Title)
	if title == "" {
		return fmt.Errorf("%w: draft title is required", ErrFactoryPlanningSessionInvalid)
	}
	s.PendingDraft = datatypes.NewJSONType(PlanningSessionDraft{
		Title:       title,
		Description: strings.TrimSpace(draft.Description),
	})
	s.UpdatedAt = time.Now()
	return tx.Model(s).Select("PendingDraft", "UpdatedAt").Updates(s).Error
}

func (s *FactoryPlanningSession) UpdateDraft(tx *gorm.DB, draft PlanningSessionDraft) error {
	if err := s.guardOpen(); err != nil {
		return err
	}
	if strings.TrimSpace(s.PendingDraft.Data().Title) == "" {
		return ErrFactoryPlanningSessionNoDraft
	}
	title := strings.TrimSpace(draft.Title)
	if title == "" {
		title = s.PendingDraft.Data().Title
	}
	s.PendingDraft = datatypes.NewJSONType(PlanningSessionDraft{
		Title:       title,
		Description: strings.TrimSpace(draft.Description),
	})
	s.UpdatedAt = time.Now()
	return tx.Model(s).Select("PendingDraft", "UpdatedAt").Updates(s).Error
}

func (s *FactoryPlanningSession) SkipDraft(tx *gorm.DB) error {
	if err := s.guardOpen(); err != nil {
		return err
	}
	if strings.TrimSpace(s.PendingDraft.Data().Title) == "" {
		return ErrFactoryPlanningSessionNoDraft
	}
	s.PendingDraft = datatypes.NewJSONType(PlanningSessionDraft{})
	s.resolveWait(PlanningWaitResult{Kind: PlanningWaitKindSkipped})
	s.appendMessage(PlanningSessionMessage{
		ID:        uuid.NewString(),
		Kind:      PlanningSessionMessageKindText,
		Role:      PlanningSessionMessageRoleAgent,
		Text:      "Skipped that draft. What should we do next?",
		CreatedAt: time.Now(),
	})
	return s.saveMessagesAndWait(tx)
}

func (s *FactoryPlanningSession) CreateDraftWorkOrder(tx *gorm.DB, factoryModel *Factory, createdBy uuid.UUID) (*FactoryWorkOrder, error) {
	if err := s.guardOpen(); err != nil {
		return nil, err
	}
	draft := s.PendingDraft.Data()
	if strings.TrimSpace(draft.Title) == "" {
		return nil, ErrFactoryPlanningSessionNoDraft
	}

	order, err := factoryModel.CreateWorkOrder(tx, draft.Title, draft.Description, &createdBy, []uuid.UUID{createdBy}, nil)
	if err != nil {
		return nil, err
	}

	s.CreatedWorkOrderIDs = append(s.CreatedWorkOrderIDs, order.ID.String())
	s.PendingDraft = datatypes.NewJSONType(PlanningSessionDraft{})
	s.resolveWait(PlanningWaitResult{
		Kind:         PlanningWaitKindCreated,
		WorkOrderID:  order.ID.String(),
		WorkOrderKey: factoryModel.WorkOrderKey(order.Number),
		Text:         order.Title,
	})
	s.appendMessage(PlanningSessionMessage{
		ID:        uuid.NewString(),
		Kind:      PlanningSessionMessageKindText,
		Role:      PlanningSessionMessageRoleAgent,
		Text:      "Created the draft task. Work on a new one, or tell me what is next.",
		CreatedAt: time.Now(),
	})
	if err := s.saveMessagesAndWait(tx); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *FactoryPlanningSession) reload(tx *gorm.DB) error {
	return tx.Where("id = ?", s.ID).First(s).Error
}

func (s *FactoryPlanningSession) CreatedOrders(tx *gorm.DB) ([]FactoryWorkOrder, error) {
	if len(s.CreatedWorkOrderIDs) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, 0, len(s.CreatedWorkOrderIDs))
	for _, raw := range s.CreatedWorkOrderIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	var orders []FactoryWorkOrder
	if err := tx.Where("factory_id = ? AND id IN ?", s.FactoryID, ids).Find(&orders).Error; err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]FactoryWorkOrder, len(orders))
	for _, order := range orders {
		byID[order.ID] = order
	}
	ordered := make([]FactoryWorkOrder, 0, len(ids))
	for _, id := range ids {
		if order, ok := byID[id]; ok {
			ordered = append(ordered, order)
		}
	}
	return ordered, nil
}

func (s *FactoryPlanningSession) guardOpen() error {
	if s.State == PlanningSessionStateEnded {
		return ErrFactoryPlanningSessionEnded
	}
	return nil
}

func (s *FactoryPlanningSession) appendMessage(message PlanningSessionMessage) {
	s.Messages = append(s.Messages, message)
}

func (s *FactoryPlanningSession) clearPendingSurvey() {
	kept := s.Messages[:0]
	for _, message := range s.Messages {
		if message.Kind == PlanningSessionMessageKindSurvey {
			continue
		}
		kept = append(kept, message)
	}
	s.Messages = kept
}

func (s *FactoryPlanningSession) resolveWait(result PlanningWaitResult) {
	s.WaitState = PlanningWaitResolved
	s.WaitResult = datatypes.NewJSONType(result)
}

func (s *FactoryPlanningSession) nextUndeliveredUserText() string {
	for _, message := range s.Messages {
		if message.Kind == PlanningSessionMessageKindText && message.Role == PlanningSessionMessageRoleUser && !message.Delivered {
			return message.Text
		}
	}
	return ""
}

func (s *FactoryPlanningSession) deliverUserText(text string) {
	for i := range s.Messages {
		if s.Messages[i].Kind != PlanningSessionMessageKindText || s.Messages[i].Role != PlanningSessionMessageRoleUser {
			continue
		}
		if s.Messages[i].Delivered || s.Messages[i].Text != text {
			continue
		}
		s.Messages[i].Delivered = true
		break
	}
	s.resolveWait(PlanningWaitResult{Kind: PlanningWaitKindMessage, Text: text})
}

func (s *FactoryPlanningSession) saveMessagesAndWait(tx *gorm.DB) error {
	s.UpdatedAt = time.Now()
	return tx.Model(s).Select("Messages", "PendingDraft", "CreatedWorkOrderIDs", "WaitState", "WaitResult", "UpdatedAt").Updates(s).Error
}

func normalizePlanningSurvey(survey PlanningSessionSurvey) (PlanningSessionSurvey, error) {
	if len(survey.Questions) == 0 {
		return PlanningSessionSurvey{}, fmt.Errorf("%w: survey needs a question", ErrFactoryPlanningSessionInvalid)
	}
	if len(survey.Questions) > maxPlanningSurveyQuestions {
		return PlanningSessionSurvey{}, fmt.Errorf("%w: survey has too many questions", ErrFactoryPlanningSessionInvalid)
	}
	questions := make([]PlanningSessionSurveyQuestion, 0, len(survey.Questions))
	for _, question := range survey.Questions {
		prompt := strings.TrimSpace(question.Prompt)
		if prompt == "" {
			return PlanningSessionSurvey{}, fmt.Errorf("%w: survey question is required", ErrFactoryPlanningSessionInvalid)
		}
		options := make([]string, 0, len(question.Options))
		for _, option := range question.Options {
			option = strings.TrimSpace(option)
			if option == "" {
				continue
			}
			options = append(options, option)
			if len(options) == maxPlanningSurveyOptions {
				break
			}
		}
		if len(options) == 0 {
			return PlanningSessionSurvey{}, fmt.Errorf("%w: survey question needs an option", ErrFactoryPlanningSessionInvalid)
		}
		questions = append(questions, PlanningSessionSurveyQuestion{Prompt: prompt, Options: options})
	}
	return PlanningSessionSurvey{Questions: questions}, nil
}
