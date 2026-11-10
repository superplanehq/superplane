package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	FactoryHumanReviewStateWaitingAssessment = "waiting_assessment"
	FactoryHumanReviewStateWaitingDecision   = "waiting_decision"
	FactoryHumanReviewStateDecided           = "decided"
	FactoryHumanReviewStateFailed            = "failed"
	FactoryHumanReviewStateDisabled          = "disabled"
	FactoryHumanReviewStateSuperseded        = "superseded"
)

type FactoryHumanReviewReviewer struct {
	ID     int64  `json:"id"`
	Login  string `json:"login"`
	Reason string `json:"reason"`
}

// Each policy evaluation owns its immutable assessment and agent result. The
// current decision and all external publication are serialized by the PR lock.
type FactoryHumanReviewDecision struct {
	ID                   uuid.UUID
	OrganizationID       uuid.UUID
	FactoryID            uuid.UUID
	PullRequestID        uuid.UUID
	RevisionID           uuid.UUID
	SourceRunID          *uuid.UUID
	RunID                *uuid.UUID
	ConfigurationVersion string
	State                string
	AssessedCommit       string
	Assessment           datatypes.JSONMap
	Result               datatypes.JSONMap
	ReviewRequired       bool
	Rationale            string
	SelectedReviewers    datatypes.JSONSlice[FactoryHumanReviewReviewer]
	ApprovalIDs          datatypes.JSONSlice[int64]
	BlockingReason       string
	CheckRunID           int64
	CheckURL             string
	PublicationError     string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (FactoryHumanReviewDecision) TableName() string { return "factory_human_review_decisions" }

// NewHumanReviewDecision requires the caller to hold the PR lock. Superseding
// and inserting are part of that same transaction, so late results lose authority.
func (p *FactoryPullRequest) NewHumanReviewDecision(tx *gorm.DB, decision FactoryHumanReviewDecision) (*FactoryHumanReviewDecision, error) {
	if p.CurrentRevisionID == nil || decision.RevisionID != *p.CurrentRevisionID {
		return nil, fmt.Errorf("human review revision is not current")
	}
	if err := tx.Model(&FactoryHumanReviewDecision{}).
		Where("pull_request_id = ? AND state <> ?", p.ID, FactoryHumanReviewStateSuperseded).
		Updates(map[string]any{"state": FactoryHumanReviewStateSuperseded, "updated_at": time.Now()}).Error; err != nil {
		return nil, err
	}
	decision.ID = uuid.New()
	decision.OrganizationID = p.OrganizationID
	decision.FactoryID = p.FactoryID
	decision.PullRequestID = p.ID
	decision.CreatedAt = time.Now()
	decision.UpdatedAt = decision.CreatedAt
	if decision.Assessment == nil {
		decision.Assessment = datatypes.JSONMap{}
	}
	if decision.Result == nil {
		decision.Result = datatypes.JSONMap{}
	}
	if decision.SelectedReviewers == nil {
		decision.SelectedReviewers = datatypes.NewJSONSlice([]FactoryHumanReviewReviewer{})
	}
	if decision.ApprovalIDs == nil {
		decision.ApprovalIDs = datatypes.NewJSONSlice([]int64{})
	}
	if err := tx.Create(&decision).Error; err != nil {
		return nil, err
	}
	return &decision, nil
}

func FindFactoryHumanReviewDecision(tx *gorm.DB, id uuid.UUID) (*FactoryHumanReviewDecision, error) {
	var decision FactoryHumanReviewDecision
	if err := tx.Where("id = ?", id).First(&decision).Error; err != nil {
		return nil, err
	}
	return &decision, nil
}

func (p *FactoryPullRequest) CurrentHumanReview(tx *gorm.DB) (*FactoryHumanReviewDecision, error) {
	var decision FactoryHumanReviewDecision
	err := tx.Where("pull_request_id = ? AND state <> ?", p.ID, FactoryHumanReviewStateSuperseded).
		Order("created_at DESC, id DESC").First(&decision).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &decision, nil
}

func (d *FactoryHumanReviewDecision) Save(tx *gorm.DB) error {
	d.UpdatedAt = time.Now()
	return tx.Save(d).Error
}

func ListFactoryHumanReviewPullRequestsToReconcile(tx *gorm.DB, afterID uuid.UUID, limit int) ([]FactoryPullRequest, error) {
	// Disabled policies still publish an explanatory success for required gates.
	// Only a published Human review installation makes a workspace relevant.
	marker, err := json.Marshal([]map[string]any{{
		"id": HumanReviewTriggerNodeID, "type": NodeTypeTrigger,
		"metadata": map[string]any{FactoryAppTemplateMetadataKey: map[string]any{"id": FactoryAppTemplateHumanReviewID}},
	}})
	if err != nil {
		return nil, err
	}
	query := tx.Where("provider = ? AND (state IN ? OR human_review_request_id IS NOT NULL)", FactoryPullRequestProviderGitHub, []string{FactoryPullRequestStateOpen, FactoryPullRequestStateDraft}).
		Where(`EXISTS (
			SELECT 1 FROM factories
			JOIN organizations ON organizations.id = factories.organization_id AND organizations.deleted_at IS NULL
			JOIN workflows ON workflows.factory_id = factories.id AND workflows.organization_id = factories.organization_id AND workflows.deleted_at IS NULL
			JOIN workflow_versions live_version ON live_version.id = workflows.live_version_id AND live_version.workflow_id = workflows.id
			WHERE factories.id = factory_pull_requests.factory_id
			  AND factories.organization_id = factory_pull_requests.organization_id
			  AND factories.deleted_at IS NULL AND live_version.nodes @> ?::jsonb
		)`, string(marker))
	if afterID != uuid.Nil {
		query = query.Where("factory_pull_requests.id > ?", afterID)
	}
	var prs []FactoryPullRequest
	err = query.Order("factory_pull_requests.id ASC").Limit(limit).Find(&prs).Error
	return prs, err
}

func (p *FactoryPullRequest) RequestVerificationChecks(tx *gorm.DB) error {
	if p.Provider != FactoryPullRequestProviderGitHub || p.State != FactoryPullRequestStateOpen {
		return nil
	}
	workspaceRepository, err := isFactoryWorkspaceRepository(tx, p.OrganizationID, p.FactoryID, p.Repository)
	if err != nil {
		return err
	}
	nodes, err := verificationTriggers(tx.Where(
		"workflows.organization_id = ? AND workflows.factory_id = ?", p.OrganizationID, p.FactoryID,
	).Where(`EXISTS (SELECT 1 FROM workflow_nodes merge_node WHERE merge_node.workflow_id = workflows.id AND merge_node.deleted_at IS NULL AND merge_node.metadata->'factoryTemplate'->>'id' = 'risk-score')`), p.Provider, p.Repository, workspaceRepository)
	if err != nil {
		return err
	}
	return p.enqueueVerificationTriggers(tx, nodes)
}

func ListCurrentFactoryHumanReviewDecisions(tx *gorm.DB, ids []uuid.UUID) (map[uuid.UUID]FactoryHumanReviewDecision, error) {
	result := map[uuid.UUID]FactoryHumanReviewDecision{}
	if len(ids) == 0 {
		return result, nil
	}
	var decisions []FactoryHumanReviewDecision
	if err := tx.Where("pull_request_id IN ? AND state <> ?", ids, FactoryHumanReviewStateSuperseded).Find(&decisions).Error; err != nil {
		return nil, err
	}
	for _, decision := range decisions {
		result[decision.PullRequestID] = decision
	}
	return result, nil
}

func FindFactoryHumanReviewDecisionByRun(tx *gorm.DB, runID uuid.UUID) (*FactoryHumanReviewDecision, error) {
	var decision FactoryHumanReviewDecision
	err := tx.Where("run_id = ?", runID).First(&decision).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &decision, nil
}

func (p *FactoryPullRequest) queueHumanReviewRevision(tx *gorm.DB, revision *FactoryPullRequestRevision) error {
	previous, err := p.CurrentHumanReview(tx)
	if err != nil || previous == nil || previous.State == FactoryHumanReviewStateDisabled {
		return err
	}
	if previous.RevisionID == revision.ID && previous.AssessedCommit == revision.SHA {
		return nil
	}
	_, err = p.NewHumanReviewDecision(tx, FactoryHumanReviewDecision{
		RevisionID:           revision.ID,
		AssessedCommit:       revision.SHA,
		ConfigurationVersion: previous.ConfigurationVersion,
		State:                FactoryHumanReviewStateWaitingAssessment,
		BlockingReason:       "Wait for merge confidence to assess the current head commit.",
	})
	return err
}
