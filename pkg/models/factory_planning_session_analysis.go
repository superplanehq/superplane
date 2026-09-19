package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	workOrderCreatedPayloadType              = "workOrder.created"
	WorkOrderCreatedRefinementEnabledDataKey = "taskRefinementEnabled"
)

const (
	PlanningSpecArtifactKey         = "spec"
	PlanningSpecArtifactTitle       = "spec.md"
	PlanningSpecArtifactCanvasRunID = "canvasRunId"

	// Clarity is how well the task is defined. Confidence is how likely a
	// coding agent completes the task in one run without steering.
	PlanningClarityCheckKey     = "clarity"
	PlanningClarityCheckName    = "Clarity score"
	PlanningConfidenceCheckKey  = "confidence"
	PlanningConfidenceCheckName = "Confidence score"
	PlanningScoreMax            = 5

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
	Repository  string
	CanvasID    uuid.UUID
	CanvasRunID uuid.UUID
	WorkOrderID uuid.UUID
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
		stored, err := planningSpecMarkdownForStorage(inner, order, markdown)
		if err != nil {
			return err
		}
		return upsertPlanningSpecArtifact(inner, order, stored, s.CanvasRunID)
	})
}

func planningSpecMarkdownForStorage(tx *gorm.DB, order *FactoryWorkOrder, markdown string) (string, error) {
	restored, err := RestoreFileRefs(tx, order.OrganizationID, order.FactoryID, order.ID, markdown)
	if err != nil {
		return "", err
	}
	return appendMissingDescriptionFileRefs(tx, order, restored)
}

func appendMissingDescriptionFileRefs(tx *gorm.DB, order *FactoryWorkOrder, spec string) (string, error) {
	present := map[uuid.UUID]struct{}{}
	for _, id := range blob.FileIDsInMarkdown(spec) {
		present[id] = struct{}{}
	}
	var missing []uuid.UUID
	for _, id := range blob.FileIDsInMarkdown(order.Description) {
		if _, ok := present[id]; ok {
			continue
		}
		missing = append(missing, id)
	}
	if len(missing) == 0 {
		return spec, nil
	}

	files, err := ListFilesByIDs(tx, missing)
	if err != nil {
		return "", err
	}
	byID := map[uuid.UUID]File{}
	for _, file := range files {
		byID[file.ID] = file
	}

	var b strings.Builder
	b.WriteString(strings.TrimRight(spec, "\n"))
	appended := false
	for _, id := range missing {
		file, ok := byID[id]
		if !ok || !file.IsDispatchable(order.OrganizationID, order.FactoryID, order.ID) {
			continue
		}
		b.WriteString("\n\n")
		b.WriteString(markdownFileRef(file))
		appended = true
	}
	if !appended {
		return spec, nil
	}
	b.WriteByte('\n')
	return b.String(), nil
}

func markdownFileRef(file File) string {
	ref := blob.FileRef(file.ID)
	label := blob.MarkdownLinkLabel(file.Filename)
	if IsInlineMediaContentType(file.ContentType) {
		return fmt.Sprintf("![%s](%s)", label, ref)
	}
	return fmt.Sprintf("[%s](%s)", label, ref)
}

// planningScoreKind names one of the two 1 through 5 scores a refine session
// publishes as a work-order check.
type planningScoreKind struct {
	key  string
	name string
}

var (
	planningClarityScore    = planningScoreKind{key: PlanningClarityCheckKey, name: PlanningClarityCheckName}
	planningConfidenceScore = planningScoreKind{key: PlanningConfidenceCheckKey, name: PlanningConfidenceCheckName}
)

// ProposeClarity publishes how well the task is defined.
func (s *FactoryPlanningSession) ProposeClarity(tx *gorm.DB, score float64, summary string) error {
	return s.proposePlanningScore(tx, planningClarityScore, score, summary)
}

// ProposeConfidence publishes how likely a coding agent completes the task in
// one run without steering.
func (s *FactoryPlanningSession) ProposeConfidence(tx *gorm.DB, score float64, summary string) error {
	return s.proposePlanningScore(tx, planningConfidenceScore, score, summary)
}

func (s *FactoryPlanningSession) proposePlanningScore(tx *gorm.DB, kind planningScoreKind, score float64, summary string) error {
	if err := validatePlanningScore(kind, score); err != nil {
		return err
	}
	return s.withLockedSession(tx, func(inner *gorm.DB) error {
		if err := s.guardOpen(); err != nil {
			return err
		}
		order, err := s.analysisWorkOrder(inner)
		if err != nil {
			return err
		}
		return reportPlanningScore(inner, s, order, kind, score, summary)
	})
}

func validatePlanningScore(kind planningScoreKind, score float64) error {
	if !isFiniteCheckNumber(score) || score < 1 || score > PlanningScoreMax {
		return fmt.Errorf("%w: %s score must be 1 through 5", ErrFactoryPlanningSessionInvalid, kind.key)
	}
	return nil
}

func reportPlanningScore(tx *gorm.DB, session *FactoryPlanningSession, order *FactoryWorkOrder, kind planningScoreKind, score float64, summary string) error {
	var run *factory.RunRef
	if session.CanvasRunID != nil {
		run = &factory.RunRef{ID: *session.CanvasRunID}
	}
	_, err := order.ReportCheck(tx, FactoryWorkOrderCheckParams{
		Key:      kind.key,
		Name:     kind.name,
		Score:    score,
		MaxScore: PlanningScoreMax,
		Format:   FactoryWorkOrderCheckFormatFraction,
		Level:    planningScoreLevel(score),
		Summary:  strings.TrimSpace(summary),
		Run:      run,
	})
	return err
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

func upsertPlanningSpecArtifact(tx *gorm.DB, order *FactoryWorkOrder, body string, runID *uuid.UUID) error {
	key := planningSpecArtifactKey(order.ID)
	data := map[string]any{
		"name":  PlanningSpecArtifactTitle,
		"title": PlanningSpecArtifactTitle,
		"body":  body,
	}
	if runID != nil && *runID != uuid.Nil {
		data[PlanningSpecArtifactCanvasRunID] = runID.String()
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

func planningScoreLevel(score float64) string {
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
	artifacts, err := analysisContinuationArtifacts(tx, session)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := writeSplitSiblingsBlock(&b, tx, session); err != nil {
		return "", err
	}
	if artifacts.spec == "" && artifacts.clarity.score == "" && artifacts.confidence.score == "" && len(messages) == 0 {
		return b.String(), nil
	}
	window := analysisConversationWindow(messages, analysisRewindMessageCharacterLimit)

	if b.Len() > 0 {
		b.WriteString("\n")
	}
	b.WriteString("Continue this SuperPlane analysis session. Do not greet as if the session is new. Follow the task prompt for tone, Clarity and Confidence rules, and specification shape. Call propose_clarity and propose_confidence every turn. If you write or update a specification this turn, call propose_spec before you stop. Do not leave a written plan unpublished. Call survey only when the task prompt says to ask. You may update a score without rewriting the specification. Apply the latest user message.\n")
	if artifacts.spec != "" {
		b.WriteString("\nCurrent specification:\n\n")
		b.WriteString(artifacts.spec)
		b.WriteString("\n")
	}
	writePlanningScoreBlock(&b, "Clarity", artifacts.clarity)
	writePlanningScoreBlock(&b, "Confidence", artifacts.confidence)
	if len(window.Messages) > 0 {
		b.WriteString("\nRecent messages retained for this rewind:\n")
		if window.Omitted > 0 {
			fmt.Fprintf(
				&b,
				"\n%d older messages are not in this rewind. The current specification and scores above contain the durable task state.\n",
				window.Omitted,
			)
		}
		for _, message := range window.Messages {
			fmt.Fprintf(&b, "\n%s: %s\n", rewindMessageRole(message), rewindMessageText(message))
		}
	}
	if err := writeCreatedTasksBlock(&b, tx, session); err != nil {
		return "", err
	}
	b.WriteString("\nApply the latest user message. Do not rewrite the specification from scratch unless the new context requires it.\n")
	return b.String(), nil
}

// writeCreatedTasksBlock lists every task this session split off its draft.
// It reads the durable session links, not the chat, so a task stays listed
// after its chat message falls out of the bounded rewind window.
func writeCreatedTasksBlock(b *strings.Builder, tx *gorm.DB, session *FactoryPlanningSession) error {
	created, err := session.SplitTaskOrders(tx)
	if err != nil || len(created) == 0 {
		return err
	}
	factoryModel, err := FindFactory(tx, session.OrganizationID, session.FactoryID)
	if err != nil {
		return err
	}
	b.WriteString("\nTasks this session already created. Do not create them again:\n")
	for _, order := range created {
		fmt.Fprintf(b, "\n- %s: %s", factoryModel.WorkOrderKey(order.Number), order.Title)
	}
	b.WriteString("\n\nCreate a task only for a part that is not listed above.\n")
	return nil
}

// writeSplitSiblingsBlock tells the agent that its draft is one part of a
// split and names the other parts, so it plans against that boundary
// instead of asking about work another part owns.
func writeSplitSiblingsBlock(b *strings.Builder, tx *gorm.DB, session *FactoryPlanningSession) error {
	siblings, err := session.SplitSiblings(tx)
	if err != nil || len(siblings) == 0 {
		return err
	}
	factoryModel, err := FindFactory(tx, session.OrganizationID, session.FactoryID)
	if err != nil {
		return err
	}
	b.WriteString("This task is one part of a split. The other parts are separate tasks, each with its own refinement and its own run:\n")
	for _, sibling := range siblings {
		fmt.Fprintf(b, "\n- %s: %s", factoryModel.WorkOrderKey(sibling.Order.Number), sibling.Order.Title)
		if sibling.Parent {
			b.WriteString(" (the task it was split from)")
		}
	}
	b.WriteString("\n\nTreat the boundary between the parts as decided. Work that another part owns is not missing and is not an open question: do not ask how to handle it, do not propose a stub for it, and do not lower Confidence because it is not in the repository yet. Plan against the interface the task description gives. Say once, in chat, which part you depend on. Do not create a task for work a listed part already owns.\n")
	return nil
}

func analysisConversationWindow(messages []PlanningSessionMessage, hardLimit int) analysisMessageWindow {
	messages = filterPlanningRewindMessages(messages)
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

func filterPlanningRewindMessages(messages []PlanningSessionMessage) []PlanningSessionMessage {
	filtered := make([]PlanningSessionMessage, 0, len(messages))
	for _, message := range messages {
		if message.Role == PlanningSessionMessageRolePlan {
			continue
		}
		filtered = append(filtered, message)
	}
	return filtered
}

func analysisMessagesContextCharacters(messages []PlanningSessionMessage) int {
	total := 0
	for _, message := range messages {
		total += analysisMessageContextCharacters(message)
	}
	return total
}

func analysisMessageContextCharacters(message PlanningSessionMessage) int {
	return analysisMessageEnvelopeCharacters(message) + len([]rune(rewindMessageText(message)))
}

func analysisMessageEnvelopeCharacters(message PlanningSessionMessage) int {
	return len([]rune(rewindMessageRole(message))) + len(": \n")
}

// rewindMessageRole names the speaker of a message in the rewind prompt.
// Task messages come from SuperPlane, not from either side of the chat.
func rewindMessageRole(message PlanningSessionMessage) string {
	switch message.Role {
	case PlanningSessionMessageRoleAgent:
		return "Agent"
	case PlanningSessionMessageRoleTask:
		return "SuperPlane"
	default:
		return "User"
	}
}

// rewindMessageText renders a message for the rewind prompt. A task message
// becomes a short sentence, so the agent sees the key and title, not JSON.
func rewindMessageText(message PlanningSessionMessage) string {
	if message.Role != PlanningSessionMessageRoleTask {
		return strings.TrimSpace(message.Text)
	}
	task, ok := ParsePlanningTaskMessage(message.Text)
	if !ok {
		return "Created a task."
	}
	return fmt.Sprintf("Created task %s: %s", task.Key, task.Title)
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

type planningScoreText struct {
	score   string
	summary string
}

type analysisContinuationState struct {
	spec       string
	clarity    planningScoreText
	confidence planningScoreText
}

func writePlanningScoreBlock(b *strings.Builder, label string, text planningScoreText) {
	if text.score == "" {
		return
	}
	b.WriteString("\nCurrent ")
	b.WriteString(label)
	b.WriteString(": ")
	b.WriteString(text.score)
	if text.summary != "" {
		b.WriteString("\n")
		b.WriteString(text.summary)
	}
	b.WriteString("\n")
}

func analysisContinuationArtifacts(tx *gorm.DB, session *FactoryPlanningSession) (analysisContinuationState, error) {
	var state analysisContinuationState
	if session.DraftWorkOrderID == nil {
		return state, nil
	}
	order, err := session.analysisWorkOrder(tx)
	if err != nil {
		if errors.Is(err, ErrFactoryPlanningSessionNoDraft) {
			return state, nil
		}
		return state, err
	}
	state.spec, err = planningSpecBody(tx, order)
	if err != nil {
		return state, err
	}
	checks, err := order.ListChecks(tx)
	if err != nil {
		return state, err
	}
	state.clarity = planningScoreTextFromChecks(checks, PlanningClarityCheckKey)
	state.confidence = planningScoreTextFromChecks(checks, PlanningConfidenceCheckKey)
	return state, nil
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

func planningScoreTextFromChecks(checks []FactoryWorkOrderCheck, key string) planningScoreText {
	for i := range checks {
		if checks[i].Key != key {
			continue
		}
		score := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", checks[i].Score), "0"), ".")
		if score == "" {
			score = "0"
		}
		return planningScoreText{score: score + "/5", summary: strings.TrimSpace(checks[i].Summary)}
	}
	return planningScoreText{}
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
	repository := created.Repository
	if repository == "" && order.Repository != nil {
		repository = strings.TrimSpace(*order.Repository)
	}
	if repository == "" {
		repository = strings.TrimSpace(factoryModel.OnboardingConfigValue().AppRepository)
	}
	_, err = factoryModel.AttachAnalysisSession(tx, AttachAnalysisSessionParams{
		Repository:  repository,
		CanvasID:    canvas.ID,
		CanvasRunID: run.ID,
		WorkOrderID: order.ID,
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
