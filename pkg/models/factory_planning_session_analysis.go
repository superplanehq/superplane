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
	"gorm.io/gorm/clause"
)

const (
	workOrderCreatedPayloadType              = "workOrder.created"
	WorkOrderCreatedRefinementEnabledDataKey = "taskRefinementEnabled"
)

const (
	PlanningSpecArtifactKey   = "spec"
	PlanningSpecArtifactTitle = "spec.md"

	PlanningConfidenceCheckKey  = "confidence"
	PlanningConfidenceCheckName = "Confidence score"
	PlanningConfidenceScoreMax  = 5

	// Rewinds keep a contiguous recent suffix. When history exceeds the hard
	// limit, retain 80 percent so the next turn has room for new output.
	analysisRewindMessageCharacterLimit = 24_000
	analysisRewindRetentionPercent      = 80
	analysisRewindTruncationMarker      = "\n… middle omitted from this rewind …\n"
)

type analysisMessageWindow struct {
	Messages []PlanningSessionMessage
	Omitted  int
}

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
	order, err := f.planningRefineWorkOrder(tx, params.WorkOrderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrFactoryPlanningSessionInvalid
	}

	existing, err := FindPlanningSessionByRun(tx, params.CanvasRunID)
	if err == nil {
		if !analysisSessionMatchesWorkOrder(existing, params.WorkOrderID) {
			return nil, ErrFactoryPlanningSessionInvalid
		}
		return existing, nil
	}
	if !errors.Is(err, ErrFactoryPlanningSessionNotFound) {
		return nil, err
	}

	existing, err = FindPlanningSessionByDraftWorkOrder(tx, f.OrganizationID, f.ID, params.WorkOrderID)
	if err == nil {
		if err := existing.LockForUpdate(tx); err != nil {
			return nil, err
		}
		if existing.State != PlanningSessionStateEnded && existing.hasActiveAnalysisRun(tx) {
			return existing, nil
		}
		if err := existing.Reopen(tx); err != nil {
			return nil, err
		}
		if err := existing.AttachAgentRun(tx, params.CanvasRunID, existing.SelectableModelKey); err != nil {
			return nil, err
		}
		return existing, nil
	}
	if !errors.Is(err, ErrFactoryPlanningSessionNotFound) {
		return nil, err
	}

	now := time.Now()
	canvasID := params.CanvasID
	runID := params.CanvasRunID
	workOrderID := order.ID
	session := &FactoryPlanningSession{
		ID:               uuid.New(),
		OrganizationID:   f.OrganizationID,
		FactoryID:        f.ID,
		CreatedByUserID:  params.CreatedByUserID,
		Repository:       strings.TrimSpace(params.Repository),
		Kind:             PlanningSessionKindWorkOrderAnalysis,
		State:            PlanningSessionStateRunning,
		CanvasID:         &canvasID,
		CanvasRunID:      &runID,
		DraftWorkOrderID: &workOrderID,
		HeartbeatAt:      now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(session)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return FindPlanningSessionByDraftWorkOrder(tx, f.OrganizationID, f.ID, params.WorkOrderID)
	}
	if err := session.attachRefineDraft(tx, order); err != nil {
		return nil, err
	}
	return session, nil
}

func analysisSessionMatchesWorkOrder(session *FactoryPlanningSession, workOrderID uuid.UUID) bool {
	return session.IsAnalysisSession() && session.DraftWorkOrderID != nil && *session.DraftWorkOrderID == workOrderID
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
	key := planningSpecArtifactKey(order.ID)
	data := map[string]any{
		"name":  PlanningSpecArtifactTitle,
		"title": PlanningSpecArtifactTitle,
		"body":  body,
	}
	if _, err := order.UpdateArtifactData(tx, key, data); err == nil {
		return nil
	} else if !errors.Is(err, ErrFactoryWorkOrderArtifactNotFound) {
		return err
	}
	_, err := order.CreateArtifact(tx, FactoryWorkOrderArtifactParams{
		Type: FactoryWorkOrderArtifactTypeMarkdown,
		Key:  key,
		Data: data,
	})
	return err
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
	window := analysisConversationWindow(messages, analysisRewindMessageCharacterLimit)

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
	if len(window.Messages) > 0 {
		b.WriteString("\nRecent messages retained for this rewind:\n")
		if window.Omitted > 0 {
			fmt.Fprintf(
				&b,
				"\n%d older messages are not in this rewind. The current specification and confidence above contain the durable task state.\n",
				window.Omitted,
			)
		}
		for _, message := range window.Messages {
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

func analysisConversationWindow(messages []PlanningSessionMessage, hardLimit int) analysisMessageWindow {
	if len(messages) == 0 || hardLimit <= 0 {
		return analysisMessageWindow{Omitted: len(messages)}
	}
	if analysisMessagesContextCharacters(messages) <= hardLimit {
		return analysisMessageWindow{Messages: messages}
	}

	target := max(1, hardLimit*analysisRewindRetentionPercent/100)
	selected := make([]PlanningSessionMessage, 0, len(messages))
	used := 0
	for index := len(messages) - 1; index >= 0; index-- {
		message := messages[index]
		cost := analysisMessageContextCharacters(message)
		if used+cost <= target {
			selected = append(selected, message)
			used += cost
			continue
		}
		if len(selected) == 0 {
			message.Text = truncateAnalysisMessageForRewind(message.Text, target-analysisMessageEnvelopeCharacters(message))
			selected = append(selected, message)
			index--
		}
		reversePlanningMessages(selected)
		return analysisMessageWindow{Messages: selected, Omitted: index + 1}
	}
	reversePlanningMessages(selected)
	return analysisMessageWindow{Messages: selected}
}

func analysisMessagesContextCharacters(messages []PlanningSessionMessage) int {
	total := 0
	for _, message := range messages {
		total += analysisMessageContextCharacters(message)
	}
	return total
}

func analysisMessageContextCharacters(message PlanningSessionMessage) int {
	return analysisMessageEnvelopeCharacters(message) + len([]rune(strings.TrimSpace(message.Text)))
}

func analysisMessageEnvelopeCharacters(message PlanningSessionMessage) int {
	role := "User"
	if message.Role == PlanningSessionMessageRoleAgent {
		role = "Agent"
	}
	return len([]rune(role)) + len(": \n")
}

func truncateAnalysisMessageForRewind(text string, limit int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	marker := []rune(analysisRewindTruncationMarker)
	if limit <= len(marker) {
		return string(runes[:max(0, limit)])
	}
	contentLimit := limit - len(marker)
	headLength := contentLimit / 2
	tailLength := contentLimit - headLength
	return string(runes[:headLength]) + analysisRewindTruncationMarker + string(runes[len(runes)-tailLength:])
}

func reversePlanningMessages(messages []PlanningSessionMessage) {
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
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
	key := planningSpecArtifactKey(order.ID)
	for i := range artifacts {
		if artifacts[i].Key == nil || *artifacts[i].Key != key {
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
		Where(
			"organization_id = ? AND factory_id = ? AND draft_work_order_id = ? AND kind = ?",
			organizationID,
			factoryID,
			workOrderID,
			PlanningSessionKindWorkOrderAnalysis,
		).
		Order("created_at DESC").
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
	if event.WorkflowID != canvas.ID || run.WorkflowID != canvas.ID || event.NodeID != FactoryAppBacklogTriggerID || run.NodeID != FactoryAppBacklogTriggerID {
		return nil
	}
	if event.RunID != uuid.Nil && event.RunID != run.ID {
		return nil
	}
	version, err := FindCanvasVersionInTransaction(tx, canvas.ID, run.VersionID)
	if err != nil {
		return err
	}
	if !IsBacklogFactoryApp(version.Nodes, version.Edges) {
		return nil
	}
	created, ok := parseAnalysisWorkOrderCreatedEvent(event)
	if !ok || !created.RefinementEnabled {
		return nil
	}
	factoryModel, err := FindFactory(tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return err
	}
	order, err := factoryModel.FindWorkOrder(tx, created.WorkOrderID)
	if err != nil {
		return err
	}
	if order.State != FactoryWorkOrderStateDraft {
		return nil
	}
	if order.CreatedByID == nil || *order.CreatedByID == uuid.Nil {
		return nil
	}
	repository := created.Repository
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

type analysisWorkOrderCreatedPayload struct {
	WorkOrderID       uuid.UUID
	Repository        string
	RefinementEnabled bool
}

func parseAnalysisWorkOrderCreatedEvent(event *CanvasEvent) (analysisWorkOrderCreatedPayload, bool) {
	raw, err := json.Marshal(event.Data.Data())
	if err != nil {
		return analysisWorkOrderCreatedPayload{}, false
	}
	var envelope struct {
		Type string `json:"type"`
		Data struct {
			RefinementEnabled bool `json:"taskRefinementEnabled"`
			WorkOrder         struct {
				ID         string `json:"id"`
				Repository string `json:"repository"`
			} `json:"workOrder"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return analysisWorkOrderCreatedPayload{}, false
	}
	if envelope.Type != workOrderCreatedPayloadType {
		return analysisWorkOrderCreatedPayload{}, false
	}
	id, err := uuid.Parse(strings.TrimSpace(envelope.Data.WorkOrder.ID))
	if err != nil {
		return analysisWorkOrderCreatedPayload{}, false
	}
	return analysisWorkOrderCreatedPayload{
		WorkOrderID:       id,
		Repository:        strings.TrimSpace(envelope.Data.WorkOrder.Repository),
		RefinementEnabled: envelope.Data.RefinementEnabled,
	}, true
}
