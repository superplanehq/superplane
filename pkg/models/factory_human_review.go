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
	PublicationPending   bool
	PublicationError     string
	RetryAt              *time.Time
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
		Updates(map[string]any{"state": FactoryHumanReviewStateSuperseded, "publication_pending": false, "updated_at": time.Now()}).Error; err != nil {
		return nil, err
	}
	decision.ID = uuid.New()
	decision.OrganizationID = p.OrganizationID
	decision.FactoryID = p.FactoryID
	decision.PullRequestID = p.ID
	decision.CreatedAt = time.Now()
	decision.UpdatedAt = decision.CreatedAt
	decision.PublicationPending = true
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

func ListOpenFactoryHumanReviewPullRequests(tx *gorm.DB) ([]FactoryPullRequest, error) {
	var prs []FactoryPullRequest
	err := tx.Where("provider = ? AND state IN ?", FactoryPullRequestProviderGitHub, []string{FactoryPullRequestStateOpen, FactoryPullRequestStateDraft}).
		Order("updated_at ASC, id ASC").Find(&prs).Error
	return prs, err
}

// LatestMergeConfidenceAssessment includes an unfinished newer run. A rerun
// immediately retires the old assessment, even when the head SHA is unchanged.
func (p *FactoryPullRequest) LatestMergeConfidenceAssessment(tx *gorm.DB, headSHA string) (*CanvasRun, datatypes.JSONMap, error) {
	runs, err := p.ListActivityRuns(tx)
	if err != nil {
		return nil, nil, err
	}
	for i := range runs {
		run := &runs[i]
		version, err := FindCanvasVersionInTransaction(tx, run.WorkflowID, run.VersionID)
		if err != nil {
			return nil, nil, err
		}
		if FactoryAppTemplateID(version.Nodes) != "risk-score" {
			continue
		}
		root, err := run.RootEvent(tx)
		if err != nil {
			return nil, nil, err
		}
		payload, ok := objectMap(RootEventSourcePayload(root.Data.Data()))
		if !ok {
			continue
		}
		pr, _ := objectMap(payload["pull_request"])
		head, _ := objectMap(pr["head"])
		if head["sha"] != headSHA {
			continue
		}
		report, err := run.MergeConfidenceAssessment(tx)
		if err != nil {
			return nil, nil, err
		}
		return run, report, nil
	}
	return nil, nil, nil
}

// MergeConfidenceAssessment reads only the server-owned assessment node from
// this run's frozen version and validates the execution that emitted it.
func (r *CanvasRun) MergeConfidenceAssessment(tx *gorm.DB) (datatypes.JSONMap, error) {
	version, err := FindCanvasVersionInTransaction(tx, r.WorkflowID, r.VersionID)
	if err != nil {
		return nil, err
	}
	if FactoryAppTemplateID(version.Nodes) != "risk-score" {
		return nil, nil
	}
	assessmentNode := mergeConfidenceAssessmentNode(version.Nodes)
	if assessmentNode == nil {
		return nil, nil
	}
	var events []CanvasEvent
	if err := tx.Where("run_id = ? AND execution_id IS NOT NULL", r.ID).
		Order("created_at ASC, id ASC").Find(&events).Error; err != nil {
		return nil, err
	}
	for _, event := range events {
		if event.NodeID != assessmentNode.ID || (event.Channel != "passed" && event.Channel != "failed") {
			continue
		}
		execution, err := FindNodeExecutionInTransaction(tx, r.WorkflowID, *event.ExecutionID)
		if err != nil {
			return nil, err
		}
		if execution.RunID != r.ID || execution.NodeID != assessmentNode.ID || (execution.FrozenComponentName() != "" && execution.FrozenComponentName() != assessmentNode.ComponentName()) {
			continue
		}
		envelope, ok := objectMap(event.Data.Data())
		if !ok {
			continue
		}
		if envelope["type"] != assessmentNode.ComponentName()+".finished" {
			continue
		}
		body, _ := objectMap(envelope["data"])
		if report, ok := objectMap(body["mergeConfidence"]); ok {
			return datatypes.JSONMap(report), nil
		}
	}
	return nil, nil
}

func mergeConfidenceAssessmentNode(nodes []Node) *Node {
	for i := range nodes {
		node := &nodes[i]
		if node.ID != "assess-risk" || !IsAgentRunnerComponent(node.ComponentName()) {
			continue
		}
		encoded, err := json.Marshal(node.Configuration)
		if err != nil {
			continue
		}
		var configuration struct {
			Environment []struct {
				Name string `json:"name"`
			} `json:"environment"`
		}
		if json.Unmarshal(encoded, &configuration) != nil {
			continue
		}
		for _, variable := range configuration.Environment {
			// This is the server-owned assessment scope, not an arbitrary agent result.
			if variable.Name == "SUPERPLANE_MERGE_CONFIDENCE_ORDER_ID" {
				return node
			}
		}
	}
	return nil
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
