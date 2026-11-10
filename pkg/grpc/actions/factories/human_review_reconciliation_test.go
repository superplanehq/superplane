package factories

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/integrations/prbody"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	supportcontexts "github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
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

func TestHumanReviewStartsAfterDraftBecomesReadyOnTheSameCommit(t *testing.T) {
	f := newHumanReviewReconciliationFixture(t)
	f.client.current.Draft = github.Ptr(true)
	f.reconcile(t)
	decision, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.Nil(t, decision, "draft pull requests must not start Human review")
	require.Empty(t, f.client.publications)

	f.client.current.Draft = github.Ptr(false)
	f.reconcile(t)
	decision, err = f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.NotNil(t, decision)
	require.NotNil(t, decision.RunID)
	require.Equal(t, "current-head", decision.AssessedCommit)
	require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
		run, err := models.FindCanvasRunInTransaction(tx, f.automation.canvas.ID, *decision.RunID)
		if err != nil {
			return err
		}
		root, err := run.RootEvent(tx)
		if err != nil {
			return err
		}
		return models.MaybeLinkHumanReviewRun(tx, f.automation.canvas, root, run)
	}))
	f.complete(t, decision)
	owner := reviewTestUser(8, "owner", "User")
	f.client.reviews = []*github.PullRequestReview{{ID: github.Ptr(int64(99)), User: owner, State: github.Ptr("APPROVED"), CommitID: github.Ptr("current-head")}}
	f.client.permissions = map[string]*github.RepositoryPermissionLevel{"owner": {User: owner, Permission: github.Ptr("write")}}
	f.reconcile(t)
	require.Equal(t, humanReviewPublication{Status: "completed", Conclusion: "success"}, f.client.publications[len(f.client.publications)-1])
}

// Retention can delete the newer-created assessment run while an older run for
// the same head remains. That older run must not reopen a completed gate.
func TestHumanReviewRetentionOfNewerAssessmentKeepsCompletedGate(t *testing.T) {
	f := newHumanReviewReconciliationFixture(t)
	older := f.addAssessmentRun(t)
	require.NoError(t, f.db.Model(&models.CanvasRun{}).Where("id = ?", older.ID).Update("created_at", f.source.CreatedAt.Add(-time.Minute)).Error)
	f.reconcile(t)
	decision, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.Equal(t, f.source.ID, *decision.SourceRunID)
	f.complete(t, decision)
	owner := reviewTestUser(8, "owner", "User")
	f.client.permissions = map[string]*github.RepositoryPermissionLevel{"owner": {User: owner, Permission: github.Ptr("write")}}
	f.client.reviews = []*github.PullRequestReview{{ID: github.Ptr(int64(99)), User: owner, State: github.Ptr("APPROVED"), CommitID: github.Ptr("current-head")}}
	f.reconcile(t)

	require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
		_, err := f.source.DeleteChain(tx)
		return err
	}))
	f.reconcile(t)
	stored, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.Equal(t, decision.ID, stored.ID)
	require.Equal(t, decision.RunID, stored.RunID)
	require.Equal(t, models.FactoryHumanReviewStateDecided, stored.State)
	require.Equal(t, humanReviewPublication{Status: "completed", Conclusion: "success"}, f.client.publications[len(f.client.publications)-1])
}

// Retention can delete the assessment run of a decision whose agent did not
// start. An older run for the same head must then become the new source.
func TestHumanReviewRetentionOfStartupFailureUsesOlderAssessment(t *testing.T) {
	f := newHumanReviewReconciliationFixture(t)
	older := f.addAssessmentRun(t)
	require.NoError(t, f.db.Model(&models.CanvasRun{}).Where("id = ?", older.ID).Update("created_at", f.source.CreatedAt.Add(-time.Minute)).Error)
	f.reconcile(t)
	failed, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.Equal(t, f.source.ID, *failed.SourceRunID)
	require.NoError(t, f.db.Model(failed).Updates(map[string]any{"run_id": nil, "state": models.FactoryHumanReviewStateFailed, "blocking_reason": "Request review could not start."}).Error)

	require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
		_, err := f.source.DeleteChain(tx)
		return err
	}))
	f.reconcile(t)
	stored, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.NotEqual(t, failed.ID, stored.ID)
	require.NotNil(t, stored.SourceRunID)
	require.Equal(t, older.ID, *stored.SourceRunID)
	require.NotNil(t, stored.RunID)
}

func (f humanReviewReconciliationFixture) addAssessmentRun(t *testing.T) *models.CanvasRun {
	t.Helper()
	root := &models.CanvasEvent{WorkflowID: f.source.WorkflowID, NodeID: "on-pr-risk", Data: models.NewJSONValue(map[string]any{"type": "github.pullRequest", "data": map[string]any{"pull_request": map[string]any{"head": map[string]any{"sha": "current-head"}}}})}
	require.NoError(t, f.db.Create(root).Error)
	require.NoError(t, f.pr.LinkRun(f.db, root.RunID, "Merge confidence"))
	execution := support.CreateCanvasNodeExecution(t, f.source.WorkflowID, "assess-risk", root.ID, root.ID)
	score := 4.0
	report := mergeconfidence.ReportData{
		PullRequestID: f.pr.ID.String(), Complete: true, ReviewedCommit: "current-head", OverallScore: &score, OverallLevel: "positive",
		Checks: []mergeconfidence.ReportCheck{{ID: "security", Score: 4, MaxScore: 5, Summary: "No security issue found.", Level: "positive"}},
	}
	require.NoError(t, f.db.Create(&models.CanvasEvent{
		WorkflowID: f.source.WorkflowID, NodeID: "assess-risk", ExecutionID: &execution.ID, Channel: "passed",
		Data: models.NewJSONValue(map[string]any{"type": "runnerClaudeCode.finished", "data": map[string]any{"mergeConfidence": report}}),
	}).Error)
	require.NoError(t, f.db.Model(&models.CanvasRun{}).Where("id = ?", root.RunID).Updates(map[string]any{"state": models.CanvasRunStateFinished, "result": models.CanvasRunResultPassed}).Error)
	run, err := models.FindCanvasRunInTransaction(f.db, f.source.WorkflowID, root.RunID)
	require.NoError(t, err)
	return run
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

func TestHumanReviewReconciliationPublishesGateAndRequestsMissingAssessment(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("human-review"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
	require.NoError(t, err)
	pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
	require.NoError(t, err)
	createMergeConfidenceStartupCanvas(t, r, factory, "acme/app")
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: models.HumanReviewTriggerNodeID, Type: models.NodeTypeTrigger,
		Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}}),
		Configuration: datatypes.NewJSONType(map[string]any{"enabled": true, "defaultReviewer": "owner"}),
		Metadata:      datatypes.NewJSONType(models.FactoryAppTemplateMetadata(models.FactoryAppTemplateHumanReviewID, 1)),
	}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	summary, err := prbody.UpdateBodySection("", "human-review", "## SuperPlane Request review\n\nWaiting for merge confidence.\n\nReviewed commit: current-head")
	require.NoError(t, err)
	current, err := json.Marshal(map[string]any{"state": "open", "head": map[string]any{"sha": "current-head"}, "body": summary})
	require.NoError(t, err)
	host := &supportcontexts.HTTPContext{Responses: []*http.Response{
		publicationHTTPResponse(200, `{"state":"open","head":{"sha":"current-head"}}`),
		publicationHTTPResponse(200, `{"check_runs":[]}`),
		publicationHTTPResponse(201, `{"id":123}`),
		publicationHTTPResponse(200, `{"state":"open","head":{"sha":"current-head"}}`),
		publicationHTTPResponse(200, `{}`),
		publicationHTTPResponse(200, `{"check_runs":[]}`),
		publicationHTTPResponse(200, `{"id":123}`),
		publicationHTTPResponse(200, string(current)),
	}}
	installPublicationGitHub(t, host)

	require.NoError(t, reconcileFactoryHumanReview(t.Context(), IntakeDependencies{}, *pr))

	decision, err := pr.CurrentHumanReview(db)
	require.NoError(t, err)
	require.NotNil(t, decision)
	require.Equal(t, models.FactoryHumanReviewStateWaitingAssessment, decision.State)
	require.Equal(t, "current-head", decision.AssessedCommit)
	require.Equal(t, int64(123), decision.CheckRunID)
	require.Len(t, host.Requests, 8)
	var check map[string]any
	require.NoError(t, json.NewDecoder(host.Requests[2].Body).Decode(&check))
	require.Equal(t, humanReviewCheckName, check["name"])
	require.Equal(t, "current-head", check["head_sha"])
	require.Equal(t, "queued", check["status"])
	requests, err := models.ListPullRequestChecksStartRequests(db, pr.ID)
	require.NoError(t, err)
	require.Len(t, requests, 1)
}
