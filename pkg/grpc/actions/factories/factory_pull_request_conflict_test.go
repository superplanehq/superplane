package factories

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestRefreshFactoryPullRequestMergeabilityFromMergedPullRequest(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	useFastMergeabilityUnknownPoll(t)
	_, sibling := createOpenGitHubFactoryPullRequest(t, db, r, "acme/app", 11)
	_, otherBase := createOpenGitHubFactoryPullRequest(t, db, r, "acme/app", 12)
	integration := readyGitHubIntegration(t, db, r)

	stubFactoryGitHub(t, &pullRequestByNumberGitHub{
		byNumber: map[int]*github.PullRequest{
			11: conflictingGitHubPullRequest("abc123def456", "feature", "main"),
			12: conflictingGitHubPullRequest("def456abc123", "other", "develop"),
		},
	})
	silenceFactoryWorkOrderUpdates(t)

	RefreshFactoryPullRequestMergeabilityFromGitHubEvent(
		t.Context(),
		IntakeDependencies{},
		&models.Webhook{AppInstallationID: &integration.ID},
		"pull_request",
		[]byte(`{
			"action": "closed",
			"repository": {"full_name": "acme/app"},
			"pull_request": {"number": 10, "merged": true, "base": {"ref": "main"}, "head": {"sha": "merged"}}
		}`),
	)

	storedSibling := reloadFactoryPullRequest(t, db, sibling)
	assert.Equal(t, "CONFLICTING", storedSibling.MergeBlockedReason)
	storedOther := reloadFactoryPullRequest(t, db, otherBase)
	assert.Empty(t, storedOther.MergeBlockedReason)
}

func TestRefreshFactoryPullRequestMergeabilityPollsUnknownMergeability(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	useFastMergeabilityUnknownPoll(t)
	factory, pullRequest := createOpenGitHubFactoryPullRequest(t, db, r, "acme/app", 21)
	require.NoError(t, pullRequest.SetMergeability(db, models.FactoryPullRequestMergeabilitySnapshot{
		Mergeable: true,
		HeadSHA:   "clean-sha",
	}))
	pullRequest = reloadFactoryPullRequest(t, db, pullRequest)

	sequenced := &sequencedPullRequestGitHub{
		pulls: []*github.PullRequest{
			unknownGitHubPullRequest("abc123def456", "feature", "main"),
			conflictingGitHubPullRequest("abc123def456", "feature", "main"),
		},
	}
	sequenced.onGet = func(call int) {
		if call != 1 {
			return
		}
		stored := reloadFactoryPullRequest(t, db, pullRequest)
		assert.True(t, stored.Mergeable)
		assert.Empty(t, stored.MergeBlockedReason)
		assert.Equal(t, "clean-sha", stored.MergeableHeadSHA)
	}
	stubFactoryGitHub(t, sequenced)
	silenceFactoryWorkOrderUpdates(t)

	err := pollFactoryPullRequestMergeability(t.Context(), db, IntakeDependencies{}, factory, pullRequest, "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, sequenced.calls, 2)

	stored := reloadFactoryPullRequest(t, db, pullRequest)
	assert.Equal(t, "CONFLICTING", stored.MergeBlockedReason)
	assert.False(t, stored.Mergeable)
}

func TestPushRefreshSchedulesUnknownMergeability(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	_, unknownPullRequest := createOpenGitHubFactoryPullRequest(t, db, r, "acme/app", 11)
	_, conflictingPullRequest := createOpenGitHubFactoryPullRequest(t, db, r, "acme/app", 12)
	integration := readyGitHubIntegration(t, db, r)

	previousPoll := startFactoryMergeabilityUnknownPoll
	var scheduled []uuid.UUID
	startFactoryMergeabilityUnknownPoll = func(_ context.Context, _ IntakeDependencies, _, _, pullRequestID uuid.UUID) {
		scheduled = append(scheduled, pullRequestID)
	}
	t.Cleanup(func() { startFactoryMergeabilityUnknownPoll = previousPoll })

	previousDelays := factoryMergeabilityUnknownDelays
	factoryMergeabilityUnknownDelays = []time.Duration{time.Hour}
	t.Cleanup(func() { factoryMergeabilityUnknownDelays = previousDelays })

	stubFactoryGitHub(t, &pullRequestByNumberGitHub{
		byNumber: map[int]*github.PullRequest{
			11: unknownGitHubPullRequest("abc123def456", "feature", "main"),
			12: conflictingGitHubPullRequest("def456abc123", "other", "main"),
		},
	})
	silenceFactoryWorkOrderUpdates(t)

	RefreshFactoryPullRequestMergeabilityFromGitHubEvent(
		t.Context(),
		IntakeDependencies{},
		&models.Webhook{AppInstallationID: &integration.ID},
		"push",
		[]byte(`{"ref":"refs/heads/main","repository":{"full_name":"acme/app"}}`),
	)

	assert.Equal(t, []uuid.UUID{unknownPullRequest.ID}, scheduled)
	stored := reloadFactoryPullRequest(t, db, conflictingPullRequest)
	assert.Equal(t, "CONFLICTING", stored.MergeBlockedReason)
	unknownStored := reloadFactoryPullRequest(t, db, unknownPullRequest)
	assert.Empty(t, unknownStored.MergeBlockedReason)
}

func TestConflictRepairDoesNotKeepClaimWithoutEvent(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factory, pullRequest := createOpenGitHubFactoryPullRequest(t, db, r, "acme/app", 41)
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{
			NodeID: "start",
			Name:   "Start",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "start"},
			}),
		}},
		nil,
	)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	_, err := factory.CreatePRFeedbackHandler(
		db,
		canvas.ID,
		models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest,
		models.FactoryPRFeedbackHandlerSourcePullRequestConflicts,
	)
	require.NoError(t, err)

	stubFactoryGitHub(t, &fakeFactoryGitHub{
		pullRequest: conflictingGitHubPullRequest("abc123def456", "feature", "main"),
	})
	silenceFactoryWorkOrderUpdates(t)

	require.NoError(t, refreshFactoryPullRequestMergeability(t.Context(), db, IntakeDependencies{}, factory, pullRequest))

	var claims int64
	require.NoError(t, db.Model(&models.FactoryPRConflictClaim{}).
		Where("pull_request_id = ?", pullRequest.ID).
		Count(&claims).Error)
	assert.Equal(t, int64(0), claims)
	assert.Equal(t, int64(0), countCanvasEvents(t, db, canvas.ID, "on-pull-request-conflict"))
}

func TestStartFactoryPullRequestConflictRepairEmitsOneEventPerHead(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	useFastMergeabilityUnknownPoll(t)
	factory, pullRequest := createOpenGitHubFactoryPullRequest(t, db, r, "acme/app", 31)
	canvas := conflictTriggerCanvas(t, r, factory.ID)
	handler, err := factory.CreatePRFeedbackHandler(
		db,
		canvas.ID,
		models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest,
		models.FactoryPRFeedbackHandlerSourcePullRequestConflicts,
	)
	require.NoError(t, err)
	require.NoError(t, handler.SetMaximumAttempts(db, 3))

	stubFactoryGitHub(t, &fakeFactoryGitHub{
		pullRequest: conflictingGitHubPullRequest("abc123def456", "feature", "main"),
	})
	silenceFactoryWorkOrderUpdates(t)

	require.NoError(t, refreshFactoryPullRequestMergeability(t.Context(), db, IntakeDependencies{}, factory, pullRequest))
	require.NoError(t, refreshFactoryPullRequestMergeability(t.Context(), db, IntakeDependencies{}, factory, pullRequest))
	assert.Equal(t, int64(1), countCanvasEvents(t, db, canvas.ID, "on-pull-request-conflict"))

	stubFactoryGitHub(t, &fakeFactoryGitHub{
		pullRequest: conflictingGitHubPullRequest("fff123def456", "feature", "main"),
	})
	require.NoError(t, refreshFactoryPullRequestMergeability(t.Context(), db, IntakeDependencies{}, factory, pullRequest))
	assert.Equal(t, int64(2), countCanvasEvents(t, db, canvas.ID, "on-pull-request-conflict"))

	attempt := 1
	limit := 1
	root := support.EmitCanvasEventForNode(t, canvas.ID, "on-pull-request-conflict", "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, root)
	require.NoError(t, err)
	require.NoError(t, handler.SetMaximumAttempts(db, 1))
	require.NoError(t, db.Create(&models.FactoryPullRequestRun{
		PullRequestID:     pullRequest.ID,
		RunID:             run.ID,
		FeedbackHandlerID: &handler.ID,
		Access:            models.FactoryPullRequestAccessExclusive,
		State:             models.FactoryPullRequestActivityStateFinished,
		Attempt:           &attempt,
		AttemptLimit:      &limit,
	}).Error)
	require.NoError(t, handler.SetMaximumAttempts(db, 1))

	stubFactoryGitHub(t, &fakeFactoryGitHub{
		pullRequest: conflictingGitHubPullRequest("999123def456", "feature", "main"),
	})
	beforeLimit := countCanvasEvents(t, db, canvas.ID, "on-pull-request-conflict")
	require.NoError(t, refreshFactoryPullRequestMergeability(t.Context(), db, IntakeDependencies{}, factory, pullRequest))
	assert.Equal(t, beforeLimit, countCanvasEvents(t, db, canvas.ID, "on-pull-request-conflict"))
}

type pullRequestByNumberGitHub struct {
	fakeFactoryGitHub
	byNumber map[int]*github.PullRequest
}

func (g *pullRequestByNumberGitHub) GetPullRequest(_ context.Context, _ string, number int) (*github.PullRequest, *github.Response, error) {
	if pullRequest, ok := g.byNumber[number]; ok {
		return pullRequest, nil, nil
	}
	return g.pullRequest, nil, nil
}

type sequencedPullRequestGitHub struct {
	fakeFactoryGitHub
	pulls []*github.PullRequest
	calls int
	onGet func(call int)
}

func (g *sequencedPullRequestGitHub) GetPullRequest(_ context.Context, _ string, _ int) (*github.PullRequest, *github.Response, error) {
	g.calls++
	if g.onGet != nil {
		g.onGet(g.calls)
	}
	index := g.calls - 1
	if index >= len(g.pulls) {
		index = len(g.pulls) - 1
	}
	return g.pulls[index], nil, nil
}

func conflictingGitHubPullRequest(sha, headRef, baseRef string) *github.PullRequest {
	return &github.PullRequest{
		Mergeable:      github.Ptr(false),
		MergeableState: github.Ptr("dirty"),
		Draft:          github.Ptr(false),
		Head:           &github.PullRequestBranch{SHA: github.Ptr(sha), Ref: github.Ptr(headRef)},
		Base:           &github.PullRequestBranch{Ref: github.Ptr(baseRef)},
	}
}

func unknownGitHubPullRequest(sha, headRef, baseRef string) *github.PullRequest {
	return &github.PullRequest{
		MergeableState: github.Ptr("unknown"),
		Draft:          github.Ptr(false),
		Head:           &github.PullRequestBranch{SHA: github.Ptr(sha), Ref: github.Ptr(headRef)},
		Base:           &github.PullRequestBranch{Ref: github.Ptr(baseRef)},
	}
}

func useFastMergeabilityUnknownPoll(t *testing.T) {
	t.Helper()
	previous := factoryMergeabilityUnknownDelays
	factoryMergeabilityUnknownDelays = []time.Duration{0}
	t.Cleanup(func() { factoryMergeabilityUnknownDelays = previous })
}

func silenceFactoryWorkOrderUpdates(t *testing.T) {
	t.Helper()
	original := publishFactoryWorkOrderUpdated
	publishFactoryWorkOrderUpdated = func(string, string, string) error { return nil }
	t.Cleanup(func() { publishFactoryWorkOrderUpdated = original })
}

func readyGitHubIntegration(t *testing.T, db *gorm.DB, r *support.ResourceRegistry) *models.Integration {
	t.Helper()
	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"github",
		support.RandomName("github"),
		map[string]any{},
	)
	require.NoError(t, err)
	require.NoError(t, db.Model(integration).Update("state", models.IntegrationStateReady).Error)
	return integration
}

func reloadFactoryPullRequest(t *testing.T, db *gorm.DB, pullRequest *models.FactoryPullRequest) *models.FactoryPullRequest {
	t.Helper()
	var stored models.FactoryPullRequest
	require.NoError(t, db.First(&stored, "id = ?", pullRequest.ID).Error)
	return &stored
}

func conflictTriggerCanvas(t *testing.T, r *support.ResourceRegistry, factoryID uuid.UUID) *models.Canvas {
	t.Helper()
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{
			NodeID: "on-pull-request-conflict",
			Name:   "On Merge Conflict",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "onPullRequestConflict"},
			}),
		}},
		nil,
	)
	require.NoError(t, database.Conn().Model(canvas).Update("factory_id", factoryID).Error)
	canvas.FactoryID = &factoryID
	return canvas
}

func countCanvasEvents(t *testing.T, db *gorm.DB, canvasID uuid.UUID, nodeID string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&models.CanvasEvent{}).
		Where("workflow_id = ? AND node_id = ?", canvasID, nodeID).
		Count(&count).Error)
	return count
}
