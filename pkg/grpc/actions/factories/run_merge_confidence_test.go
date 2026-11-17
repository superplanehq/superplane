package factories

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Exercise the action against real events and task state; only provider I/O is replaced.
func mergeConfidenceTask(t *testing.T, r *support.ResourceRegistry) (*models.Factory, *models.FactoryWorkOrder, *models.Canvas) {
	t.Helper()
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	canvas := createMergeConfidenceStartupCanvas(t, r, factory, "acme/app")
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "implement", "start")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry}})
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Review", "Saved task", &r.User, nil, nil)
	require.NoError(t, err)
	require.NoError(t, order.TransitionOnDispatch(db, &r.User))
	_, _, err = line.RecordPassedStageWithoutRun(db, order, 0)
	require.NoError(t, err)
	_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
	require.NoError(t, err)
	return factory, order, canvas
}

func TestRunMergeConfidenceReusesAutomaticAndFinishedAssessments(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, order, canvas := mergeConfidenceTask(t, r)
	current := checksStartupGitHubPullRequest()
	useGitHub(t, current)
	prs, err := trackedPullRequests(db, order.ID)
	require.NoError(t, err)
	requests, err := models.ListPullRequestChecksStartRequests(db, prs[0].ID)
	require.NoError(t, err)
	require.Len(t, requests, 1)
	automatic, err := ProcessPullRequestChecksStartRequest(t.Context(), IntakeDependencies{}, requests[0].ID)
	require.NoError(t, err)
	require.Len(t, automatic, 1)
	for i, ref := range []string{order.ID.String(), strconv.FormatInt(order.Number, 10), factory.WorkOrderKey(order.Number)} {
		result, err := RunMergeConfidence(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), RunMergeConfidenceRequest{FactoryID: factory.ID.String(), Task: ref})
		require.NoError(t, err)
		require.Len(t, result.Runs, 1)
		assert.Equal(t, automatic[0].ID.String(), result.Runs[0].EventID)
		assert.Equal(t, automatic[0].RunID.String(), result.Runs[0].RunID)
		assert.Equal(t, canvas.ID.String(), result.Runs[0].AutomationID)
		assert.Equal(t, "reused", result.Runs[0].Status)
		assert.Equal(t, prs[0].URL, result.Runs[0].PullRequestURL)
		outcome := models.CanvasRunResultPassed
		if i == 1 {
			outcome = models.CanvasRunResultFailed
		}
		require.NoError(t, db.Model(&models.CanvasRun{}).Where("id = ?", automatic[0].RunID).Updates(map[string]any{"state": models.CanvasRunStateFinished, "result": outcome}).Error)
	}
	assert.EqualValues(t, 1, countCanvasEvents(t, canvas.ID))
	current.Head.SHA = github.Ptr("new-head")
	result, err := RunMergeConfidence(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), RunMergeConfidenceRequest{FactoryID: factory.ID.String(), Task: order.ID.String()})
	require.NoError(t, err)
	require.Len(t, result.Runs, 1)
	assert.Equal(t, "started", result.Runs[0].Status)
	assert.NotEqual(t, automatic[0].RunID.String(), result.Runs[0].RunID)
	var event models.CanvasEvent
	require.NoError(t, db.Where("id = ?", result.Runs[0].EventID).First(&event).Error)
	activity, err := models.FindPullRequestActivityByRunID(db, uuid.MustParse(result.Runs[0].RunID))
	require.NoError(t, err)
	assert.Equal(t, prs[0].ID, activity.PullRequestID)
	payload := models.RootEventSourcePayload(event.Data.Data()).(map[string]any)
	assert.Equal(t, "new-head", payload["pull_request"].(map[string]any)["head"].(map[string]any)["sha"])
	report, err := BuildTaskReport(t.Context(), factory.OrganizationID.String(), factory.ID.String(), order.ID.String(), models.TaskRunHistoryQuery{})
	require.NoError(t, err)
	var reported []string
	for _, run := range report.Runs.Runs {
		reported = append(reported, run.ID.String())
	}
	assert.Contains(t, reported, result.Runs[0].RunID)
	assert.EqualValues(t, 2, countCanvasEvents(t, canvas.ID))
	assert.Len(t, report.Runs.Runs, 2, "implementation does not start")
}

func TestRunMergeConfidencePreconditions(t *testing.T) {
	for _, scenario := range []string{"outside Verify", "archived", "no open PRs", "no configuration", "unpublished", "other repository", "other workspace"} {
		t.Run(scenario, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory, order, canvas := mergeConfidenceTask(t, r)
			switch scenario {
			case "outside Verify":
				require.NoError(t, db.Model(order).Update("state", models.FactoryWorkOrderStateDraft).Error)
			case "archived":
				require.NoError(t, db.Model(order).Updates(map[string]any{"state": models.FactoryWorkOrderStateClosed, "result": models.FactoryWorkOrderResultArchived}).Error)
			case "no open PRs":
				require.NoError(t, db.Model(&models.FactoryPullRequest{}).Where("work_order_id = ?", order.ID).Update("state", models.FactoryPullRequestStateClosed).Error)
			case "no configuration":
				require.NoError(t, canvas.SoftDeleteInTransaction(db))
			case "unpublished":
				version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
				require.NoError(t, err)
				version.Nodes[0].Metadata = nil
				require.NoError(t, db.Save(version).Error)
			case "other repository":
				version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
				require.NoError(t, err)
				version.Nodes[0].Configuration["repository"] = "other/app"
				require.NoError(t, db.Save(version).Error)
			case "other workspace":
				factory = seedHandOffFactory(t, r, db)
			}
			_, err := RunMergeConfidence(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), RunMergeConfidenceRequest{FactoryID: factory.ID.String(), Task: order.ID.String()})
			require.Error(t, err)
			code, _, ok := grpcerrors.HandlerStatus(err)
			require.True(t, ok)
			expected := codes.FailedPrecondition
			if scenario == "other workspace" {
				expected = codes.NotFound
			}
			assert.Equal(t, expected, code)
			assert.Zero(t, countCanvasEvents(t, canvas.ID))
		})
	}
}

func TestRunMergeConfidenceBatchAndDraftRules(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, order, canvas := mergeConfidenceTask(t, r)
	checks := createChecksStartupCanvas(t, r, factory, "acme/app")
	unrelated := createMergeConfidenceStartupCanvas(t, r, factory, "other/app")
	custom := createMergeConfidenceStartupCanvas(t, r, factory, "acme/app")
	require.NoError(t, custom.StampFactoryAppTemplate(db, "renamed-pr-trigger", models.FactoryAppTemplateCustomAutomationID, 1))
	_, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/43"})
	require.NoError(t, err)
	current := checksStartupGitHubPullRequest()
	draft := checksStartupGitHubPullRequest()
	draft.Number, draft.Draft = github.Ptr(43), github.Ptr(true)
	stubFactoryGitHub(t, &fakeFactoryGitHub{pullRequests: map[string]*github.PullRequest{"acme/app#42": current, "acme/app#43": draft}})
	result, err := RunMergeConfidence(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), RunMergeConfidenceRequest{FactoryID: factory.ID.String(), Task: order.ID.String()})
	require.NoError(t, err)
	require.Len(t, result.Runs, 1)
	require.Len(t, result.Skipped, 1)
	assert.Contains(t, result.Skipped[0].PullRequestURL, "/43")
	assert.Contains(t, result.Skipped[0].Reason, "draft")
	assert.Zero(t, countCanvasEvents(t, checks.ID))
	assert.Zero(t, countCanvasEvents(t, unrelated.ID))
	assert.Zero(t, countCanvasEvents(t, custom.ID))
	require.NoError(t, db.Model(&models.CanvasNode{}).Where("workflow_id = ?", canvas.ID).Update("configuration", datatypes.NewJSONType(map[string]any{"repository": "acme/app", "ignoreDrafts": false})).Error)
	result, err = RunMergeConfidence(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), RunMergeConfidenceRequest{FactoryID: factory.ID.String(), Task: order.ID.String()})
	require.NoError(t, err)
	assert.Len(t, result.Runs, 2)
	assert.Empty(t, result.Skipped)
	assert.EqualValues(t, 2, countCanvasEvents(t, canvas.ID))
}

func TestRunMergeConfidenceFetchFailureCreatesNoPartialRuns(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, order, canvas := mergeConfidenceTask(t, r)
	_, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/43"})
	require.NoError(t, err)
	api := &mergeConfidenceFailingGitHub{fakeFactoryGitHub: fakeFactoryGitHub{pullRequest: checksStartupGitHubPullRequest()}}
	stubFactoryGitHub(t, api)
	_, err = RunMergeConfidence(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), RunMergeConfidenceRequest{FactoryID: factory.ID.String(), Task: order.ID.String()})
	require.Error(t, err)
	assert.Equal(t, []int{42, 43}, api.fetched)
	assert.Zero(t, countCanvasEvents(t, canvas.ID))
}

func TestRunMergeConfidenceConcurrentCallsReuseOneAssessment(t *testing.T) {
	r := support.Setup(t)
	factory, order, canvas := mergeConfidenceTask(t, r)
	arrivals, release := make(chan struct{}, 2), make(chan struct{})
	resume := sync.OnceFunc(func() { close(release) })
	stubFactoryGitHub(t, &concurrentImportGitHub{fakeFactoryGitHub: &fakeFactoryGitHub{pullRequest: checksStartupGitHubPullRequest()}, arrivals: arrivals, release: release})
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	defer resume()
	type outcome struct {
		result *RunMergeConfidenceResult
		err    error
	}
	outcomes := make(chan outcome, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			result, err := RunMergeConfidence(ctx, IntakeDependencies{}, r.Organization.ID.String(), RunMergeConfidenceRequest{FactoryID: factory.ID.String(), Task: order.ID.String()})
			outcomes <- outcome{result, err}
		})
	}
	defer wg.Wait()
	for range 2 {
		select {
		case <-arrivals:
		case <-ctx.Done():
			resume()
			t.Fatal("provider lookup did not arrive")
		}
	}
	resume()
	first, second := <-outcomes, <-outcomes
	require.NoError(t, first.err)
	require.NoError(t, second.err)
	require.Len(t, first.result.Runs, 1)
	require.Len(t, second.result.Runs, 1)
	assert.Equal(t, first.result.Runs[0].RunID, second.result.Runs[0].RunID)
	assert.NotEqual(t, first.result.Runs[0].Status, second.result.Runs[0].Status)
	assert.EqualValues(t, 1, countCanvasEvents(t, canvas.ID))
}

func TestRunMergeConfidenceBitbucketReusesAutomaticAndRetargets(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, order, canvas := mergeConfidenceTask(t, r)
	require.NoError(t, db.Model(&models.FactoryPullRequest{}).Where("work_order_id = ?", order.ID).Updates(map[string]any{"provider": "bitbucket", "url": "https://bitbucket.org/acme/app/pull-requests/42"}).Error)
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	require.NoError(t, err)
	version.Nodes[0].Ref.Trigger.Name = "bitbucket.onPullRequest"
	require.NoError(t, db.Save(version).Error)
	require.NoError(t, db.Model(&models.CanvasNode{}).Where("workflow_id = ?", canvas.ID).Update("ref", datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "bitbucket.onPullRequest"}})).Error)
	body := `{"id":42,"state":"OPEN","source":{"branch":{"name":"feature"},"commit":{"hash":"abc123"},"repository":{"full_name":"acme/app"}},"destination":{"branch":{"name":"main"},"repository":{"full_name":"acme/app"}}}`
	responses := []*http.Response{}
	for _, target := range []string{"main", "main", "main", "release", "main"} {
		responses = append(responses, okBitbucketResponse(strings.ReplaceAll(body, `"main"`, strconv.Quote(target))))
	}
	responses = append(responses, okBitbucketResponse(strings.ReplaceAll(body, "abc123", "new-head")))
	stubBitbucketMergeAPI(t, map[string][]*http.Response{"/2.0/repositories/acme/app/pullrequests/42": responses})
	prs, err := trackedPullRequests(db, order.ID)
	require.NoError(t, err)
	requests, err := models.ListPullRequestChecksStartRequests(db, prs[0].ID)
	require.NoError(t, err)
	require.Len(t, requests, 1)
	automatic, err := ProcessPullRequestChecksStartRequest(t.Context(), IntakeDependencies{}, requests[0].ID)
	require.NoError(t, err)
	require.Len(t, automatic, 1)
	for _, expected := range []string{"reused", "reused", "started", "started", "started"} {
		result, err := RunMergeConfidence(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), RunMergeConfidenceRequest{FactoryID: factory.ID.String(), Task: order.ID.String()})
		require.NoError(t, err)
		require.Len(t, result.Runs, 1)
		assert.Equal(t, expected, result.Runs[0].Status)
		if expected == "reused" {
			assert.Equal(t, automatic[0].RunID.String(), result.Runs[0].RunID)
		}
	}
	assert.EqualValues(t, 4, countCanvasEvents(t, canvas.ID))
}

func TestRunMergeConfidenceRechecksEligibilityAfterProviderFetch(t *testing.T) {
	for _, scenario := range []string{"task leaves Verify", "PR closes", "PR changes", "automation disabled"} {
		t.Run(scenario, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory, order, canvas := mergeConfidenceTask(t, r)
			arrivals, release := make(chan struct{}, 1), make(chan struct{})
			resume := sync.OnceFunc(func() { close(release) })
			stubFactoryGitHub(t, &concurrentImportGitHub{fakeFactoryGitHub: &fakeFactoryGitHub{pullRequest: checksStartupGitHubPullRequest()}, arrivals: arrivals, release: release})
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			finished := make(chan error, 1)
			go func() {
				_, err := RunMergeConfidence(ctx, IntakeDependencies{}, r.Organization.ID.String(), RunMergeConfidenceRequest{FactoryID: factory.ID.String(), Task: order.ID.String()})
				finished <- err
			}()
			t.Cleanup(func() { cancel(); resume() })
			select {
			case <-arrivals:
			case <-ctx.Done():
				t.Fatal("provider lookup did not arrive")
			}
			switch scenario {
			case "task leaves Verify":
				require.NoError(t, db.Model(order).Update("state", models.FactoryWorkOrderStateClosed).Error)
			case "PR closes":
				require.NoError(t, db.Model(&models.FactoryPullRequest{}).Where("work_order_id = ?", order.ID).Update("state", models.FactoryPullRequestStateClosed).Error)
			case "PR changes":
				require.NoError(t, db.Model(&models.FactoryPullRequest{}).Where("work_order_id = ?", order.ID).Update("number", 99).Error)
			case "automation disabled":
				require.NoError(t, db.Model(&models.CanvasNode{}).Where("workflow_id = ?", canvas.ID).Update("state", models.CanvasNodeStateError).Error)
			}
			resume()
			require.Error(t, <-finished)
			assert.Zero(t, countCanvasEvents(t, canvas.ID))
		})
	}
}

func TestRunMergeConfidenceRepositoryMatching(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryMultiRepoTasks))
	factory, order, canvas := mergeConfidenceTask(t, r)
	for _, repository := range []string{"acme/web", "acme/docs"} {
		_, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{Repository: repository, Access: models.FactoryContextRepoAccessWrite})
		require.NoError(t, err)
		_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/" + repository + "/pull/7"})
		require.NoError(t, err)
	}
	webCanvas := createMergeConfidenceStartupCanvas(t, r, factory, "ACME/Web")
	api := checksStartupGitHubPullRequest()
	web := checksStartupGitHubPullRequest()
	web.Number = github.Ptr(7)
	web.Head.Repo.FullName, web.Base.Repo.FullName = github.Ptr("acme/web"), github.Ptr("acme/web")
	stubFactoryGitHub(t, &fakeFactoryGitHub{pullRequests: map[string]*github.PullRequest{"acme/app#42": api, "acme/web#7": web}})
	result, err := RunMergeConfidence(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), RunMergeConfidenceRequest{FactoryID: factory.ID.String(), Task: order.ID.String()})
	require.NoError(t, err)
	require.Len(t, result.Runs, 2)
	require.Len(t, result.Skipped, 1)
	assert.Equal(t, "https://github.com/acme/docs/pull/7", result.Skipped[0].PullRequestURL)
	assert.Contains(t, result.Skipped[0].Reason, "automation")
	assert.ElementsMatch(t, []string{canvas.ID.String(), webCanvas.ID.String()}, []string{result.Runs[0].AutomationID, result.Runs[1].AutomationID})
}

func TestRunMergeConfidenceRollsBackAllStarts(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, order, canvas := mergeConfidenceTask(t, r)
	_, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/43"})
	require.NoError(t, err)
	second := checksStartupGitHubPullRequest()
	second.Number = github.Ptr(43)
	stubFactoryGitHub(t, &fakeFactoryGitHub{pullRequests: map[string]*github.PullRequest{"acme/app#42": checksStartupGitHubPullRequest(), "acme/app#43": second}})
	const callback = "test:merge_confidence_batch_failure"
	inserts := 0
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "workflow_events" {
			inserts++
			if inserts == 2 {
				tx.AddError(errors.New("event persistence failed"))
			}
		}
	}))
	t.Cleanup(func() { require.NoError(t, db.Callback().Create().Remove(callback)) })
	_, err = RunMergeConfidence(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), RunMergeConfidenceRequest{FactoryID: factory.ID.String(), Task: order.ID.String()})
	require.Error(t, err)
	assert.Equal(t, 2, inserts)
	assert.Zero(t, countCanvasEvents(t, canvas.ID))
	runs, err := order.ListRuns(db)
	require.NoError(t, err)
	assert.Empty(t, runs, "failed batches leave no runs or task activity")
}

func TestRunMergeConfidenceSkipsDraftWithoutRevision(t *testing.T) {
	for _, provider := range []string{"github", "bitbucket"} {
		t.Run(provider, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory, order, canvas := mergeConfidenceTask(t, r)
			if provider == "github" {
				draft := checksStartupGitHubPullRequest()
				draft.Draft, draft.Head = github.Ptr(true), nil
				useGitHub(t, draft)
			} else {
				require.NoError(t, db.Model(&models.FactoryPullRequest{}).Where("work_order_id = ?", order.ID).Update("provider", provider).Error)
				version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
				require.NoError(t, err)
				version.Nodes[0].Ref.Trigger.Name = "bitbucket.onPullRequest"
				require.NoError(t, db.Save(version).Error)
				require.NoError(t, db.Model(&models.CanvasNode{}).Where("workflow_id = ?", canvas.ID).Update("ref", datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "bitbucket.onPullRequest"}})).Error)
				stubBitbucketMergeAPI(t, map[string][]*http.Response{"/2.0/repositories/acme/app/pullrequests/42": {okBitbucketResponse(`{"id":42,"state":"OPEN","draft":true,"source":{"branch":{"name":"feature"},"repository":{"full_name":"acme/app"}},"destination":{"branch":{"name":"main"},"repository":{"full_name":"acme/app"}}}`)}})
			}
			result, err := RunMergeConfidence(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), RunMergeConfidenceRequest{FactoryID: factory.ID.String(), Task: order.ID.String()})
			require.NoError(t, err)
			assert.Empty(t, result.Runs)
			require.Len(t, result.Skipped, 1)
			assert.Contains(t, result.Skipped[0].Reason, "draft")
			assert.Zero(t, countCanvasEvents(t, canvas.ID))
		})
	}
}

type mergeConfidenceFailingGitHub struct {
	fakeFactoryGitHub
	fetched []int
}

func (f *mergeConfidenceFailingGitHub) GetPullRequest(ctx context.Context, repository string, number int) (*github.PullRequest, *github.Response, error) {
	f.fetched = append(f.fetched, number)
	if number == 43 {
		return nil, nil, errors.New("provider unavailable")
	}
	return f.fakeFactoryGitHub.GetPullRequest(ctx, repository, number)
}

func TestRunMergeConfidenceReusesWorkspaceScopedAssessment(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, order, canvas := mergeConfidenceTask(t, r)
	configuration := map[string]any{"repository": models.WorkspaceRepositoriesExpression, "ignoreDrafts": true, "onlyFactoryPullRequests": true}
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	require.NoError(t, err)
	version.Nodes[0].Configuration = configuration
	require.NoError(t, db.Save(version).Error)
	require.NoError(t, db.Model(&models.CanvasNode{}).Where("workflow_id = ?", canvas.ID).Update("configuration", datatypes.NewJSONType(configuration)).Error)
	useGitHub(t, checksStartupGitHubPullRequest())
	prs, err := trackedPullRequests(db, order.ID)
	require.NoError(t, err)
	requests, err := models.ListPullRequestChecksStartRequests(db, prs[0].ID)
	require.NoError(t, err)
	require.Len(t, requests, 1)
	automatic, err := ProcessPullRequestChecksStartRequest(t.Context(), IntakeDependencies{}, requests[0].ID)
	require.NoError(t, err)
	require.Len(t, automatic, 1)
	for range 2 {
		result, err := RunMergeConfidence(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), RunMergeConfidenceRequest{FactoryID: factory.ID.String(), Task: order.ID.String()})
		require.NoError(t, err)
		require.Len(t, result.Runs, 1)
		assert.Equal(t, "reused", result.Runs[0].Status)
		assert.Equal(t, automatic[0].RunID.String(), result.Runs[0].RunID)
	}
	assert.EqualValues(t, 1, countCanvasEvents(t, canvas.ID))
}

func TestRunMergeConfidenceReusesWithoutNewRunnerAdmission(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, order, canvas := mergeConfidenceTask(t, r)
	current := checksStartupGitHubPullRequest()
	useGitHub(t, current)
	request := RunMergeConfidenceRequest{FactoryID: factory.ID.String(), Task: order.ID.String()}
	first, err := RunMergeConfidence(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), request)
	require.NoError(t, err)
	require.Len(t, first.Runs, 1)
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("id = ?", first.Runs[0].RunID).Updates(map[string]any{"state": models.CanvasRunStateFinished, "result": models.CanvasRunResultFailed}).Error)
	// Saved assessments remain readable when current settings cannot admit a new run.
	subscription := true
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
	reused, err := RunMergeConfidence(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), request)
	require.NoError(t, err)
	require.Len(t, reused.Runs, 1)
	assert.Equal(t, "reused", reused.Runs[0].Status)
	assert.Equal(t, first.Runs[0].RunID, reused.Runs[0].RunID)
	current.Head.SHA = github.Ptr("another-head")
	_, err = RunMergeConfidence(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), request)
	require.Error(t, err)
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
	assert.EqualValues(t, 1, countCanvasEvents(t, canvas.ID))
}
