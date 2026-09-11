package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

const workOrderCreatedPayloadType = "workOrder.created"

const (
	PlanningSpecArtifactKey   = "spec"
	PlanningSpecArtifactTitle = "spec.md"

	PlanningConfidenceCheckKey  = "confidence"
	PlanningConfidenceCheckName = "Confidence score"
	PlanningConfidenceScoreMax  = 5

	analysisContinuationMessageLimit = 20
)

type AttachAnalysisSessionParams struct {
	CreatedByUserID uuid.UUID
	Repository      string
	CanvasID        uuid.UUID
	CanvasRunID     uuid.UUID
	WorkOrderID     uuid.UUID
}

func (f *Factory) AttachAnalysisSession(tx *gorm.DB, params AttachAnalysisSessionParams) (*FactoryPlanningSession, error) {
	if params.CanvasID == uuid.Nil || params.CanvasRunID == uuid.Nil || params.WorkOrderID == uuid.Nil {
		return nil, ErrFactoryPlanningSessionInvalid
	}

	existing, err := FindPlanningSessionByRun(tx, params.CanvasRunID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrFactoryPlanningSessionNotFound) {
		return nil, err
	}

	existing, err = FindPlanningSessionByDraftWorkOrder(tx, f.OrganizationID, f.ID, params.WorkOrderID)
	if err == nil {
		if existing.State == PlanningSessionStateEnded {
			if err := existing.Reopen(tx); err != nil {
				return nil, err
			}
		}
		if err := existing.AttachAgentRun(tx, params.CanvasRunID, existing.SelectableModelKey); err != nil {
			return nil, err
		}
		return existing, nil
	}
	if !errors.Is(err, ErrFactoryPlanningSessionNotFound) {
		return nil, err
	}

	order, err := f.planningRefineWorkOrder(tx, params.WorkOrderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrFactoryPlanningSessionInvalid
	}

	now := time.Now()
	canvasID := params.CanvasID
	runID := params.CanvasRunID
	session := &FactoryPlanningSession{
		ID:              uuid.New(),
		OrganizationID:  f.OrganizationID,
		FactoryID:       f.ID,
		CreatedByUserID: params.CreatedByUserID,
		Repository:      strings.TrimSpace(params.Repository),
		State:           PlanningSessionStateRunning,
		CanvasID:        &canvasID,
		CanvasRunID:     &runID,
		HeartbeatAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := tx.Create(session).Error; err != nil {
		return nil, err
	}
	if err := session.attachRefineDraft(tx, order); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *FactoryPlanningSession) ProposeSpec(tx *gorm.DB, body string) error {
	markdown := strings.TrimSpace(body)
	if markdown == "" {
		return fmt.Errorf("%w: spec body is required", ErrFactoryPlanningSessionInvalid)
	}
	return s.withLockedSession(tx, func(inner *gorm.DB) error {
		if err := s.guardOpen(); err != nil {
			return err
		}
		order, err := s.analysisWorkOrder(inner)
		if err != nil {
			return err
		}
		return upsertPlanningSpecArtifact(inner, order, markdown)
	})
}

func (s *FactoryPlanningSession) ProposeConfidence(tx *gorm.DB, score float64, summary string) error {
	if score < 0 || score > PlanningConfidenceScoreMax {
		return fmt.Errorf("%w: confidence score must be 0 through 5", ErrFactoryPlanningSessionInvalid)
	}
	return s.withLockedSession(tx, func(inner *gorm.DB) error {
		if err := s.guardOpen(); err != nil {
			return err
		}
		order, err := s.analysisWorkOrder(inner)
		if err != nil {
			return err
		}
		var run *factory.RunRef
		if s.CanvasRunID != nil {
			run = &factory.RunRef{ID: *s.CanvasRunID}
		}
		_, err = order.ReportCheck(inner, FactoryWorkOrderCheckParams{
			Key:      PlanningConfidenceCheckKey,
			Name:     PlanningConfidenceCheckName,
			Score:    score,
			MaxScore: PlanningConfidenceScoreMax,
			Format:   FactoryWorkOrderCheckFormatFraction,
			Level:    planningConfidenceLevel(score),
			Summary:  strings.TrimSpace(summary),
			Run:      run,
		})
		return err
	})
}

func (s *FactoryPlanningSession) analysisWorkOrder(tx *gorm.DB) (*FactoryWorkOrder, error) {
	if s.DraftWorkOrderID == nil {
		return nil, ErrFactoryPlanningSessionNoDraft
	}
	factoryModel, err := FindFactory(tx, s.OrganizationID, s.FactoryID)
	if err != nil {
		return nil, err
	}
	return factoryModel.FindWorkOrder(tx, *s.DraftWorkOrderID)
}

func planningSpecArtifactKey(orderID uuid.UUID) string {
	return PlanningSpecArtifactKey + ":" + orderID.String()
}

func upsertPlanningSpecArtifact(tx *gorm.DB, order *FactoryWorkOrder, body string) error {
	data := map[string]any{
		"name":  PlanningSpecArtifactTitle,
		"title": PlanningSpecArtifactTitle,
		"body":  body,
	}
	artifacts, err := order.ListArtifacts(tx)
	if err != nil {
		return err
	}
	updated := false
	for i := range artifacts {
		if !isPlanningSpecArtifact(artifacts[i]) {
			continue
		}
		if err := writePlanningSpecArtifact(tx, order, &artifacts[i], data); err != nil {
			return err
		}
		updated = true
	}
	if updated {
		return nil
	}
	_, err = order.CreateArtifact(tx, FactoryWorkOrderArtifactParams{
		Type: FactoryWorkOrderArtifactTypeMarkdown,
		Key:  planningSpecArtifactKey(order.ID),
		Data: data,
	})
	return err
}

func isPlanningSpecArtifact(artifact FactoryWorkOrderArtifact) bool {
	var data map[string]any
	if json.Unmarshal(artifact.Data, &data) != nil {
		return false
	}
	name := extractArtifactString(data, "name")
	title := extractArtifactString(data, "title")
	return name == PlanningSpecArtifactTitle || title == PlanningSpecArtifactTitle
}

func writePlanningSpecArtifact(
	tx *gorm.DB,
	order *FactoryWorkOrder,
	artifact *FactoryWorkOrderArtifact,
	data map[string]any,
) error {
	if artifact.Key != nil && strings.TrimSpace(*artifact.Key) != "" {
		_, err := order.UpdateArtifactData(tx, *artifact.Key, data)
		return err
	}
	if err := validateArtifactData(artifact.Type, data); err != nil {
		return err
	}
	dataJSON, err := encodeArtifactData(data)
	if err != nil {
		return err
	}
	if len(dataJSON) > MaxFactoryWorkOrderArtifactDataBytes {
		return fmt.Errorf(
			"%w: artifact data exceeds %d bytes",
			ErrFactoryWorkOrderArtifactInvalid,
			MaxFactoryWorkOrderArtifactDataBytes,
		)
	}
	if err := tx.Model(artifact).Update("data", dataJSON).Error; err != nil {
		return err
	}
	artifact.Data = dataJSON
	return nil
}

func planningConfidenceLevel(score float64) string {
	if score >= 4 {
		return FactoryWorkOrderCheckLevelPositive
	}
	if score >= 3 {
		return FactoryWorkOrderCheckLevelCaution
	}
	return FactoryWorkOrderCheckLevelCritical
}

func AnalysisContinuationText(tx *gorm.DB, session *FactoryPlanningSession) (string, error) {
	if session == nil {
		return "", nil
	}
	messages := session.Messages
	if len(messages) == 0 {
		loaded, err := ListPlanningSessionMessages(tx, session.ID)
		if err != nil {
			return "", err
		}
		messages = loaded
	}
	spec, score, summary, err := analysisContinuationArtifacts(tx, session)
	if err != nil {
		return "", err
	}
	if spec == "" && score == "" && len(messages) == 0 {
		return "", nil
	}
	if len(messages) > analysisContinuationMessageLimit {
		messages = messages[len(messages)-analysisContinuationMessageLimit:]
	}

	var b strings.Builder
	b.WriteString("Continue this SuperPlane analysis session. Do not greet as if the session is new. Update the current specification and the score when the new context changes them.\n")
	if spec != "" {
		b.WriteString("\nCurrent specification:\n\n")
		b.WriteString(spec)
		b.WriteString("\n")
	}
	if score != "" {
		b.WriteString("\nCurrent confidence score: ")
		b.WriteString(score)
		if summary != "" {
			b.WriteString("\n")
			b.WriteString(summary)
		}
		b.WriteString("\n")
	}
	if len(messages) > 0 {
		b.WriteString("\nPrior messages:\n")
		for _, message := range messages {
			role := "User"
			if message.Role == PlanningSessionMessageRoleAgent {
				role = "Agent"
			}
			fmt.Fprintf(&b, "\n%s: %s\n", role, strings.TrimSpace(message.Text))
		}
	}
	b.WriteString("\nApply the latest user message. Do not rewrite the specification from scratch unless the new context requires it.\n")
	return b.String(), nil
}

func analysisContinuationArtifacts(tx *gorm.DB, session *FactoryPlanningSession) (string, string, string, error) {
	if session.DraftWorkOrderID == nil {
		return "", "", "", nil
	}
	order, err := session.analysisWorkOrder(tx)
	if err != nil {
		if errors.Is(err, ErrFactoryPlanningSessionNoDraft) {
			return "", "", "", nil
		}
		return "", "", "", err
	}
	spec, err := planningSpecBody(tx, order)
	if err != nil {
		return "", "", "", err
	}
	score, summary, err := planningConfidenceText(tx, order)
	if err != nil {
		return "", "", "", err
	}
	return spec, score, summary, nil
}

func planningSpecBody(tx *gorm.DB, order *FactoryWorkOrder) (string, error) {
	artifacts, err := order.ListArtifacts(tx)
	if err != nil {
		return "", err
	}
	for i := range artifacts {
		if !isPlanningSpecArtifact(artifacts[i]) {
			continue
		}
		var data map[string]any
		if json.Unmarshal(artifacts[i].Data, &data) != nil {
			continue
		}
		if body := extractArtifactString(data, "body"); body != "" {
			return body, nil
		}
	}
	return "", nil
}

func planningConfidenceText(tx *gorm.DB, order *FactoryWorkOrder) (string, string, error) {
	checks, err := order.ListChecks(tx)
	if err != nil {
		return "", "", err
	}
	for i := range checks {
		if checks[i].Key != PlanningConfidenceCheckKey {
			continue
		}
		score := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", checks[i].Score), "0"), ".")
		if score == "" {
			score = "0"
		}
		return score + "/5", strings.TrimSpace(checks[i].Summary), nil
	}
	return "", "", nil
}

func FindPlanningSessionByDraftWorkOrder(
	tx *gorm.DB,
	organizationID, factoryID, workOrderID uuid.UUID,
) (*FactoryPlanningSession, error) {
	if workOrderID == uuid.Nil {
		return nil, ErrFactoryPlanningSessionNotFound
	}
	var session FactoryPlanningSession
	err := tx.
		Joins("LEFT JOIN workflows ON workflows.id = factory_planning_sessions.canvas_id AND workflows.deleted_at IS NULL").
		Where(
			"factory_planning_sessions.organization_id = ? AND factory_planning_sessions.factory_id = ? AND factory_planning_sessions.draft_work_order_id = ?",
			organizationID,
			factoryID,
			workOrderID,
		).
		Where("workflows.name IS NULL OR workflows.name <> ?", PlanningCanvasName).
		Order("factory_planning_sessions.created_at DESC").
		First(&session).Error
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

func MaybeAttachAnalysisSession(tx *gorm.DB, canvas *Canvas, event *CanvasEvent, run *CanvasRun) error {
	if canvas == nil || event == nil || run == nil || canvas.FactoryID == nil {
		return nil
	}
	workOrderID, repository, ok := parseWorkOrderCreatedEvent(event)
	if !ok {
		return nil
	}
	factoryModel, err := FindFactory(tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return err
	}
	order, err := factoryModel.FindWorkOrder(tx, workOrderID)
	if err != nil {
		return err
	}
	if order.State != FactoryWorkOrderStateDraft {
		return nil
	}
	if order.CreatedByID == nil || *order.CreatedByID == uuid.Nil {
		return nil
	}
	if repository == "" && order.Repository != nil {
		repository = strings.TrimSpace(*order.Repository)
	}
	if repository == "" {
		repository = strings.TrimSpace(factoryModel.OnboardingConfigValue().AppRepository)
	}
	_, err = factoryModel.AttachAnalysisSession(tx, AttachAnalysisSessionParams{
		CreatedByUserID: *order.CreatedByID,
		Repository:      repository,
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
	})
	return err
}

func parseWorkOrderCreatedEvent(event *CanvasEvent) (uuid.UUID, string, bool) {
	raw, err := json.Marshal(event.Data.Data())
	if err != nil {
		return uuid.Nil, "", false
	}
	var envelope struct {
		Type string `json:"type"`
		Data struct {
			WorkOrder struct {
				ID         string `json:"id"`
				Repository string `json:"repository"`
			} `json:"workOrder"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return uuid.Nil, "", false
	}
	if envelope.Type != workOrderCreatedPayloadType {
		return uuid.Nil, "", false
	}
	id, err := uuid.Parse(strings.TrimSpace(envelope.Data.WorkOrder.ID))
	if err != nil {
		return uuid.Nil, "", false
	}
	return id, strings.TrimSpace(envelope.Data.WorkOrder.Repository), true
}
