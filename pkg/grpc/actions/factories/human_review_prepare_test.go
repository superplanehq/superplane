package factories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// A broadcast is only a notification. Its source must match the stored latest
// assessment, and only one run may claim reviewer selection for that assessment.
func TestPrepareHumanReviewBroadcastAuthority(t *testing.T) {
	for _, scenario := range []string{"current", "duplicate", "stale run", "foreign automation", "foreign PR", "old commit", "premature", "failed", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			f := newHumanReviewReconciliationFixture(t)
			sourceID, appID, prID, sha := f.source.ID.String(), f.source.WorkflowID.String(), f.pr.ID.String(), "current-head"
			switch scenario {
			case "stale run":
				sourceID = uuid.NewString()
			case "foreign automation":
				appID = uuid.NewString()
			case "foreign PR":
				prID = uuid.NewString()
			case "old commit":
				sha = "old-head"
			case "premature":
				require.NoError(t, f.db.Model(f.source).Update("state", models.CanvasRunStateStarted).Error)
			case "failed":
				require.NoError(t, f.db.Model(f.source).Update("result", models.CanvasRunResultFailed).Error)
			}
			root := humanReviewBroadcastRoot(t, f, appID, sourceID, prID, sha)
			run, err := models.FindCanvasRunInTransaction(f.db, f.automation.canvas.ID, root.RunID)
			require.NoError(t, err)
			if scenario == "cancelled" {
				require.NoError(t, f.db.Model(run).Update("cancelled_at", time.Now()).Error)
			}
			prepared, err := prepareHumanReview(t.Context(), f.db, f.factory, f.pr, f.automation, f.client, run, root)
			require.NoError(t, err)
			valid := scenario == "current" || scenario == "duplicate"
			require.Equal(t, valid, prepared.AgentNeeded)
			if scenario == "duplicate" {
				again := humanReviewBroadcastRoot(t, f, appID, sourceID, prID, sha)
				duplicateRun, err := models.FindCanvasRunInTransaction(f.db, f.automation.canvas.ID, again.RunID)
				require.NoError(t, err)
				duplicate, err := prepareHumanReview(t.Context(), f.db, f.factory, f.pr, f.automation, f.client, duplicateRun, again)
				require.NoError(t, err)
				require.False(t, duplicate.AgentNeeded)
				require.Equal(t, prepared.Decision.ID, duplicate.Decision.ID)
				require.Equal(t, run.ID, *duplicate.Decision.RunID)
			}
			if scenario == "premature" {
				require.NoError(t, f.db.Model(f.source).Update("state", models.CanvasRunStateFinished).Error)
				recovery := humanReviewBroadcastRoot(t, f, appID, sourceID, prID, sha)
				recoveryRun, err := models.FindCanvasRunInTransaction(f.db, f.automation.canvas.ID, recovery.RunID)
				require.NoError(t, err)
				recovered, err := prepareHumanReview(t.Context(), f.db, f.factory, f.pr, f.automation, f.client, recoveryRun, recovery)
				require.NoError(t, err)
				require.True(t, recovered.AgentNeeded)
			}
		})
	}
}

func TestHumanReviewFailedAgentNeedsExplicitRerun(t *testing.T) {
	f := newHumanReviewReconciliationFixture(t)
	f.reconcile(t)
	decision, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.NotNil(t, decision.RunID)
	require.NoError(t, f.db.Model(&models.CanvasRun{}).Where("id = ?", *decision.RunID).Updates(map[string]any{"state": models.CanvasRunStateFinished, "result": models.CanvasRunResultFailed}).Error)
	f.reconcile(t)
	failed, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.Equal(t, models.FactoryHumanReviewStateFailed, failed.State)
	f.reconcile(t)
	duplicate, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.Equal(t, failed.ID, duplicate.ID)
	require.Equal(t, failed.RunID, duplicate.RunID)
	failedRun, err := models.FindUnscopedCanvasRun(f.db, *failed.RunID)
	require.NoError(t, err)
	require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error { _, err := failedRun.DeleteChain(tx); return err }))
	f.reconcile(t)
	retained, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.Equal(t, failed.ID, retained.ID)
	require.Nil(t, retained.RunID, "retention must not restart a failed agent")
	root := humanReviewBroadcastRoot(t, f, f.source.WorkflowID.String(), f.source.ID.String(), f.pr.ID.String(), "current-head")
	envelope := root.Data.Data().(map[string]any)
	envelope["replayedFromEventId"] = uuid.NewString()
	root.Data = models.NewJSONValue(envelope)
	require.NoError(t, f.db.Model(root).Update("data", root.Data).Error)
	run, err := models.FindCanvasRunInTransaction(f.db, f.automation.canvas.ID, root.RunID)
	require.NoError(t, err)
	prepared, err := prepareHumanReview(t.Context(), f.db, f.factory, f.pr, f.automation, f.client, run, root)
	require.NoError(t, err)
	require.True(t, prepared.AgentNeeded)
	require.NotEqual(t, failed.ID, prepared.Decision.ID)
}

func TestHumanReviewCancellationInvalidatesSavedResult(t *testing.T) {
	f := newHumanReviewReconciliationFixture(t)
	f.reconcile(t)
	decision, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	f.complete(t, decision)
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	// Refresh recovers the stored completion through the same Record service.
	require.NoError(t, refreshHumanReviewGate(ctx, f.db.WithContext(ctx), f.factory, f.pr, f.automation, f.client))
	saved, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.NotEmpty(t, saved.Result)
	require.NoError(t, f.db.Model(&models.CanvasRun{}).Where("id = ?", *saved.RunID).Update("cancelled_at", time.Now()).Error)
	require.NoError(t, refreshHumanReviewGate(ctx, f.db.WithContext(ctx), f.factory, f.pr, f.automation, f.client))
	current, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.Equal(t, models.FactoryHumanReviewStateFailed, current.State)
	require.Equal(t, "failure", f.client.publications[len(f.client.publications)-1].Conclusion)
}

func humanReviewBroadcastRoot(t *testing.T, f humanReviewReconciliationFixture, appID, runID, prID, sha string) *models.CanvasEvent {
	t.Helper()
	root := &models.CanvasEvent{WorkflowID: f.automation.canvas.ID, NodeID: "on-merge-confidence", Data: models.NewJSONValue(map[string]any{"type": "app.broadcast", "data": map[string]any{"app": map[string]any{"id": appID}, "run": map[string]any{"id": runID}, "payload": map[string]any{"pullRequestId": prID, "assessedCommit": sha}}})}
	require.NoError(t, f.db.Create(root).Error)
	return root
}
