package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MergeConfidenceCheckRequest struct {
	RunID         uuid.UUID `json:"run_id"`
	PullRequestID uuid.UUID `json:"pull_request_id"`
}

// QueueMergeConfidenceCheck keeps publication independent of an execution, so
// cancellation still lets the worker finish the GitHub check.
func (r *CanvasRun) QueueMergeConfidenceCheck(tx *gorm.DB, pr *FactoryPullRequest) error {
	if pr.Provider != FactoryPullRequestProviderGitHub {
		return nil
	}
	now := time.Now()
	request := &CanvasNodeRequest{
		ID:         uuid.NewSHA1(r.ID, []byte(NodeRequestTypeMergeConfidenceCheck)),
		WorkflowID: r.WorkflowID,
		NodeID:     r.NodeID,
		State:      NodeExecutionRequestStatePending,
		Type:       NodeRequestTypeMergeConfidenceCheck,
		Spec: datatypes.NewJSONType(NodeExecutionRequestSpec{
			MergeConfidenceCheck: &MergeConfidenceCheckRequest{RunID: r.ID, PullRequestID: pr.ID},
		}),
		RunAt:     now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoNothing: true}).Create(request).Error
}
