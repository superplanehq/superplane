package contexts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type EventContext struct {
	tx             *gorm.DB
	node           *models.CanvasNode
	run            *models.CanvasRun
	maxPayloadSize int
	onNewEvents    func([]models.CanvasEvent)
}

func NewEventContext(tx *gorm.DB, node *models.CanvasNode, run *models.CanvasRun, onNewEvents func([]models.CanvasEvent)) *EventContext {
	return &EventContext{tx: tx, node: node, run: run, maxPayloadSize: config.MaxPayloadSize(), onNewEvents: onNewEvents}
}

func (s *EventContext) Emit(payloadType string, payload any) error {
	return s.emit(payloadType, payload, "")
}

func (s *EventContext) EmitDeduplicated(payloadType string, payload any, dedupKey string) error {
	return s.emit(payloadType, payload, dedupKey)
}

func (s *EventContext) emit(payloadType string, payload any, dedupKey string) error {
	structuredPayload := map[string]any{
		"type":      payloadType,
		"timestamp": time.Now(),
		"data":      payload,
	}

	data, err := json.Marshal(structuredPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	if len(data) > s.maxPayloadSize {
		return fmt.Errorf("event payload too large: %d bytes (max %d)", len(data), s.maxPayloadSize)
	}

	now := time.Now()

	//
	// We use RawMessage here to avoid a second marshal when GORM persists the JSON value.
	//
	event := models.CanvasEvent{
		WorkflowID: s.node.WorkflowID,
		NodeID:     s.node.NodeID,
		Channel:    "default",
		Data:       models.NewJSONValue(json.RawMessage(data)),
		State:      models.CanvasEventStatePending,
		CreatedAt:  &now,
	}

	if s.run != nil {
		event.RunID = s.run.ID
	}

	wrappedPayload := map[string]any{"data": payload}
	customName, err := s.resolveCustomName(wrappedPayload, structuredPayload)
	if err != nil {
		failed := fmt.Sprintf("Failed to resolve run title: %s", err.Error())
		event.CustomName = &failed
	} else if customName != nil {
		event.CustomName = customName
	}

	created := true
	if dedupKey == "" {
		var suppress bool
		dedupKey, suppress = customAutomationWebhookPolicy(s.tx, s.node, payloadType, payload)
		if suppress {
			return nil
		}
	}
	if dedupKey != "" && s.run == nil {
		created, err = s.node.CreateDeduplicatedRootEvent(s.tx, &event, dedupKey)
	} else if payloadType == "github.pullRequest" && s.run == nil && s.node.Type == models.NodeTypeTrigger && s.node.Ref.Data().Trigger != nil && s.node.Ref.Data().Trigger.Name == "github.onPullRequest" {
		created, err = s.node.CreateGitHubChecksEvent(s.tx, &event)
	} else {
		err = s.tx.Create(&event).Error
	}
	if err != nil {
		if s.run == nil && errors.Is(err, gorm.ErrRecordNotFound) {
			canvas, findErr := models.FindUnscopedCanvasInTransaction(s.tx, s.node.WorkflowID)
			if findErr != nil {
				canvas = nil
			}
			logging.WithCanvasWorkspace(logging.ForNode(*s.node), canvas).
				Warnf("skipping event %s: canvas %s cannot start a run", payloadType, s.node.WorkflowID)
			return nil
		}

		return err
	}

	if !created {
		return nil
	}

	if s.onNewEvents != nil {
		s.onNewEvents([]models.CanvasEvent{event})
	}

	return nil
}

func customAutomationWebhookPolicy(tx *gorm.DB, node *models.CanvasNode, payloadType string, payload any) (string, bool) {
	if node == nil || node.Type != models.NodeTypeTrigger ||
		(payloadType != "github.pullRequest" && payloadType != "bitbucket.pullRequest") {
		return "", false
	}

	template, _ := node.Metadata.Data()[models.FactoryAppTemplateMetadataKey].(map[string]any)
	templateID, _ := template["id"].(string)
	startMode, _ := template["startMode"].(string)
	if templateID != models.FactoryAppTemplateCustomAutomationID {
		return "", false
	}
	if startMode != "automatic" {
		return "", true
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", true
	}
	var identity struct {
		PullRequest struct {
			ID string `json:"id"`
		} `json:"pullRequest"`
		WorkOrder struct {
			ID string `json:"id"`
		} `json:"workOrder"`
		GitHub struct {
			Head struct {
				SHA string `json:"sha"`
			} `json:"head"`
		} `json:"pull_request"`
		Bitbucket struct {
			Source struct {
				Commit struct {
					Hash string `json:"hash"`
				} `json:"commit"`
			} `json:"source"`
		} `json:"pullrequest"`
	}
	if err := json.Unmarshal(raw, &identity); err != nil {
		return "", true
	}
	pullRequestID := strings.TrimSpace(identity.PullRequest.ID)
	revision := strings.TrimSpace(identity.GitHub.Head.SHA)
	if revision == "" {
		revision = strings.TrimSpace(identity.Bitbucket.Source.Commit.Hash)
	}
	if pullRequestID == "" || revision == "" {
		return "", true
	}
	workOrderID, err := uuid.Parse(strings.TrimSpace(identity.WorkOrder.ID))
	if err != nil {
		return "", true
	}
	inVerify, err := models.WorkOrderIsWaitingAfterPassedStage(tx, workOrderID)
	if err != nil || !inVerify {
		return "", true
	}
	return pullRequestID + "|" + revision, false
}

func (s *EventContext) resolveCustomName(payload any, rootPayload any) (*string, error) {
	config := s.node.Configuration.Data()
	if config == nil {
		return nil, nil
	}

	rawTemplate, ok := config["customName"]
	if !ok || rawTemplate == nil {
		return nil, nil
	}

	template, ok := rawTemplate.(string)
	if !ok {
		return nil, nil
	}

	template = strings.TrimSpace(template)
	if template == "" {
		return nil, nil
	}

	builder := NewNodeConfigurationBuilder(s.tx, s.node.WorkflowID).
		WithNodeID(s.node.NodeID).
		WithInput(map[string]any{s.node.NodeID: payload}).
		WithRootPayload(rootPayload)
	resolved, err := builder.ResolveTemplateExpressions(template)
	if err != nil {
		return nil, err
	}

	resolvedName := strings.TrimSpace(fmt.Sprintf("%v", resolved))
	if resolvedName == "" {
		return nil, nil
	}

	return &resolvedName, nil
}
