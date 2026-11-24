package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MaybeAttachCustomAutomationSession keeps each automation run's conversation
// separate from task planning and from other runs on the same task.
func MaybeAttachCustomAutomationSession(tx *gorm.DB, canvas *Canvas, event *CanvasEvent, run *CanvasRun) error {
	if canvas == nil || event == nil || run == nil || canvas.FactoryID == nil || event.WorkflowID != canvas.ID || run.WorkflowID != canvas.ID {
		return nil
	}
	version, err := FindCanvasVersionInTransaction(tx, canvas.ID, run.VersionID)
	if err != nil {
		return err
	}
	if FactoryAppTemplateID(version.Nodes) != FactoryAppTemplateCustomAutomationID {
		return nil
	}
	enabled := false
	for _, node := range version.Nodes {
		steps, ok := node.Configuration["steps"].([]any)
		if !ok || len(steps) == 0 {
			continue
		}
		step, _ := steps[len(steps)-1].(map[string]any)
		enabled = step["continueInChat"] == true && step["type"] == "prompt"
		if enabled {
			break
		}
	}
	if !enabled {
		return nil
	}
	if _, err := FindPlanningSessionByRun(tx, run.ID); err == nil {
		return nil
	} else if !errors.Is(err, ErrFactoryPlanningSessionNotFound) {
		return err
	}
	payload, _ := objectMap(RootEventSourcePayload(event.Data.Data()))
	task, _ := objectMap(payload["workOrder"])
	raw, _ := task["id"].(string)
	taskID, _ := uuid.Parse(raw)
	factory, err := FindFactory(tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return err
	}
	if taskID == uuid.Nil {
		prID := visualEvidencePullRequestID(event.Data.Data())
		if prID == uuid.Nil {
			return nil
		}
		pr, err := factory.FindPullRequest(tx, FactoryPullRequestLookup{ID: prID})
		if err != nil {
			return err
		}
		taskID = pr.WorkOrderID
	}
	order, err := factory.FindWorkOrder(tx, taskID)
	if err != nil {
		return err
	}
	now := time.Now()
	session := FactoryPlanningSession{
		ID: uuid.New(), OrganizationID: canvas.OrganizationID, FactoryID: factory.ID,
		Kind: PlanningSessionKindCustomAutomation, State: PlanningSessionStateRunning,
		CanvasID: &canvas.ID, CanvasRunID: &run.ID, OriginRunID: &run.ID,
		DraftWorkOrderID: &order.ID, DraftTitle: order.Title, DraftDescription: order.Description,
		HeartbeatAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if order.Repository != nil {
		session.Repository = *order.Repository
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&session).Error
}

func FindCustomAutomationSession(tx *gorm.DB, organizationID, factoryID, taskID, runID uuid.UUID) (*FactoryPlanningSession, error) {
	var session FactoryPlanningSession
	err := tx.Where("organization_id = ? AND factory_id = ? AND draft_work_order_id = ? AND kind = ?", organizationID, factoryID, taskID, PlanningSessionKindCustomAutomation).
		Where("origin_run_id = ? OR canvas_run_id = ? OR origin_run_id IN (?)", runID, runID, tx.Model(&CanvasEvent{}).Select("source_run_id").Where("run_id = ? AND execution_id IS NULL", runID)).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFactoryPlanningSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// CustomAutomationContinuationText supplies conversation history without
// importing planning instructions, scores, or draft mutation rules.
func CustomAutomationContinuationText(tx *gorm.DB, session *FactoryPlanningSession) (string, error) {
	messages, err := ListPlanningSessionMessages(tx, session.ID)
	if err != nil {
		return "", err
	}
	if len(messages) == 0 {
		return "", nil
	}
	window := analysisConversationWindow(messages, analysisRewindMessageCharacterLimit)
	var text strings.Builder
	text.WriteString("Continue this automation conversation using its original prompt. Respond to the latest user message. Do not repeat completed actions unless requested.\n\n")
	for _, message := range window.Messages {
		fmt.Fprintf(&text, "%s:\n%s\n\n", message.Role, message.Text)
	}
	return text.String(), nil
}

// RestartCustomAutomationSession replays the original event/version, preserving
// task authority. Dispatch limits resumed execution to the final prompt step.
func RestartCustomAutomationSession(tx *gorm.DB, session *FactoryPlanningSession) (*CanvasEvent, error) {
	if !session.IsCustomAutomationSession() || session.OriginRunID == nil || session.CanvasID == nil || session.DraftWorkOrderID == nil {
		return nil, ErrFactoryPlanningSessionInvalid
	}
	factory, err := FindFactory(tx, session.OrganizationID, session.FactoryID)
	if err != nil {
		return nil, err
	}
	order, err := factory.FindWorkOrder(tx, *session.DraftWorkOrderID)
	if err != nil {
		return nil, err
	}
	if err := order.RequireExecutionOwner(tx, true); err != nil {
		return nil, err
	}
	if err := order.ValidateSubscriptionCanvasModels(tx, *session.CanvasID, ""); err != nil {
		return nil, err
	}
	if _, err := FindCanvasInTransaction(tx, session.OrganizationID, *session.CanvasID); err != nil {
		return nil, err
	}
	original, err := FindCanvasRunInTransaction(tx, *session.CanvasID, *session.OriginRunID)
	if err != nil {
		return nil, err
	}
	var root CanvasEvent
	if err := tx.Where("run_id = ? AND execution_id IS NULL", original.ID).Order("created_at ASC").First(&root).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	run := CanvasRun{WorkflowID: original.WorkflowID, NodeID: original.NodeID, VersionID: original.VersionID, State: CanvasRunStateStarted, CreatedAt: &now, UpdatedAt: &now}
	if err := tx.Create(&run).Error; err != nil {
		return nil, err
	}
	event := CanvasEvent{WorkflowID: root.WorkflowID, NodeID: root.NodeID, Channel: root.Channel, Data: root.Data, RunID: run.ID, SourceRunID: session.OriginRunID, State: CanvasEventStatePending}
	if err := tx.Create(&event).Error; err != nil {
		return nil, err
	}
	if err := session.Reopen(tx); err != nil {
		return nil, err
	}
	if err := session.AttachAgentRun(tx, run.ID, session.SelectableModelKey); err != nil {
		return nil, err
	}
	return &event, nil
}
