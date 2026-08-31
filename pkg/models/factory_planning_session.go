package models

import (
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

	PlanningSessionStateStarting = "starting"
	PlanningSessionStateRunning  = "running"
	PlanningSessionStateEnded    = "ended"

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

	PlanningSessionGreeting          = "The repository is ready. What do you want to do?"
	PlanningSessionHeartbeatStale    = 45 * time.Second
	DefaultPlanningSurveyTimeoutSecs = 3600
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

type PlanningSessionMessage struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Role      string    `json:"role"`
	Text      string    `json:"text,omitempty"`
	SurveyID  string    `json:"survey_id,omitempty"`
	Answered  bool      `json:"answered,omitempty"`
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
		Messages: datatypes.JSONSlice[PlanningSessionMessage]{
			{
				ID:        "greet",
				Kind:      PlanningSessionMessageKindText,
				Role:      PlanningSessionMessageRoleAgent,
				Text:      PlanningSessionGreeting,
				CreatedAt: now,
			},
		},
		HeartbeatAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := tx.Create(session).Error; err != nil {
		return nil, err
	}
	return session, nil
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
	canvas, err := findPlanningCanvasByName(tx, organizationID, factoryID, PlanningCanvasName)
	if err == nil {
		return canvas, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return findPlanningCanvasByName(tx, organizationID, factoryID, PlanningCanvasLegacyName)
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
	if s.WaitState == PlanningWaitPending {
		return nil
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
	if s.WaitState == PlanningWaitPending {
		s.resolveWait(PlanningWaitResult{Kind: PlanningWaitKindMessage, Text: body})
	}
	return s.saveMessagesAndWait(tx)
}

func (s *FactoryPlanningSession) AppendAgentMessage(tx *gorm.DB, text string) error {
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
		Role:      PlanningSessionMessageRoleAgent,
		Text:      body,
		CreatedAt: time.Now(),
	})
	s.UpdatedAt = time.Now()
	return tx.Model(s).Select("Messages", "UpdatedAt").Updates(s).Error
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
	if s.WaitState == PlanningWaitPending {
		s.resolveWait(PlanningWaitResult{Kind: PlanningWaitKindSkipped})
	}
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
	if s.WaitState == PlanningWaitPending {
		s.resolveWait(PlanningWaitResult{
			Kind:         PlanningWaitKindCreated,
			WorkOrderID:  order.ID.String(),
			WorkOrderKey: factoryModel.WorkOrderKey(order.Number),
			Text:         order.Title,
		})
	}
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

func (s *FactoryPlanningSession) CreateSurvey(tx *gorm.DB, questions []WorkOrderSurveyQuestion) (*FactoryPlanningSessionSurvey, error) {
	if err := s.guardOpen(); err != nil {
		return nil, err
	}
	if s.CanvasRunID == nil {
		return nil, fmt.Errorf("%w: canvas run is required", ErrFactoryPlanningSessionInvalid)
	}

	normalized, err := normalizeSurveyParams(FactoryWorkOrderSurveyParams{
		CanvasRunID:    *s.CanvasRunID,
		TimeoutSeconds: DefaultPlanningSurveyTimeoutSecs,
		Questions:      questions,
	})
	if err != nil {
		return nil, err
	}

	pending, err := s.PendingSurvey(tx)
	if err == nil {
		return pending, nil
	}
	if !errors.Is(err, ErrFactoryWorkOrderSurveyNotFound) {
		return nil, err
	}

	now := time.Now()
	survey := &FactoryPlanningSessionSurvey{
		ID:             uuid.New(),
		OrganizationID: s.OrganizationID,
		FactoryID:      s.FactoryID,
		SessionID:      s.ID,
		CanvasRunID:    *s.CanvasRunID,
		Status:         FactoryWorkOrderSurveyPending,
		Questions:      normalized.Questions,
		TimeoutSeconds: normalized.TimeoutSeconds,
		ExpiresAt:      now.Add(time.Duration(normalized.TimeoutSeconds) * time.Second),
		CreatedAt:      now,
	}
	if err := tx.Create(survey).Error; err != nil {
		return nil, err
	}
	s.appendMessage(PlanningSessionMessage{
		ID:        survey.ID.String(),
		Kind:      PlanningSessionMessageKindSurvey,
		Role:      PlanningSessionMessageRoleAgent,
		SurveyID:  survey.ID.String(),
		CreatedAt: now,
	})
	s.UpdatedAt = now
	if err := tx.Model(s).Select("Messages", "UpdatedAt").Updates(s).Error; err != nil {
		return nil, err
	}
	return survey, nil
}

func (s *FactoryPlanningSession) PendingSurvey(tx *gorm.DB) (*FactoryPlanningSessionSurvey, error) {
	var survey FactoryPlanningSessionSurvey
	err := tx.Where("session_id = ? AND status = ?", s.ID, FactoryWorkOrderSurveyPending).First(&survey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFactoryWorkOrderSurveyNotFound
	}
	if err != nil {
		return nil, err
	}
	return &survey, nil
}

func (s *FactoryPlanningSession) markSurveyAnswered(tx *gorm.DB, surveyID uuid.UUID) error {
	for i := range s.Messages {
		if s.Messages[i].Kind == PlanningSessionMessageKindSurvey && s.Messages[i].SurveyID == surveyID.String() {
			s.Messages[i].Answered = true
		}
	}
	s.UpdatedAt = time.Now()
	return tx.Model(s).Select("Messages", "UpdatedAt").Updates(s).Error
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

func (s *FactoryPlanningSession) resolveWait(result PlanningWaitResult) {
	s.WaitState = PlanningWaitResolved
	s.WaitResult = datatypes.NewJSONType(result)
}

func (s *FactoryPlanningSession) saveMessagesAndWait(tx *gorm.DB) error {
	s.UpdatedAt = time.Now()
	return tx.Model(s).Select("Messages", "PendingDraft", "CreatedWorkOrderIDs", "WaitState", "WaitResult", "UpdatedAt").Updates(s).Error
}

type FactoryPlanningSessionSurvey struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	FactoryID        uuid.UUID
	SessionID        uuid.UUID
	CanvasRunID      uuid.UUID
	Status           string
	Questions        datatypes.JSONSlice[WorkOrderSurveyQuestion]
	Answers          datatypes.JSONSlice[WorkOrderSurveyAnswer]
	TimeoutSeconds   int
	ExpiresAt        time.Time
	CreatedAt        time.Time
	AnsweredAt       *time.Time
	AnsweredByUserID *uuid.UUID
}

func (FactoryPlanningSessionSurvey) TableName() string {
	return "factory_planning_session_surveys"
}

func FindPlanningSessionSurvey(tx *gorm.DB, organizationID, id uuid.UUID) (*FactoryPlanningSessionSurvey, error) {
	var survey FactoryPlanningSessionSurvey
	err := tx.Where("organization_id = ? AND id = ?", organizationID, id).First(&survey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFactoryWorkOrderSurveyNotFound
	}
	if err != nil {
		return nil, err
	}
	return &survey, nil
}

func (survey *FactoryPlanningSessionSurvey) Answer(tx *gorm.DB, actor uuid.UUID, answers []WorkOrderSurveyAnswer) error {
	if survey.Status != FactoryWorkOrderSurveyPending {
		return ErrFactoryWorkOrderSurveyNotPending
	}
	resolved, err := resolveSurveyAnswers(survey.Questions, answers)
	if err != nil {
		return err
	}
	now := time.Now()
	survey.Status = FactoryWorkOrderSurveyAnswered
	survey.Answers = resolved
	survey.AnsweredAt = &now
	survey.AnsweredByUserID = &actor
	if err := tx.Model(survey).Select("Status", "Answers", "AnsweredAt", "AnsweredByUserID").Updates(survey).Error; err != nil {
		return err
	}
	session, err := FindPlanningSession(tx, survey.OrganizationID, survey.FactoryID, survey.SessionID)
	if err != nil {
		return err
	}
	return session.markSurveyAnswered(tx, survey.ID)
}
