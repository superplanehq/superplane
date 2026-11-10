package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestHumanReviewPublicationClaimsAreExclusiveAndRecoverable(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
	require.NoError(t, err)
	pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
	require.NoError(t, err)

	type claimResult struct {
		token uuid.UUID
		err   error
	}
	start := make(chan struct{})
	results := make(chan claimResult, 2)
	for range 2 {
		go func() {
			<-start
			token, err := pr.ClaimHumanReviewPublication(db, time.Minute)
			results <- claimResult{token, err}
		}()
	}
	close(start)
	var owner uuid.UUID
	for range 2 {
		result := <-results
		require.NoError(t, result.err)
		if result.token != uuid.Nil {
			require.Equal(t, uuid.Nil, owner, "only one contender can claim the PR")
			owner = result.token
		}
	}
	require.NotEqual(t, uuid.Nil, owner)
	require.ErrorIs(t, db.Transaction(func(tx *gorm.DB) error {
		return pr.RequireHumanReviewPublicationIdle(tx)
	}), models.ErrHumanReviewPublicationBusy)

	other, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/43"})
	require.NoError(t, err)
	otherOwner, err := other.ClaimHumanReviewPublication(db, time.Minute)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, otherOwner, "another PR can publish independently")
	require.NoError(t, other.ReleaseHumanReviewPublication(db, otherOwner))

	// Model a crashed publisher whose claim has expired.
	require.NoError(t, db.Exec("UPDATE factory_human_review_publication_leases SET expires_at = ? WHERE pull_request_id = ?", time.Now().Add(-time.Minute), pr.ID).Error)
	replacement, err := pr.ClaimHumanReviewPublication(db, time.Minute)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, replacement)
	require.NotEqual(t, owner, replacement)
	require.NoError(t, pr.ReleaseHumanReviewPublication(db, owner))
	blocked, err := pr.ClaimHumanReviewPublication(db, time.Minute)
	require.NoError(t, err)
	require.Equal(t, uuid.Nil, blocked, "the old publisher must not release its replacement")

	require.NoError(t, pr.ReleaseHumanReviewPublication(db, replacement))
	next, err := pr.ClaimHumanReviewPublication(db, time.Minute)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, next, "completed work releases its claim immediately")
	require.NoError(t, pr.ReleaseHumanReviewPublication(db, next))

	// A policy transaction must exclude new publishers until it commits.
	tx := db.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	require.NoError(t, pr.RequireHumanReviewPublicationIdle(tx))
	blocked, err = pr.ClaimHumanReviewPublication(db, time.Minute)
	require.NoError(t, err)
	require.Equal(t, uuid.Nil, blocked)
	require.NoError(t, tx.Commit().Error)
	next, err = pr.ClaimHumanReviewPublication(db, time.Minute)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, next)
	require.NoError(t, pr.ReleaseHumanReviewPublication(db, next))
}
