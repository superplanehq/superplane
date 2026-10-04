package models_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestClaimFactoryPullRequestConflictHead(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas := support.CreateFactoryCanvas(t, r, factory.ID, "Fix merge conflicts")
	handler, err := factory.CreatePRFeedbackHandler(
		db,
		canvas.ID,
		models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest,
		models.FactoryPRFeedbackHandlerSourcePullRequestConflicts,
	)
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL: "https://github.com/acme/app/pull/8",
	})
	require.NoError(t, err)

	claimed, err := models.ClaimFactoryPullRequestConflictHead(db, handler.ID, pullRequest.ID, "")
	require.NoError(t, err)
	assert.False(t, claimed)

	claimed, err = models.ClaimFactoryPullRequestConflictHead(db, handler.ID, pullRequest.ID, "abc123")
	require.NoError(t, err)
	assert.True(t, claimed)

	claimed, err = models.ClaimFactoryPullRequestConflictHead(db, handler.ID, pullRequest.ID, "abc123")
	require.NoError(t, err)
	assert.False(t, claimed)

	claimed, err = models.ClaimFactoryPullRequestConflictHead(db, handler.ID, pullRequest.ID, "def456")
	require.NoError(t, err)
	assert.True(t, claimed)

	order, err = factory.CreateWorkOrder(db, "Race", "", &r.User, nil, nil)
	require.NoError(t, err)
	raced, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL: "https://github.com/acme/app/pull/9",
	})
	require.NoError(t, err)

	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, claimErr := models.ClaimFactoryPullRequestConflictHead(db, handler.ID, raced.ID, "same-head")
			if claimErr != nil {
				t.Errorf("claim conflict head: %v", claimErr)
				return
			}
			if ok {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(1), wins.Load())
}
