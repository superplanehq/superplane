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

// GitHubChecksTrigger resolves registered checks triggers from the live graph,
// including graphs whose trigger IDs have been renamed.
func (h *FactoryPRFeedbackHandler) GitHubChecksTrigger(tx *gorm.DB, repository string) (*CanvasNode, error) {
	if h.Source != FactoryPRFeedbackHandlerSourcePullRequestChecks || h.Subject != FactoryPRFeedbackHandlerSubjectGitHubPullRequest {
		return nil, nil
	}
	if _, err := FindOrganizationByIDInTransaction(tx, h.OrganizationID.String()); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	canvas, err := FindCanvasInTransaction(tx, h.OrganizationID, h.CanvasID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if canvas.FactoryID == nil || *canvas.FactoryID != h.FactoryID {
		return nil, nil
	}
	version, err := FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, candidate := range version.Nodes {
		if candidate.Type != NodeTypeTrigger || candidate.ComponentName() != "github.onPullRequest" {
			continue
		}
		configured, _ := candidate.Configuration["repository"].(string)
		if !strings.EqualFold(strings.TrimSpace(configured), repository) {
			continue
		}
		node, err := FindCanvasNode(tx, h.CanvasID, candidate.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if node.State == CanvasNodeStateError || node.Type != NodeTypeTrigger || node.ComponentName() != "github.onPullRequest" {
			return nil, nil
		}
		return node, nil
	}
	return nil, nil
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

// CreateGitHubChecksEvent serializes startup and webhook delivery for one
// handler revision. The same delivery stays a duplicate after the run finishes.
// A later reopen, ready-for-review, or new push starts again only when its
// pull request time is newer than the stored snapshot.
func (n *CanvasNode) CreateGitHubChecksEvent(tx *gorm.DB, event *CanvasEvent) (bool, error) {
	handler, err := FindPRFeedbackHandlerByCanvasID(tx, n.WorkflowID)
	if err != nil {
		return false, err
	}
	if handler == nil || handler.Source != FactoryPRFeedbackHandlerSourcePullRequestChecks || handler.Subject != FactoryPRFeedbackHandlerSubjectGitHubPullRequest {
		return true, tx.Create(event).Error
	}
	configuredRepository, _ := n.Configuration.Data()["repository"].(string)
	trigger, err := handler.GitHubChecksTrigger(tx, configuredRepository)
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
	factory := &Factory{ID: p.FactoryID, OrganizationID: p.OrganizationID}
	handlers, err := factory.ListPRFeedbackHandlers(tx)
	if err != nil {
		return err
	}
	for i := range handlers {
		node, err := handlers[i].GitHubChecksTrigger(tx, p.Repository)
		if err != nil {
			return err
		}
		if node == nil {
			continue
		}
		now := time.Now()
		request := CanvasNodeRequest{
			ID: uuid.New(), WorkflowID: node.WorkflowID, NodeID: node.NodeID,
			Type: NodeRequestTypePullRequestChecksStart, State: NodeExecutionRequestStatePending,
			Spec: datatypes.NewJSONType(NodeExecutionRequestSpec{PullRequestChecksStart: &PullRequestChecksStartRequest{
				OrganizationID: p.OrganizationID.String(), FactoryID: p.FactoryID.String(), PullRequestID: p.ID.String(),
			}}),
			RunAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&request).Error; err != nil {
			return err
		}
	}
	return nil
}

type checksEventIdentity struct {
	repository   string
	number       string
	sha          string
	action       string
	before       string
	updatedAt    time.Time
	hasUpdatedAt bool
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
		repository:   repository,
		number:       strconv.FormatInt(payload.Data.PullRequest.Number, 10),
		sha:          sha,
		action:       strings.TrimSpace(payload.Data.Action),
		before:       strings.TrimSpace(payload.Data.Before),
		updatedAt:    updatedAt,
		hasUpdatedAt: hasUpdatedAt,
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
