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

// CreateGitHubChecksEvent serializes startup and webhook delivery for the same
// handler revision. Routed events remain part of the identity check.
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
	var payload struct {
		Data struct {
			Repository struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
			PullRequest struct {
				Number int64 `json:"number"`
				Head   struct {
					SHA string `json:"sha"`
				} `json:"head"`
			} `json:"pull_request"`
		} `json:"data"`
	}
	raw, err := event.Data.MarshalJSON()
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false, fmt.Errorf("decode checks startup identity: %w", err)
	}
	repository := strings.ToLower(strings.TrimSpace(payload.Data.Repository.FullName))
	number := strconv.FormatInt(payload.Data.PullRequest.Number, 10)
	sha := strings.TrimSpace(payload.Data.PullRequest.Head.SHA)
	if repository == "" || payload.Data.PullRequest.Number <= 0 || sha == "" {
		return false, errors.New("GitHub checks event has no repository, PR number, or head SHA")
	}
	created := false
	err = tx.Transaction(func(inner *gorm.DB) error {
		lockKey := "pr-checks:" + n.WorkflowID.String() + ":" + n.NodeID + ":" + repository + ":" + number + ":" + sha
		if err := inner.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}
		var count int64
		if err := inner.Model(&CanvasEvent{}).
			Where("workflow_id = ? AND node_id = ?", n.WorkflowID, n.NodeID).
			Where("data->>'type' = ?", "github.pullRequest").
			Where("lower(data #>> '{data,repository,full_name}') = ?", repository).
			Where("data #>> '{data,pull_request,number}' = ?", number).
			Where("data #>> '{data,pull_request,head,sha}' = ?", sha).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
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
