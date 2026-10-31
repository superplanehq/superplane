package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const SubscriptionFailureChatGPTQuotaExhausted = "chatgpt_quota_exhausted"

// SubscriptionFailure contains recovery information safe to show to a member.
// SubscriptionOwnerID is assigned by the server from the actual task lease.
type SubscriptionFailure struct {
	Reason              string     `json:"reason"`
	LimitID             string     `json:"limit_id,omitempty"`
	ResetsAt            *time.Time `json:"resets_at,omitempty"`
	SubscriptionOwnerID string     `json:"subscription_owner_id,omitempty"`
}

func SubscriptionFailureFromResult(result []byte) *SubscriptionFailure {
	var document struct {
		Failure *SubscriptionFailure `json:"failure"`
	}
	if json.Unmarshal(result, &document) != nil || document.Failure == nil || document.Failure.Reason != SubscriptionFailureChatGPTQuotaExhausted {
		return nil
	}
	failure := document.Failure
	if id, err := uuid.Parse(failure.SubscriptionOwnerID); err != nil || id == uuid.Nil {
		return nil
	}
	if len(failure.LimitID) > 200 {
		failure.LimitID = ""
	}
	failure.LimitID = strings.TrimSpace(failure.LimitID)
	return failure
}

type RunnerResultSummary struct {
	SubscriptionFailure *SubscriptionFailure
	UsageState          string
}

// ListRunnerResultSummariesForRuns uses durable task results, so cleanup of a
// subscription connection or lease cannot change the reported owner or reason.
// Each requested run includes descendants, even when a descendant is also a
// requested root. This matches the execution summary's run-tree usage scope.
func ListRunnerResultSummariesForRuns(tx *gorm.DB, runIDs []uuid.UUID) (map[uuid.UUID]RunnerResultSummary, error) {
	summaries := make(map[uuid.UUID]RunnerResultSummary)
	if len(runIDs) == 0 {
		return summaries, nil
	}
	_, treeIDs, err := collectRunTreeIDs(tx, runIDs)
	if err != nil {
		return nil, err
	}
	direct, err := listDirectRunnerResultSummaries(tx, treeIDs)
	if err != nil {
		return nil, err
	}
	var runs []CanvasRun
	if err := tx.Select("id", "parent_run_id").Where("id IN ?", treeIDs).Find(&runs).Error; err != nil {
		return nil, err
	}
	parents := make(map[uuid.UUID]*uuid.UUID, len(runs))
	for _, run := range runs {
		parents[run.ID] = run.ParentRunID
	}
	requested := make(map[uuid.UUID]bool, len(runIDs))
	for _, runID := range runIDs {
		requested[runID] = true
	}
	for runID, summary := range direct {
		seen := map[uuid.UUID]bool{}
		for id := runID; !seen[id]; {
			seen[id] = true
			if requested[id] {
				summaries[id] = mergeRunnerResultSummary(summaries[id], summary)
			}
			parent := parents[id]
			if parent == nil {
				break
			}
			id = *parent
		}
	}
	return summaries, nil
}

func listDirectRunnerResultSummaries(tx *gorm.DB, runIDs []uuid.UUID) (map[uuid.UUID]RunnerResultSummary, error) {
	summaries := make(map[uuid.UUID]RunnerResultSummary)
	type record struct {
		RunID  uuid.UUID
		Result datatypes.JSON
	}
	var records []record
	err := tx.Table("workflow_node_executions AS execution").
		Select(`DISTINCT ON (execution.id) execution.run_id,
			jsonb_build_object('failure', task.result->'failure', 'usage_state', task.result->'usage_state') AS result`).
		Joins("JOIN workflow_node_execution_kvs AS kv ON kv.execution_id = execution.id AND kv.workflow_id = execution.workflow_id AND kv.node_id = execution.node_id AND kv.key = 'task_id'").
		Joins("JOIN workflows AS canvas ON canvas.id = execution.workflow_id").
		Joins("JOIN runner_tasks AS task ON task.id::text = kv.value AND task.organization_id = canvas.organization_id AND task.backend = ?", RunnerTaskBackendIntegrated).
		Where("execution.run_id IN ?", runIDs).
		Order("execution.id, kv.created_at DESC, kv.id DESC, task.created_at DESC, task.id DESC").Scan(&records).Error
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		summary := RunnerResultSummary{SubscriptionFailure: SubscriptionFailureFromResult(record.Result)}
		var result struct {
			UsageState string `json:"usage_state"`
		}
		if json.Unmarshal(record.Result, &result) == nil {
			summary.UsageState = result.UsageState
		}
		summaries[record.RunID] = mergeRunnerResultSummary(summaries[record.RunID], summary)
	}
	return summaries, nil
}

func mergeRunnerResultSummary(current, next RunnerResultSummary) RunnerResultSummary {
	if next.SubscriptionFailure != nil {
		current.SubscriptionFailure = next.SubscriptionFailure
	}
	current.UsageState = mergeUsageMeasurementState(current.UsageState, next.UsageState)
	return current
}

func mergeUsageMeasurementState(current, next string) string {
	switch next {
	case "pending", "measured", "partial", "unavailable":
	default:
		return current
	}
	if current == "" || current == next {
		return next
	}
	if current == "partial" || next == "partial" {
		return "partial"
	}
	if current == "measured" || next == "measured" {
		return "partial"
	}
	if current == "unavailable" || next == "unavailable" {
		return "unavailable"
	}
	return "pending"
}
