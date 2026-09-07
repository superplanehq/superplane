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
	"gorm.io/gorm/clause"
)

const (
	PlanningCanvasName        = "Create with an Agent"
	PlanningCanvasDescription = "Starts the machine when you create a task with an agent."

	PlanningSessionStateRunning = "running"
	PlanningSessionStateEnded   = "ended"

	PlanningSessionMessageRoleUser  = "user"
	PlanningSessionMessageRoleAgent = "agent"

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
	ErrFactoryPlanningSessionBusy     = errors.New("too many Create with an Agent sessions are running")
	ErrFactoryPlanningWaitIdle        = errors.New("planning session is not waiting")
)

type PlanningSessionDraft struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	WorkOrderID string `json:"work_order_id,omitempty"`
}

type PlanningSessionSurveyQuestion struct {
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"`
}

type PlanningSessionSurvey struct {
	Questions []PlanningSessionSurveyQuestion `json:"questions,omitempty"`
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
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	FactoryID          uuid.UUID
	CreatedByUserID    uuid.UUID
	Repository         string
	State              string
	CanvasID           *uuid.UUID
	CanvasRunID        *uuid.UUID
	DraftTitle         string
	DraftDescription   string
	DraftWorkOrderID   *uuid.UUID
	WaitState          string
	WaitKind           string
	WaitText           string
	WaitWorkOrderID    *uuid.UUID
	WaitWorkOrderKey   string
	SurveyID           *uuid.UUID
	Survey             datatypes.JSONType[PlanningSessionSurvey]
	SelectableModelKey string
	HeartbeatAt        time.Time
	EndedAt            *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Messages           []PlanningSessionMessage `gorm:"-"`
}

func (FactoryPlanningSession) TableName() string {
	return "factory_planning_sessions"
}

func (s *FactoryPlanningSession) Draft() PlanningSessionDraft {
	draft := PlanningSessionDraft{
		Title:       s.DraftTitle,
		Description: s.DraftDescription,
	}
	if s.DraftWorkOrderID != nil {
		draft.WorkOrderID = s.DraftWorkOrderID.String()
	}
	return draft
}

func (s *FactoryPlanningSession) Wait() PlanningWaitResult {
	result := PlanningWaitResult{
		Kind:         s.WaitKind,
		Text:         s.WaitText,
		WorkOrderKey: s.WaitWorkOrderKey,
	}
	if s.WaitWorkOrderID != nil {
		result.WorkOrderID = s.WaitWorkOrderID.String()
	}
	return result
}

func (s *FactoryPlanningSession) CurrentSurvey() PlanningSessionSurvey {
	if s.SurveyID == nil {
		return PlanningSessionSurvey{}
	}
	return s.Survey.Data()
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
	run := NewPlanningSessionRun(params.CanvasID, liveVersion.ID, params.Entrypoint, f.ID.String(), repository, "")
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
		HeartbeatAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := tx.Create(session).Error; err != nil {
		return nil, err
	}
	return session, nil
}

func NewPlanningSessionRun(canvasID, versionID uuid.UUID, entrypoint, factoryID, repository, modelKey string) *CanvasRun {
	now := time.Now()
	planning := map[string]any{
		"factory_id": factoryID,
		"repository": repository,
	}
	if key := strings.TrimSpace(modelKey); key != "" {
		planning["selectable_model_key"] = key
	}
	return &CanvasRun{
		ID:         uuid.New(),
		WorkflowID: canvasID,
		NodeID:     entrypoint,
		VersionID:  versionID,
		Callbacks: datatypes.JSONSlice[core.RunCallback]{
			{When: core.RunCallbackWhenPending, On: core.RunCallbackOnEntry, Hook: "onMessage"},
		},
		Input:     NewJSONValue(map[string]any{"planning_session": planning}),
		State:     CanvasRunStatePending,
		CreatedAt: &now,
		UpdatedAt: &now,
	}
}

func (s *FactoryPlanningSession) AttachAgentRun(tx *gorm.DB, runID uuid.UUID, modelKey string) error {
	if err := s.guardOpen(); err != nil {
		return err
	}
	s.CanvasRunID = &runID
	s.SelectableModelKey = strings.TrimSpace(modelKey)
	s.clearWait()
	s.clearSurvey()
	s.UpdatedAt = time.Now()
	return tx.Model(s).Updates(map[string]any{
		"canvas_run_id":        s.CanvasRunID,
		"selectable_model_key": s.SelectableModelKey,
		"wait_state":           s.WaitState,
		"wait_kind":            s.WaitKind,
		"wait_text":            s.WaitText,
		"wait_work_order_id":   s.WaitWorkOrderID,
		"wait_work_order_key":  s.WaitWorkOrderKey,
		"survey_id":            s.SurveyID,
		"survey":               s.Survey,
		"updated_at":           s.UpdatedAt,
	}).Error
}

func CountOpenPlanningSessions(tx *gorm.DB, organizationID, factoryID uuid.UUID) (int64, error) {
	var count int64
	err := tx.Model(&FactoryPlanningSession{}).
		Where("organization_id = ? AND factory_id = ? AND state <> ?", organizationID, factoryID, PlanningSessionStateEnded).
		Count(&count).Error
	return count, err
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
	if err := session.reloadMessages(tx); err != nil {
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

func EndPlanningSessionForFinishedRun(tx *gorm.DB, canvasRunID uuid.UUID, result string) error {
	if result != CanvasRunResultFailed && result != CanvasRunResultCancelled {
		return nil
	}
	session, err := FindPlanningSessionByRun(tx, canvasRunID)
	if errors.Is(err, ErrFactoryPlanningSessionNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return session.End(tx)
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

func FindPlanningCanvas(tx *gorm.DB, organizationID, factoryID uuid.UUID) (*Canvas, error) {
	var canvas Canvas
	err := tx.Where("organization_id = ? AND factory_id = ? AND name = ?", organizationID, factoryID, PlanningCanvasName).First(&canvas).Error
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
		s.resolveWait(PlanningWaitResult{Kind: PlanningWaitKindEnded})
	}
	return s.saveEndedState(tx)
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

func (s *FactoryPlanningSession) reload(tx *gorm.DB) error {
	return tx.Where("id = ?", s.ID).First(s).Error
}

func (s *FactoryPlanningSession) LockForUpdate(tx *gorm.DB) error {
	return s.lockAndReload(tx)
}

func (s *FactoryPlanningSession) lockAndReload(tx *gorm.DB) error {
	var locked FactoryPlanningSession
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", s.ID).First(&locked).Error
	if err != nil {
		return err
	}
	*s = locked
	return s.reloadMessages(tx)
}

func (s *FactoryPlanningSession) withLockedSession(tx *gorm.DB, run func(*gorm.DB) error) error {
	return tx.Transaction(func(inner *gorm.DB) error {
		if err := s.lockAndReload(inner); err != nil {
			return err
		}
		return run(inner)
	})
}

func (s *FactoryPlanningSession) guardOpen() error {
	if s.State == PlanningSessionStateEnded {
		return ErrFactoryPlanningSessionEnded
	}
	return nil
}

func (s *FactoryPlanningSession) setDraft(draft PlanningSessionDraft) {
	s.DraftTitle = strings.TrimSpace(draft.Title)
	s.DraftDescription = strings.TrimSpace(draft.Description)
	if id, err := uuid.Parse(strings.TrimSpace(draft.WorkOrderID)); err == nil {
		s.DraftWorkOrderID = &id
		return
	}
	s.DraftWorkOrderID = nil
}

func (s *FactoryPlanningSession) clearDraft() {
	s.setDraft(PlanningSessionDraft{})
}

func (s *FactoryPlanningSession) resolveWait(result PlanningWaitResult) {
	s.WaitState = PlanningWaitResolved
	s.WaitKind = result.Kind
	s.WaitText = result.Text
	s.WaitWorkOrderKey = result.WorkOrderKey
	if id, err := uuid.Parse(strings.TrimSpace(result.WorkOrderID)); err == nil {
		s.WaitWorkOrderID = &id
		return
	}
	s.WaitWorkOrderID = nil
}

func (s *FactoryPlanningSession) clearWait() {
	s.WaitState = PlanningWaitIdle
	s.WaitKind = ""
	s.WaitText = ""
	s.WaitWorkOrderKey = ""
	s.WaitWorkOrderID = nil
}

func (s *FactoryPlanningSession) clearSurvey() {
	s.SurveyID = nil
	s.Survey = datatypes.JSONType[PlanningSessionSurvey]{}
}

func (s *FactoryPlanningSession) saveEndedState(tx *gorm.DB) error {
	return tx.Model(s).Updates(map[string]any{
		"state":               s.State,
		"ended_at":            s.EndedAt,
		"updated_at":          s.UpdatedAt,
		"wait_state":          s.WaitState,
		"wait_kind":           s.WaitKind,
		"wait_text":           s.WaitText,
		"wait_work_order_id":  s.WaitWorkOrderID,
		"wait_work_order_key": s.WaitWorkOrderKey,
	}).Error
}

func (s *FactoryPlanningSession) saveSessionMutation(tx *gorm.DB) error {
	s.UpdatedAt = time.Now()
	return tx.Model(s).Updates(map[string]any{
		"draft_title":         s.DraftTitle,
		"draft_description":   s.DraftDescription,
		"draft_work_order_id": s.DraftWorkOrderID,
		"wait_state":          s.WaitState,
		"wait_kind":           s.WaitKind,
		"wait_text":           s.WaitText,
		"wait_work_order_id":  s.WaitWorkOrderID,
		"wait_work_order_key": s.WaitWorkOrderKey,
		"survey_id":           s.SurveyID,
		"survey":              s.Survey,
		"updated_at":          s.UpdatedAt,
	}).Error
}

func (s *FactoryPlanningSession) saveDraft(tx *gorm.DB) error {
	s.UpdatedAt = time.Now()
	return tx.Model(s).Updates(map[string]any{
		"draft_title":         s.DraftTitle,
		"draft_description":   s.DraftDescription,
		"draft_work_order_id": s.DraftWorkOrderID,
		"updated_at":          s.UpdatedAt,
	}).Error
}

func (s *FactoryPlanningSession) saveWait(tx *gorm.DB) error {
	s.UpdatedAt = time.Now()
	return tx.Model(s).Updates(map[string]any{
		"wait_state":          s.WaitState,
		"wait_kind":           s.WaitKind,
		"wait_text":           s.WaitText,
		"wait_work_order_id":  s.WaitWorkOrderID,
		"wait_work_order_key": s.WaitWorkOrderKey,
		"updated_at":          s.UpdatedAt,
	}).Error
}

func (s *FactoryPlanningSession) saveSurvey(tx *gorm.DB) error {
	s.UpdatedAt = time.Now()
	return tx.Model(s).Updates(map[string]any{
		"survey_id":  s.SurveyID,
		"survey":     s.Survey,
		"updated_at": s.UpdatedAt,
	}).Error
}
