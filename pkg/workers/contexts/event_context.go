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
	var mergeConfidence mergeConfidenceWebhookIdentity
	if dedupKey == "" {
		mergeConfidence = mergeConfidenceWebhookDedupKey(s.node, payloadType, payload)
	}
	if dedupKey == "" && mergeConfidence.Key == "" {
		var suppress bool
		dedupKey, suppress = customAutomationWebhookPolicy(s.tx, s.node, payloadType, payload)
		if suppress {
			return nil
		}
	}
	if mergeConfidence.Key != "" && s.run == nil {
		created, err = s.node.CreateRootEventUnlessLatestInScope(s.tx, &event, mergeConfidence.Scope, mergeConfidence.Key, mergeConfidence.isRedundantAfter)
	} else if dedupKey != "" && s.run == nil {
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

// mergeConfidenceWebhookIdentity identifies the diff that an automatic
// Bitbucket merge confidence start assesses.
type mergeConfidenceWebhookIdentity struct {
	// Scope is the dedup key prefix for all starts of one pull request.
	Scope string
	// Key identifies the head commit and the destination branch.
	Key string
	// UpdatedAt is the Bitbucket update time of the pull request in the event.
	UpdatedAt time.Time
}

// isRedundantAfter reports whether the newest start for the pull request makes
// this start unnecessary. A start for the same diff is redundant, so a
// description update cannot start a loop. A delayed webhook for an earlier
// pull request state is also redundant. Without this check, an old head could
// start after the current head and hide the current result.
func (m mergeConfidenceWebhookIdentity) isRedundantAfter(latest models.CanvasEvent) bool {
	if latest.DedupKey != nil && *latest.DedupKey == m.Key {
		return true
	}
	latestUpdatedAt, ok := bitbucketPullRequestUpdatedAt(latest.Data.Data())
	if !ok || m.UpdatedAt.IsZero() {
		return false
	}
	return m.UpdatedAt.Before(latestUpdatedAt)
}

// bitbucketPullRequestUpdatedAt reads pullrequest.updated_on from a stored
// event envelope.
func bitbucketPullRequestUpdatedAt(stored any) (time.Time, bool) {
	envelope, _ := stored.(map[string]any)
	payload, _ := envelope["data"].(map[string]any)
	return parseBitbucketPullRequestUpdatedAt(payload)
}

func parseBitbucketPullRequestUpdatedAt(payload map[string]any) (time.Time, bool) {
	pullRequest, _ := payload["pullrequest"].(map[string]any)
	value, _ := pullRequest["updated_on"].(string)
	if strings.TrimSpace(value) == "" {
		return time.Time{}, false
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, false
	}
	return updatedAt, true
}

// Bitbucket emits PR updates for description edits and can deliver webhooks
// late. Skip an automatic merge confidence start when the newest start for the
// PR has the same head and destination branch, or when the newest start
// reflects a later PR update. A retargeted PR gets a new assessment, also when
// it returns to an earlier destination. Explicit reruns use a different
// payload type and remain available.
func mergeConfidenceWebhookDedupKey(node *models.CanvasNode, payloadType string, payload any) mergeConfidenceWebhookIdentity {
	if node == nil || node.Type != models.NodeTypeTrigger || payloadType != "bitbucket.pullRequest" {
		return mergeConfidenceWebhookIdentity{}
	}
	template, _ := node.Metadata.Data()[models.FactoryAppTemplateMetadataKey].(map[string]any)
	if template["id"] != "risk-score" {
		return mergeConfidenceWebhookIdentity{}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return mergeConfidenceWebhookIdentity{}
	}
	var identity struct {
		PullRequest struct {
			ID string `json:"id"`
		} `json:"pullRequest"`
		View struct {
			Head struct {
				SHA string `json:"sha"`
			} `json:"head"`
			Base struct {
				Ref string `json:"ref"`
			} `json:"base"`
		} `json:"pull_request"`
		Raw map[string]any `json:"pullrequest"`
	}
	if err := json.Unmarshal(raw, &identity); err != nil {
		return mergeConfidenceWebhookIdentity{}
	}
	if identity.PullRequest.ID == "" || identity.View.Head.SHA == "" {
		return mergeConfidenceWebhookIdentity{}
	}
	// Git ref names cannot contain ":", so the key parts stay unambiguous.
	scope := "merge-confidence:" + identity.PullRequest.ID + ":"
	updatedAt, _ := parseBitbucketPullRequestUpdatedAt(map[string]any{"pullrequest": identity.Raw})
	return mergeConfidenceWebhookIdentity{
		Scope:     scope,
		Key:       scope + identity.View.Head.SHA + ":" + identity.View.Base.Ref,
		UpdatedAt: updatedAt,
	}
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
