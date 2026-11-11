package factories

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestHumanReviewGateReleasesDatabaseDuringGitHubCalls(t *testing.T) {
	f := newHumanReviewReconciliationFixture(t)
	prepareHumanReviewForPublication(t, f)
	pool, err := f.db.DB()
	require.NoError(t, err)
	originalLimit := pool.Stats().MaxOpenConnections
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { pool.SetMaxOpenConns(originalLimit) })

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	db := f.db.WithContext(ctx)
	reads := 0
	client := &humanReviewPublicationLeaseFake{humanReviewSetupFake: f.client}
	client.onRead = func() error {
		reads++
		queryCtx, queryCancel := context.WithTimeout(ctx, time.Second)
		defer queryCancel()
		// Provider work must leave the only database connection available.
		if err := db.WithContext(queryCtx).Exec("SELECT 1").Error; err != nil {
			return err
		}
		if reads == 1 {
			err := refreshHumanReviewGate(queryCtx, db.WithContext(queryCtx), f.factory, f.pr, f.automation, client)
			if err != models.ErrHumanReviewPublicationBusy {
				return fmt.Errorf("competing publication returned %v", err)
			}
			if reads != 1 {
				return fmt.Errorf("competing publication reached GitHub")
			}
		}
		return nil
	}
	require.NoError(t, refreshHumanReviewGate(ctx, db, f.factory, f.pr, f.automation, client))
	require.Positive(t, reads)
	decision, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.NotNil(t, decision)
	require.NotZero(t, decision.CheckRunID)
}

func TestHumanReviewGateRetainsCancelledPublicationUntilExpiry(t *testing.T) {
	f := newHumanReviewReconciliationFixture(t)
	prepareHumanReviewForPublication(t, f)
	require.NoError(t, f.pr.RequestHumanReviewReconciliation(f.db))
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	client := &humanReviewPublicationLeaseFake{humanReviewSetupFake: f.client}
	client.onRead = func() error {
		cancel()
		return ctx.Err()
	}
	require.ErrorIs(t, refreshHumanReviewGate(ctx, f.db.WithContext(ctx), f.factory, f.pr, f.automation, client), context.Canceled)

	claim, err := f.pr.ClaimHumanReviewPublication(f.db, 2*time.Minute)
	require.NoError(t, err)
	require.Equal(t, uuid.Nil, claim, "cancelled HTTP requests need time to settle before another publication")
	pending, err := models.ListFactoryHumanReviewPullRequestsToReconcile(f.db, uuid.Nil, 100)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.NotNil(t, pending[0].HumanReviewRequestID)

	require.NoError(t, f.db.Exec("UPDATE factory_human_review_publication_leases SET expires_at = ? WHERE pull_request_id = ?", time.Now().Add(-time.Minute), f.pr.ID).Error)
	retryCtx, retryCancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer retryCancel()
	client.onRead = nil
	require.NoError(t, refreshHumanReviewGate(retryCtx, f.db.WithContext(retryCtx), f.factory, f.pr, f.automation, client))
	decision, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.NotNil(t, decision)
	require.NotZero(t, decision.CheckRunID)
	stored, err := f.factory.FindPullRequest(f.db, models.FactoryPullRequestLookup{ID: f.pr.ID})
	require.NoError(t, err)
	require.Nil(t, stored.HumanReviewRequestID)
}

func prepareHumanReviewForPublication(t *testing.T, f humanReviewReconciliationFixture) {
	t.Helper()
	root := &models.CanvasEvent{
		WorkflowID: f.automation.canvas.ID,
		NodeID:     models.HumanReviewTriggerNodeID,
		Data:       models.NewJSONValue(map[string]any{"type": "app.invocation", "data": map[string]any{"parameters": map[string]any{"pullRequestId": f.pr.ID.String()}}}),
	}
	require.NoError(t, f.db.Create(root).Error)
	run, err := models.FindCanvasRunInTransaction(f.db, f.automation.canvas.ID, root.RunID)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	prepared, err := prepareHumanReview(ctx, f.db.WithContext(ctx), f.factory, f.pr, f.automation, f.client, run, root)
	require.NoError(t, err)
	require.True(t, prepared.AgentNeeded)
}

type humanReviewPublicationLeaseFake struct {
	*humanReviewSetupFake
	onRead func() error
}

func (f *humanReviewPublicationLeaseFake) GetPullRequest(ctx context.Context, repository string, number int) (*github.PullRequest, *github.Response, error) {
	if f.onRead != nil {
		if err := f.onRead(); err != nil {
			return nil, nil, err
		}
	}
	return f.humanReviewSetupFake.GetPullRequest(ctx, repository, number)
}
