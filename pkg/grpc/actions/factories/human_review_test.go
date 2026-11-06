package factories

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

// A failed publication must leave the durable decision pending. The next
// reconciliation must reuse the same publication identity instead of losing it.
func TestHumanReviewPublicationRecoversAfterGitHubFailure(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("human-review"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
	require.NoError(t, err)
	pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
	require.NoError(t, err)
	revision, err := pr.ObserveRevision(db, "current-head")
	require.NoError(t, err)
	decision, err := pr.NewHumanReviewDecision(db, models.FactoryHumanReviewDecision{RevisionID: revision.Revision.ID, AssessedCommit: "current-head", State: models.FactoryHumanReviewStateWaitingAssessment})
	require.NoError(t, err)
	client := &humanReviewPublicationFake{failure: errors.New("GitHub unavailable")}
	gate := humanReviewPublication{Status: "queued", Reason: "Waiting for merge confidence."}
	require.Error(t, publishHumanReviewCheck(t.Context(), db, client, pr, decision, gate))
	stored, err := models.FindFactoryHumanReviewDecision(db, decision.ID)
	require.NoError(t, err)
	require.True(t, stored.PublicationPending)
	require.NotEmpty(t, stored.PublicationError)
	client.failure = nil
	require.NoError(t, publishHumanReviewCheck(t.Context(), db, client, pr, stored, gate))
	stored, err = models.FindFactoryHumanReviewDecision(db, decision.ID)
	require.NoError(t, err)
	require.Equal(t, int64(123), stored.CheckRunID)
	require.False(t, stored.PublicationPending)
	require.Empty(t, stored.PublicationError)
	require.Equal(t, []string{decision.ID.String(), decision.ID.String()}, client.externalIDs)
}

type humanReviewPublicationFake struct {
	failure     error
	externalIDs []string
	conclusions []string
}

func (f *humanReviewPublicationFake) CreateHumanReviewCheck(_ context.Context, _ string, options github.CreateCheckRunOptions) (*github.CheckRun, *github.Response, error) {
	f.externalIDs = append(f.externalIDs, options.GetExternalID())
	f.conclusions = append(f.conclusions, options.GetConclusion())
	if f.failure != nil {
		return nil, nil, f.failure
	}
	return &github.CheckRun{ID: github.Ptr(int64(123)), HTMLURL: github.Ptr("https://github.com/acme/app/runs/123")}, nil, nil
}
func (f *humanReviewPublicationFake) UpdateHumanReviewCheck(_ context.Context, _ string, _ int64, _ github.UpdateCheckRunOptions) (*github.CheckRun, *github.Response, error) {
	if f.failure != nil {
		return nil, nil, f.failure
	}
	return &github.CheckRun{ID: github.Ptr(int64(123))}, nil, nil
}
func (f *humanReviewPublicationFake) ListCheckRunsForRef(_ context.Context, _ string, _ string, _ *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
	return &github.ListCheckRunsResults{}, nil, nil
}

func TestHumanReviewSelectsEligibleFallbackAndPreservesExistingRequests(t *testing.T) {
	for _, scenario := range []struct {
		name             string
		requested        bool
		approved         bool
		multiple         bool
		changesRequested bool
	}{
		{name: "fallback excludes author bots and readers"},
		{name: "existing selected request is reused", requested: true},
		{name: "current approval completes the gate", approved: true},
		{name: "multiple justified selections preserve existing requests", requested: true, multiple: true},
		{name: "later change request restores the approval request", changesRequested: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("human-review"), "", "")
			require.NoError(t, err)
			order, err := factory.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
			require.NoError(t, err)
			pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
			require.NoError(t, err)
			revision, err := pr.ObserveRevision(db, "head")
			require.NoError(t, err)
			decision, err := pr.NewHumanReviewDecision(db, models.FactoryHumanReviewDecision{RevisionID: revision.Revision.ID, AssessedCommit: "head", State: models.FactoryHumanReviewStateDecided, Result: datatypes.JSONMap{
				"assessedCommit": "head", "reviewRequired": true, "rationale": "Security changes need human approval.",
				"selectedReviewers": []any{map[string]any{"id": 1, "login": "author", "reason": "Repository owner."}, map[string]any{"id": 2, "login": "robot[bot]", "reason": "Recent contributor."}, map[string]any{"id": 3, "login": "reader", "reason": "Code owner."}},
			}})
			require.NoError(t, err)
			author := reviewTestUser(1, "author", "User")
			bot := reviewTestUser(2, "robot[bot]", "Bot")
			reader := reviewTestUser(3, "reader", "User")
			alice := reviewTestUser(4, "alice", "User")
			bob := reviewTestUser(5, "bob", "User")
			current := &github.PullRequest{State: github.Ptr("open"), User: author, Head: &github.PullRequestBranch{SHA: github.Ptr("head")}}
			if scenario.requested {
				current.RequestedReviewers = []*github.User{alice}
			}
			client := &humanReviewLifecycleFake{current: current, members: []*github.User{author, bot, reader, alice}, permissions: map[string]*github.RepositoryPermissionLevel{
				"author":     {User: author, Permission: github.Ptr("admin")},
				"robot[bot]": {User: bot, Permission: github.Ptr("write")},
				"reader":     {User: reader, Permission: github.Ptr("read")},
				"alice":      {User: alice, Permission: github.Ptr("write")},
				"bob":        {User: bob, Permission: github.Ptr("write")},
			}}
			if scenario.multiple {
				decision.Result["selectedReviewers"] = []any{map[string]any{"id": 4, "login": "alice", "reason": "Preserve the existing security review."}, map[string]any{"id": 5, "login": "bob", "reason": "Additional database expertise is required."}}
				require.NoError(t, decision.Save(db))
			}
			if scenario.approved {
				client.reviews = []*github.PullRequestReview{{ID: github.Ptr(int64(99)), User: alice, State: github.Ptr("APPROVED"), CommitID: github.Ptr("head")}}
			}
			if scenario.changesRequested {
				now := time.Now()
				client.reviews = []*github.PullRequestReview{
					{ID: github.Ptr(int64(98)), User: alice, State: github.Ptr("APPROVED"), CommitID: github.Ptr("head"), SubmittedAt: &github.Timestamp{Time: now.Add(-time.Minute)}},
					{ID: github.Ptr(int64(99)), User: alice, State: github.Ptr("CHANGES_REQUESTED"), CommitID: github.Ptr("head"), SubmittedAt: &github.Timestamp{Time: now}},
				}
			}
			automation := &humanReviewAutomation{settings: models.HumanReviewSettings{Enabled: true, FallbackTeam: "acme/reviewers"}}
			require.NoError(t, evaluateHumanReviewDecision(t.Context(), db, client, pr, current, automation, decision))
			stored, err := models.FindFactoryHumanReviewDecision(db, decision.ID)
			require.NoError(t, err)
			if scenario.multiple {
				require.Len(t, stored.SelectedReviewers, 2)
				require.Equal(t, int64(5), stored.SelectedReviewers[1].ID)
			} else {
				require.Len(t, stored.SelectedReviewers, 1)
			}
			require.Equal(t, int64(4), stored.SelectedReviewers[0].ID)
			if scenario.multiple {
				require.Equal(t, [][]string{{"bob"}}, client.requests)
			} else if scenario.requested || scenario.approved {
				require.Empty(t, client.requests)
			} else {
				require.Equal(t, [][]string{{"alice"}}, client.requests)
			}
			if scenario.approved {
				require.Equal(t, []int64{99}, []int64(stored.ApprovalIDs))
				require.Empty(t, stored.BlockingReason)
				require.Equal(t, []string{"success"}, client.conclusions)
			} else {
				require.NotEmpty(t, stored.BlockingReason)
				require.Equal(t, []string{"failure"}, client.conclusions)
			}
		})
	}
}

func reviewTestUser(id int64, login, userType string) *github.User {
	return &github.User{ID: &id, Login: &login, Type: &userType}
}

type humanReviewLifecycleFake struct {
	humanReviewPublicationFake
	current     *github.PullRequest
	members     []*github.User
	permissions map[string]*github.RepositoryPermissionLevel
	reviews     []*github.PullRequestReview
	requests    [][]string
}

func (f *humanReviewLifecycleFake) GetPullRequest(context.Context, string, int) (*github.PullRequest, *github.Response, error) {
	return f.current, nil, nil
}
func (f *humanReviewLifecycleFake) ListReviews(context.Context, string, int) ([]*github.PullRequestReview, error) {
	return f.reviews, nil
}
func (f *humanReviewLifecycleFake) GetRepositoryPermissionLevel(_ context.Context, _ string, login string) (*github.RepositoryPermissionLevel, *github.Response, error) {
	return f.permissions[login], nil, nil
}
func (f *humanReviewLifecycleFake) AddPullRequestReviewers(_ context.Context, _ string, _ int, request github.ReviewersRequest) (*github.PullRequest, *github.Response, error) {
	f.requests = append(f.requests, request.Reviewers)
	return f.current, nil, nil
}
func (f *humanReviewLifecycleFake) HumanReviewTeamMembers(context.Context, string, string) ([]*github.User, error) {
	return f.members, nil
}
func (f *humanReviewLifecycleFake) HumanReviewChangedFiles(context.Context, string, int) ([]*github.CommitFile, error) {
	return nil, nil
}
func (f *humanReviewLifecycleFake) HumanReviewOwnershipFile(context.Context, string, string, string) (string, error) {
	return "", nil
}
func (f *humanReviewLifecycleFake) ListCommits(context.Context, string, *github.CommitsListOptions) ([]*github.RepositoryCommit, *github.Response, error) {
	return nil, nil, nil
}

func TestHumanReviewNewAssessmentForSameCommitSupersedesOldDecision(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("human-review"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
	require.NoError(t, err)
	pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
	require.NoError(t, err)
	observed, err := pr.ObserveRevision(db, "same-head")
	require.NoError(t, err)
	assessmentCanvas := createMergeConfidenceStartupCanvas(t, r, factory, "acme/app")
	humanCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: models.HumanReviewTriggerNodeID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}}), Configuration: datatypes.NewJSONType(map[string]any{"enabled": true, "fallbackTeam": "acme/reviewers"})}}, nil)
	require.NoError(t, db.Model(humanCanvas).Update("factory_id", factory.ID).Error)
	require.NoError(t, humanCanvas.StampFactoryAppTemplate(db, models.HumanReviewTriggerNodeID, models.FactoryAppTemplateHumanReviewID, 1))
	automation, err := findHumanReviewAutomation(db, factory)
	require.NoError(t, err)
	require.NotNil(t, automation)
	createAssessment := func() *models.CanvasRun {
		event := &models.CanvasEvent{WorkflowID: assessmentCanvas.ID, NodeID: "renamed-pr-trigger", Data: models.NewJSONValue(map[string]any{"type": "github.pullRequest", "data": map[string]any{"pull_request": map[string]any{"head": map[string]any{"sha": "same-head"}}}})}
		require.NoError(t, db.Create(event).Error)
		require.NoError(t, pr.LinkRun(db, event.RunID, "Merge confidence"))
		run, err := models.FindCanvasRunInTransaction(db, assessmentCanvas.ID, event.RunID)
		require.NoError(t, err)
		return run
	}
	oldSource := createAssessment()
	oldDecision, err := pr.NewHumanReviewDecision(db, models.FactoryHumanReviewDecision{RevisionID: observed.Revision.ID, SourceRunID: &oldSource.ID, AssessedCommit: "same-head", ConfigurationVersion: automation.version.ID.String(), State: models.FactoryHumanReviewStateDecided})
	require.NoError(t, err)
	newerSource := createAssessment()
	client := &humanReviewLifecycleFake{current: &github.PullRequest{State: github.Ptr("open"), User: reviewTestUser(1, "author", "User"), Head: &github.PullRequestBranch{SHA: github.Ptr("same-head")}}}
	require.NoError(t, reconcileHumanReview(t.Context(), db, IntakeDependencies{}, factory, pr, automation, client))
	current, err := pr.CurrentHumanReview(db)
	require.NoError(t, err)
	require.NotEqual(t, oldDecision.ID, current.ID)
	require.Equal(t, newerSource.ID, *current.SourceRunID)
	require.Equal(t, models.FactoryHumanReviewStateWaitingAssessment, current.State)
	require.Nil(t, current.RunID)
	require.NoError(t, publishHumanReviewCheck(t.Context(), db, client, pr, oldDecision, humanReviewPublication{Status: "completed", Conclusion: "success", Reason: "Old favorable decision."}))
	require.Equal(t, []string{""}, client.conclusions, "only the queued current assessment can publish")
	require.NoError(t, reconcileHumanReview(t.Context(), db, IntakeDependencies{}, factory, pr, automation, client))
	duplicate, err := pr.CurrentHumanReview(db)
	require.NoError(t, err)
	require.Equal(t, current.ID, duplicate.ID)
}

func TestHumanReviewReopensOnlyPullRequestClosedTasksAfterLiveConfirmation(t *testing.T) {
	for _, liveOpen := range []bool{false, true} {
		t.Run(fmt.Sprintf("live open %t", liveOpen), func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("human-review"), "", "")
			require.NoError(t, err)
			repository := "acme/app"
			require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{AppRepository: &repository}))
			order, err := factory.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
			require.NoError(t, err)
			pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
			require.NoError(t, err)
			require.NoError(t, db.Model(order).Update("state", models.FactoryWorkOrderStateOpen).Error)
			_, err = applyGitHubPullRequestClosed(db, factory, pr, false, nil, nil, nil)
			require.NoError(t, err)
			require.Equal(t, models.FactoryPullRequestStateClosed, pr.State)
			assessmentCanvas := createMergeConfidenceStartupCanvas(t, r, factory, models.WorkspaceRepositoriesExpression)
			canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: models.HumanReviewTriggerNodeID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}}), Configuration: datatypes.NewJSONType(map[string]any{"enabled": true, "fallbackTeam": "acme/reviewers"})}}, nil)
			require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
			require.NoError(t, canvas.StampFactoryAppTemplate(db, models.HumanReviewTriggerNodeID, models.FactoryAppTemplateHumanReviewID, 1))
			automation, err := findHumanReviewAutomation(db, factory)
			require.NoError(t, err)
			state := "closed"
			if liveOpen {
				state = "open"
			}
			client := &humanReviewLifecycleFake{current: &github.PullRequest{State: &state, User: reviewTestUser(1, "author", "User"), Head: &github.PullRequestBranch{SHA: github.Ptr("reopened-head")}}}
			require.NoError(t, reconcileHumanReview(t.Context(), db, IntakeDependencies{}, factory, pr, automation, client))
			stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pr.ID})
			require.NoError(t, err)
			task, err := factory.FindWorkOrder(db, order.ID)
			require.NoError(t, err)
			requests, err := models.ListPullRequestChecksStartRequests(db, pr.ID)
			require.NoError(t, err)
			if liveOpen {
				require.Equal(t, models.FactoryPullRequestStateOpen, stored.State)
				require.Nil(t, stored.ClosedAt)
				require.True(t, task.IsOpen())
				decision, err := pr.CurrentHumanReview(db)
				require.NoError(t, err)
				require.NotNil(t, decision)
				require.Equal(t, "reopened-head", decision.AssessedCommit)
				require.Equal(t, models.FactoryHumanReviewStateWaitingAssessment, decision.State)
				require.Equal(t, []string{""}, client.conclusions)
				require.Len(t, requests, 1)
				require.Equal(t, assessmentCanvas.ID, requests[0].WorkflowID)
			} else {
				require.Equal(t, models.FactoryPullRequestStateClosed, stored.State)
				require.False(t, task.IsOpen())
				require.Empty(t, client.externalIDs)
				require.Empty(t, requests)
			}
		})
	}
}

func TestDisabledHumanReviewPublishesExplanatorySuccessForNewTrackedPullRequest(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("human-review"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
	require.NoError(t, err)
	pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
	require.NoError(t, err)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: models.HumanReviewTriggerNodeID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}}), Configuration: datatypes.NewJSONType(map[string]any{"enabled": false})}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	require.NoError(t, canvas.StampFactoryAppTemplate(db, models.HumanReviewTriggerNodeID, models.FactoryAppTemplateHumanReviewID, 1))
	automation, err := findHumanReviewAutomation(db, factory)
	require.NoError(t, err)
	client := &humanReviewLifecycleFake{current: &github.PullRequest{State: github.Ptr("open"), User: reviewTestUser(1, "author", "User"), Head: &github.PullRequestBranch{SHA: github.Ptr("new-head")}}}
	require.NoError(t, reconcileHumanReview(t.Context(), db, IntakeDependencies{}, factory, pr, automation, client))
	decision, err := pr.CurrentHumanReview(db)
	require.NoError(t, err)
	require.NotNil(t, decision)
	require.Equal(t, models.FactoryHumanReviewStateDisabled, decision.State)
	require.Equal(t, int64(123), decision.CheckRunID)
	require.Nil(t, decision.RunID)
	require.Equal(t, []string{"success"}, client.conclusions)
	require.Empty(t, decision.BlockingReason)
	require.Empty(t, client.requests)
}
