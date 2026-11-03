package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GitHubVerificationTrigger identifies PR checks and merge confidence from
// the live graph. Attachment and delivery use the same eligibility rules.
func (c *Canvas) GitHubVerificationTrigger(tx *gorm.DB, repository string) (*CanvasNode, error) {
	if c.FactoryID == nil || c.DeletedAt.Valid {
		return nil, nil
	}
	nodes, err := githubVerificationTriggers(tx.Where(
		"workflows.organization_id = ? AND workflows.factory_id = ? AND workflows.id = ?",
		c.OrganizationID, *c.FactoryID, c.ID,
	), repository)
	if err != nil || len(nodes) == 0 {
		return nil, err
	}
	return &nodes[0], nil
}

type githubVerificationTriggerRow struct {
	CanvasNode    `gorm:"embedded"`
	LiveNodes     datatypes.JSONSlice[Node]
	ChecksHandler bool
}

// Load candidate triggers and their live graphs together. The caller scopes
// workflows to the organization and factory, optionally to one canvas.
func githubVerificationTriggers(tx *gorm.DB, repository string) ([]CanvasNode, error) {
	var rows []githubVerificationTriggerRow
	err := tx.Model(&CanvasNode{}).
		Select("workflow_nodes.*", "live_version.nodes AS live_nodes", "handler.id IS NOT NULL AS checks_handler").
		Joins("JOIN workflows ON workflows.id = workflow_nodes.workflow_id AND workflows.deleted_at IS NULL").
		Joins("JOIN organizations ON organizations.id = workflows.organization_id AND organizations.deleted_at IS NULL").
		Joins("JOIN workflow_versions live_version ON live_version.id = workflows.live_version_id AND live_version.workflow_id = workflows.id").
		Joins("LEFT JOIN factory_pr_feedback_handlers handler ON handler.canvas_id = workflows.id").
		Where("workflow_nodes.type = ? AND workflow_nodes.state <> ?", NodeTypeTrigger, CanvasNodeStateError).
		Where(`(
			(handler.organization_id = workflows.organization_id AND handler.factory_id = workflows.factory_id
			 AND handler.subject = ? AND handler.source = ?)
			OR (handler.id IS NULL AND EXISTS (
				SELECT 1 FROM workflow_nodes template_node
				WHERE template_node.workflow_id = workflows.id AND template_node.deleted_at IS NULL
				  AND template_node.metadata -> 'factoryTemplate' ->> 'id' = ?
			))
		)`, FactoryPRFeedbackHandlerSubjectGitHubPullRequest, FactoryPRFeedbackHandlerSourcePullRequestChecks, "risk-score").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	nodes := make([]CanvasNode, 0, len(rows))
	for _, row := range rows {
		if row.ComponentName() != "github.onPullRequest" {
			continue
		}
		if !row.ChecksHandler && FactoryAppTemplateID(row.LiveNodes) != "risk-score" {
			continue
		}
		if liveGitHubPullRequestTriggerID(row.LiveNodes, repository) == row.NodeID {
			nodes = append(nodes, row.CanvasNode)
		}
	}
	return nodes, nil
}

func liveGitHubPullRequestTriggerID(nodes []Node, repository string) string {
	for _, candidate := range nodes {
		if candidate.Type != NodeTypeTrigger || candidate.ComponentName() != "github.onPullRequest" {
			continue
		}
		configured, _ := candidate.Configuration["repository"].(string)
		if strings.EqualFold(strings.TrimSpace(configured), repository) {
			return candidate.ID
		}
	}
	return ""
}

// Include requests whose canvas was deleted so the processor can retire them.
func ListDuePullRequestChecksStartRequests(tx *gorm.DB) ([]CanvasNodeRequest, error) {
	var requests []CanvasNodeRequest
	err := tx.Where("type = ? AND state = ? AND run_at <= ?", NodeRequestTypePullRequestChecksStart, NodeExecutionRequestStatePending, time.Now()).Find(&requests).Error
	return requests, err
}

func ListPullRequestChecksStartRequests(tx *gorm.DB, pullRequestID uuid.UUID) ([]CanvasNodeRequest, error) {
	var requests []CanvasNodeRequest
	err := tx.Where("type = ? AND state = ?", NodeRequestTypePullRequestChecksStart, NodeExecutionRequestStatePending).
		Where("spec->'pull_request_checks_start'->>'pull_request_id' = ?", pullRequestID.String()).Find(&requests).Error
	return requests, err
}

func ClaimDuePullRequestChecksStartRequest(tx *gorm.DB, id uuid.UUID, leaseUntil time.Time) (*CanvasNodeRequest, error) {
	var request CanvasNodeRequest
	err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ? AND type = ? AND state = ? AND run_at <= ?", id, NodeRequestTypePullRequestChecksStart, NodeExecutionRequestStatePending, time.Now()).First(&request).Error
	if err != nil {
		return nil, err
	}
	if err := request.Reschedule(tx, leaseUntil); err != nil {
		return nil, err
	}
	return &request, nil
}

// CreateGitHubChecksEvent serializes startup and webhook delivery for PR
// checks and merge confidence. The same delivery stays a duplicate after the
// run finishes.
// A later reopen, ready-for-review, or new push starts again only when its
// pull request time is newer than the stored snapshot.
func (n *CanvasNode) CreateGitHubChecksEvent(tx *gorm.DB, event *CanvasEvent) (bool, error) {
	canvas, err := FindUnscopedCanvasInTransaction(tx, n.WorkflowID)
	if err != nil {
		return false, err
	}
	configuredRepository, _ := n.Configuration.Data()["repository"].(string)
	trigger, err := canvas.GitHubVerificationTrigger(tx, configuredRepository)
	if err != nil {
		return false, err
	}
	if trigger == nil || trigger.NodeID != n.NodeID {
		return true, tx.Create(event).Error
	}
	incoming, err := checksEventIdentityFrom(event)
	if err != nil {
		return false, err
	}
	key := incoming.dedupKey()
	event.DedupKey = &key
	created := false
	err = tx.Transaction(func(inner *gorm.DB) error {
		lockKey := "pr-checks:" + n.WorkflowID.String() + ":" + n.NodeID + ":" + key
		if err := inner.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}
		if err := repairLegacyChecksDedupKeys(inner, n, key, incoming); err != nil {
			return err
		}
		var stored []CanvasEvent
		if err := inner.Select("data").
			Where("workflow_id = ? AND node_id = ? AND dedup_key = ?", n.WorkflowID, n.NodeID, key).
			Find(&stored).Error; err != nil {
			return err
		}
		known := make([]checksEventIdentity, 0, len(stored))
		for i := range stored {
			identity, err := checksEventIdentityFrom(&stored[i])
			if err != nil {
				return err
			}
			if incoming.pullRequestID != "" && identity.pullRequestID != "" && incoming.pullRequestID != identity.pullRequestID {
				continue
			}
			known = append(known, identity)
		}
		if checksEventIsDuplicate(known, incoming) {
			return nil
		}
		if err := inner.Create(event).Error; err != nil {
			return err
		}
		created = true
		return nil
	})
	return created, err
}

func (p *FactoryPullRequest) enqueueChecksStart(tx *gorm.DB) error {
	if p.Provider != FactoryPullRequestProviderGitHub || p.State != FactoryPullRequestStateOpen {
		return nil
	}
	nodes, err := githubVerificationTriggers(tx.Where(
		"workflows.organization_id = ? AND workflows.factory_id = ?", p.OrganizationID, p.FactoryID,
	), p.Repository)
	if err != nil || len(nodes) == 0 {
		return err
	}
	now := time.Now()
	requests := make([]CanvasNodeRequest, 0, len(nodes))
	for _, node := range nodes {
		requests = append(requests, CanvasNodeRequest{
			ID: uuid.New(), WorkflowID: node.WorkflowID, NodeID: node.NodeID,
			Type: NodeRequestTypePullRequestChecksStart, State: NodeExecutionRequestStatePending,
			Spec: datatypes.NewJSONType(NodeExecutionRequestSpec{PullRequestChecksStart: &PullRequestChecksStartRequest{
				OrganizationID: p.OrganizationID.String(), FactoryID: p.FactoryID.String(), PullRequestID: p.ID.String(),
			}}),
			RunAt: now, CreatedAt: now, UpdatedAt: now,
		})
	}
	return tx.Create(&requests).Error
}

type checksEventIdentity struct {
	pullRequestID string
	repository    string
	number        string
	sha           string
	action        string
	before        string
	updatedAt     time.Time
	hasUpdatedAt  bool
}

func (identity checksEventIdentity) dedupKey() string {
	return identity.repository + "|" + identity.number + "|" + identity.sha
}

// Events stored before dedup_key existed still match the previous JSON lookup.
// Copy the key onto those rows so the next delivery uses the indexed column.
func repairLegacyChecksDedupKeys(tx *gorm.DB, node *CanvasNode, key string, incoming checksEventIdentity) error {
	return tx.Exec(`
		UPDATE workflow_events
		SET dedup_key = ?
		WHERE workflow_id = ? AND node_id = ? AND dedup_key IS NULL
		  AND data ->> 'type' = 'github.pullRequest'
		  AND lower(data #>> '{data,repository,full_name}') = ?
		  AND data #>> '{data,pull_request,number}' = ?
		  AND data #>> '{data,pull_request,head,sha}' = ?
	`, key, node.WorkflowID, node.NodeID, incoming.repository, incoming.number, incoming.sha).Error
}

func checksEventIdentityFrom(event *CanvasEvent) (checksEventIdentity, error) {
	var payload struct {
		Data struct {
			Tracking struct {
				ID string `json:"id"`
			} `json:"pullRequest"`
			Action     string `json:"action"`
			Before     string `json:"before"`
			Repository struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
			PullRequest struct {
				Number    int64  `json:"number"`
				UpdatedAt string `json:"updated_at"`
				Head      struct {
					SHA string `json:"sha"`
				} `json:"head"`
			} `json:"pull_request"`
		} `json:"data"`
	}
	raw, err := event.Data.MarshalJSON()
	if err != nil {
		return checksEventIdentity{}, err
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return checksEventIdentity{}, fmt.Errorf("decode checks startup identity: %w", err)
	}
	repository := strings.ToLower(strings.TrimSpace(payload.Data.Repository.FullName))
	sha := strings.TrimSpace(payload.Data.PullRequest.Head.SHA)
	if repository == "" || payload.Data.PullRequest.Number <= 0 || sha == "" {
		return checksEventIdentity{}, errors.New("GitHub checks event has no repository, PR number, or head SHA")
	}
	updatedAt, hasUpdatedAt := parseChecksUpdatedAt(payload.Data.PullRequest.UpdatedAt)
	return checksEventIdentity{
		pullRequestID: payload.Data.Tracking.ID,
		repository:    repository,
		number:        strconv.FormatInt(payload.Data.PullRequest.Number, 10),
		sha:           sha,
		action:        strings.TrimSpace(payload.Data.Action),
		before:        strings.TrimSpace(payload.Data.Before),
		updatedAt:     updatedAt,
		hasUpdatedAt:  hasUpdatedAt,
	}, nil
}

func parseChecksUpdatedAt(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func checksEventIsDuplicate(stored []checksEventIdentity, incoming checksEventIdentity) bool {
	if len(stored) == 0 {
		return false
	}
	for _, event := range stored {
		if sameChecksDelivery(event, incoming) {
			return true
		}
	}
	if checksEventIsLaterLifecycle(stored, incoming) {
		return false
	}
	return true
}

func sameChecksDelivery(left, right checksEventIdentity) bool {
	if left.action != right.action || left.before != right.before || left.hasUpdatedAt != right.hasUpdatedAt {
		return false
	}
	if !left.hasUpdatedAt {
		return true
	}
	return left.updatedAt.Equal(right.updatedAt)
}

func checksEventIsLaterLifecycle(stored []checksEventIdentity, incoming checksEventIdentity) bool {
	switch incoming.action {
	case "reopened", "ready_for_review", "synchronize":
		if !incoming.hasUpdatedAt {
			return false
		}
		if incoming.action == "synchronize" {
			if incoming.before == "" {
				return false
			}
			for _, event := range stored {
				if event.action == "synchronize" && event.before == incoming.before {
					return false
				}
			}
		}
		newest, ok := newestChecksUpdatedAt(stored)
		if !ok {
			return false
		}
		return incoming.updatedAt.After(newest)
	default:
		return false
	}
}

func newestChecksUpdatedAt(stored []checksEventIdentity) (time.Time, bool) {
	var newest time.Time
	found := false
	for _, event := range stored {
		if !event.hasUpdatedAt {
			continue
		}
		if !found || event.updatedAt.After(newest) {
			newest = event.updatedAt
			found = true
		}
	}
	return newest, found
}
