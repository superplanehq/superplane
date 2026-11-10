package factories

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestHumanReviewReconciliationReleasesDatabaseDuringGitHubCalls(t *testing.T) {
	f := newHumanReviewReconciliationFixture(t)
	pool, err := f.db.DB()
	require.NoError(t, err)
	originalLimit := pool.Stats().MaxOpenConnections
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { pool.SetMaxOpenConns(originalLimit) })

	reads := 0
	client := &humanReviewRecoveryFake{humanReviewSetupFake: f.client, factoryGitHubAPI: &fakeFactoryGitHub{}}
	client.onRead = func() error {
		reads++
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		// A provider call must leave the only connection available to other work.
		if err := f.db.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
			return err
		}
		if reads == 1 {
			// A second process must skip the active publication without waiting.
			if err := reconcileFactoryHumanReview(ctx, IntakeDependencies{}, *f.pr); err != nil {
				return err
			}
			if reads != 1 {
				return fmt.Errorf("competing reconciliation reached GitHub")
			}
		}
		return nil
	}
	stubFactoryGitHub(t, client)
	require.NoError(t, reconcileFactoryHumanReview(t.Context(), IntakeDependencies{}, *f.pr))
	require.Positive(t, reads)
	decision, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.NotNil(t, decision)
	require.NotZero(t, decision.CheckRunID)
}

func TestHumanReviewReconciliationRetainsCancelledPublicationUntilExpiry(t *testing.T) {
	f := newHumanReviewReconciliationFixture(t)
	require.NoError(t, f.pr.RequestHumanReviewReconciliation(f.db))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	client := &humanReviewRecoveryFake{humanReviewSetupFake: f.client, factoryGitHubAPI: &fakeFactoryGitHub{}}
	client.onRead = func() error {
		cancel()
		return ctx.Err()
	}
	stubFactoryGitHub(t, client)
	require.ErrorIs(t, reconcileFactoryHumanReview(ctx, IntakeDependencies{}, *f.pr), context.Canceled)

	claim, err := f.pr.ClaimHumanReviewPublication(f.db, 2*time.Minute)
	require.NoError(t, err)
	require.Equal(t, uuid.Nil, claim, "cancelled HTTP requests need time to settle before another publication")
	pending, err := models.ListFactoryHumanReviewPullRequestsToReconcile(f.db, uuid.Nil, 100)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.NotNil(t, pending[0].HumanReviewRequestID)

	require.NoError(t, f.db.Exec("UPDATE factory_human_review_publication_leases SET expires_at = ? WHERE pull_request_id = ?", time.Now().Add(-time.Minute), f.pr.ID).Error)
	client.onRead = nil
	require.NoError(t, reconcileFactoryHumanReview(t.Context(), IntakeDependencies{}, *f.pr))
	decision, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.NotNil(t, decision)
	require.NotZero(t, decision.CheckRunID)
	stored, err := f.factory.FindPullRequest(f.db, models.FactoryPullRequestLookup{ID: f.pr.ID})
	require.NoError(t, err)
	require.Nil(t, stored.HumanReviewRequestID)
}
