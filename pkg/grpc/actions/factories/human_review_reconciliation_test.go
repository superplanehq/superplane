package factories

import (
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func (f humanReviewReconciliationFixture) reconcile(t *testing.T) {
	t.Helper()
	require.NoError(t, reconcileHumanReview(t.Context(), f.db, IntakeDependencies{}, f.factory, f.pr, f.automation, f.client))
}

func (f humanReviewReconciliationFixture) complete(t *testing.T, decision *models.FactoryHumanReviewDecision) {
	t.Helper()
	root, err := models.FindCanvasRunInTransaction(f.db, f.automation.canvas.ID, *decision.RunID)
	require.NoError(t, err)
	event, err := root.RootEvent(f.db)
	require.NoError(t, err)
	execution := support.CreateCanvasNodeExecution(t, f.automation.canvas.ID, models.HumanReviewAgentNodeID, event.ID, event.ID)
	require.NoError(t, f.db.Model(execution).Update("state", models.CanvasNodeExecutionStateStarted).Error)
	execution.State = models.CanvasNodeExecutionStateStarted
	require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
		payload := map[string]any{"result": map[string]any{"humanReview": map[string]any{"assessedCommit": "current-head", "reviewRequired": true, "rationale": "The security owner must approve.", "selectedReviewers": []any{map[string]any{"id": 8, "login": "owner", "reason": "Security code owner."}}}}}
		if err := contexts.NewExecutionStateContext(tx, execution, nil).Emit("passed", "runnerClaudeCode.finished", []any{payload}); err != nil {
			return err
		}
		return tx.Model(root).Updates(map[string]any{"state": models.CanvasRunStateFinished, "result": models.CanvasRunResultPassed}).Error
	}))
}

func TestHumanReviewReconciliationPreservesConcurrentCompletion(t *testing.T) {
	f := newHumanReviewReconciliationFixture(t)
	f.reconcile(t)
	decision, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.Empty(t, decision.Result)
	owner := reviewTestUser(8, "owner", "User")
	f.client.permissions = map[string]*github.RepositoryPermissionLevel{"owner": {User: owner, Permission: github.Ptr("write")}}

	// Complete through the real runner boundary after reconciliation reads the
	// empty decision, but before it observes the finished agent run.
	completed := false
	callback := "complete-human-review:" + decision.ID.String()
	require.NoError(t, f.db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if completed || tx.Statement.Table != "workflow_runs" {
			return
		}
		completed = true
		f.complete(t, decision)
	}))
	t.Cleanup(func() { f.db.Callback().Query().Remove(callback) })
	f.reconcile(t)
	require.True(t, completed)
	stored, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.Equal(t, decision.ID, stored.ID)
	require.Equal(t, models.FactoryHumanReviewStateDecided, stored.State)
	require.NotEmpty(t, stored.Result)
	require.Equal(t, [][]string{{"owner"}}, f.client.requests)
}

func TestHumanReviewReconciliationPreservesGateAfterRunRetention(t *testing.T) {
	f := newHumanReviewReconciliationFixture(t)
	f.reconcile(t)
	decision, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	f.complete(t, decision)
	owner := reviewTestUser(8, "owner", "User")
	f.client.permissions = map[string]*github.RepositoryPermissionLevel{"owner": {User: owner, Permission: github.Ptr("write")}}
	f.client.reviews = []*github.PullRequestReview{{ID: github.Ptr(int64(99)), User: owner, State: github.Ptr("APPROVED"), CommitID: github.Ptr("current-head")}}
	f.reconcile(t)

	agent, err := models.FindCanvasRunInTransaction(f.db, f.automation.canvas.ID, *decision.RunID)
	require.NoError(t, err)
	for _, run := range []*models.CanvasRun{f.source, agent} {
		require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
			_, err := run.DeleteChain(tx)
			return err
		}))
		f.reconcile(t)
		stored, err := f.pr.CurrentHumanReview(f.db)
		require.NoError(t, err)
		require.Equal(t, decision.ID, stored.ID)
		require.Equal(t, models.FactoryHumanReviewStateDecided, stored.State)
		require.NotEmpty(t, stored.Assessment)
		require.NotEmpty(t, stored.Result)
		require.Equal(t, []int64{99}, []int64(stored.ApprovalIDs))
		require.Equal(t, humanReviewPublication{Status: "completed", Conclusion: "success"}, f.client.publications[len(f.client.publications)-1])
		requests, err := models.ListPullRequestChecksStartRequests(f.db, f.pr.ID)
		require.NoError(t, err)
		require.Empty(t, requests)
	}

	// Retained evidence still follows current GitHub reviews.
	f.client.reviews[0].State = github.Ptr("CHANGES_REQUESTED")
	f.reconcile(t)
	require.Equal(t, humanReviewPublication{Status: "completed", Conclusion: "failure"}, f.client.publications[len(f.client.publications)-1])
}
