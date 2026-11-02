package factories

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// This boundary covers manual import ownership and atomic Verify creation;
// MCP handoff tests retain ownership of checks and stage selection behavior.
func TestImportFactoryPullRequest(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(ctx)
	factory := seedHandOffFactory(t, r, db)
	require.NoError(t, factory.UpdatePlanning(db, models.FactoryPlanning{Enabled: true, Clarity: true, Confidence: true}))
	backlog := createOnWorkOrderCanvas(t, r, factory.ID)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestImport))
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "implement", "start")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry}})
	require.NoError(t, err)
	pr := forkHandOffPullRequest("acme/app", "contributor/app", 42, "feature")
	pr.Body = github.Ptr("Current description")
	pr.Head.SHA = github.Ptr("0123456789abcdef0123456789abcdef01234567")
	pr.Base.Ref = github.Ptr("main")
	pr.CreatedAt = &github.Timestamp{Time: time.Now().Add(-20 * time.Minute)}
	checks := createChecksStartupCanvas(t, r, factory, "acme/app")
	mergeConfidence := createMergeConfidenceStartupCanvas(t, r, factory, "acme/app")
	useGitHub(t, pr)
	req := &pb.ImportFactoryPullRequestRequest{FactoryId: factory.ID.String(), LineId: line.ID.String(), PullRequestUrl: pr.GetHTMLURL()}
	result, err := ImportFactoryPullRequest(ctx, IntakeDependencies{}, r.Organization.ID.String(), req)
	require.NoError(t, err)
	require.True(t, result.GetCreated())
	assert.Equal(t, "Ready", result.Order.GetTitle())
	assert.Equal(t, "Current description", result.Order.GetDescription())
	assert.Zero(t, countCanvasEvents(t, backlog.ID), "PR imports must not start refinement")
	var runs int64
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("workflow_id = ?", app.ID).Count(&runs).Error)
	assert.Zero(t, runs, "PR imports must not start implementation")
	assert.Nil(t, result.Order.GetMcpClient())
	require.Len(t, result.Order.GetLineDispatches(), 1)
	assert.Equal(t, pb.WorkOrderLineDispatch_RESULT_PASSED, result.Order.GetLineDispatches()[0].GetResult())
	order, err := factory.FindWorkOrder(db, uuid.MustParse(result.Order.GetId()))
	require.NoError(t, err)
	artifacts, err := order.ListArtifacts(db)
	require.NoError(t, err)
	assert.True(t, hasBranchArtifact(artifacts, "feature", "contributor/app"))
	_, err = order.FindArtifactByKey(db, models.PlanningSpecArtifactKey+":"+order.ID.String())
	require.Error(t, err)
	pr.Title = github.Ptr("Changed upstream")
	req.PullRequestUrl = "https://github.com/ACME/App/pull/42?diff=split#files"
	again, err := ImportFactoryPullRequest(ctx, IntakeDependencies{}, r.Organization.ID.String(), req)
	require.NoError(t, err)
	assert.False(t, again.GetCreated())
	assert.Equal(t, result.Order.GetId(), again.Order.GetId())
	assert.Equal(t, "Ready", again.Order.GetTitle())
	assert.EqualValues(t, 1, countCanvasEvents(t, checks.ID), "only a new import starts checks")
	assert.EqualValues(t, 1, countCanvasEvents(t, mergeConfidence.ID), "only a new import starts merge confidence")
	owners, err := order.ListAssignees(db)
	require.NoError(t, err)
	require.Len(t, owners, 1)
	assert.Equal(t, r.User, owners[0].UserID)
	candidates, err := SearchFactoryPullRequestImportItems(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.SearchFactoryPullRequestImportItemsRequest{FactoryId: factory.ID.String(), Query: "42"})
	require.NoError(t, err)
	require.Len(t, candidates.Items, 1)
	assert.Equal(t, order.ID.String(), candidates.Items[0].GetExistingTask().GetId())

	for _, scenario := range []struct{ name, url, state, branch, line string }{
		{"closed", "https://github.com/acme/app/pull/43", "closed", "feature", line.ID.String()},
		{"wrong repository", "https://github.com/other/app/pull/43", "open", "feature", line.ID.String()},
		{"wrong provider", "https://bitbucket.org/acme/app/pull-requests/43", "open", "feature", line.ID.String()},
		{"missing branch", "https://github.com/acme/app/pull/43", "open", "", line.ID.String()},
		{"invalid line", "https://github.com/acme/app/pull/43", "open", "feature", uuid.NewString()},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			current := openHandOffPullRequest("acme/app", 43, scenario.branch)
			current.State = github.Ptr(scenario.state)
			useGitHub(t, current)
			_, err := ImportFactoryPullRequest(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.ImportFactoryPullRequestRequest{FactoryId: factory.ID.String(), LineId: scenario.line, PullRequestUrl: scenario.url})
			require.Error(t, err)
			orders, err := factory.ListWorkOrders(db, models.ListFactoryWorkOrdersFilters{})
			require.NoError(t, err)
			assert.Len(t, orders, 1)
		})
	}

	// An archived import stays in history while a fresh task takes over Verify.
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateClosed, Result: models.FactoryWorkOrderResultArchived, Actor: &r.User})
	require.NoError(t, err)
	useGitHub(t, pr)
	available, err := SearchFactoryPullRequestImportItems(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.SearchFactoryPullRequestImportItemsRequest{FactoryId: factory.ID.String(), Query: "42"})
	require.NoError(t, err)
	require.Len(t, available.Items, 1)
	assert.Nil(t, available.Items[0].ExistingTask, "an archived task must not block import")
	pr.State = github.Ptr("closed")
	_, err = ImportFactoryPullRequest(ctx, IntakeDependencies{}, r.Organization.ID.String(), req)
	require.Error(t, err)
	preserved, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{Provider: "github", Repository: "acme/app", Number: 42})
	require.NoError(t, err)
	assert.Equal(t, order.ID, preserved.WorkOrderID, "failed reimport must preserve the old tracking association")
	pr.State = github.Ptr("open")
	recreated, err := ImportFactoryPullRequest(ctx, IntakeDependencies{}, r.Organization.ID.String(), req)
	require.NoError(t, err)
	require.True(t, recreated.Created)
	assert.NotEqual(t, order.ID.String(), recreated.Order.GetId())
	archived, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.True(t, archived.IsArchived())
	tracked, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{Provider: "github", Repository: "acme/app", Number: 42})
	require.NoError(t, err)
	assert.Equal(t, recreated.Order.GetId(), tracked.WorkOrderID.String())
	history, err := models.ListPullRequestsByWorkOrderIDs(db, []uuid.UUID{order.ID})
	require.NoError(t, err)
	assert.Len(t, history[order.ID], 1, "the archived task retains its pull request history")
	assert.EqualValues(t, 2, countCanvasEvents(t, checks.ID))
	assert.EqualValues(t, 2, countCanvasEvents(t, mergeConfidence.ID), "the new task needs checks even at the same PR revision")
}

// Search tests inspect provider requests as well as results: a local filter
// cannot protect repository isolation when the provider query is unscoped.
type importSearchGitHub struct {
	*fakeFactoryGitHub
	t            *testing.T
	query        string
	pullRequests []*github.PullRequest
}

func (f *importSearchGitHub) ListPullRequests(_ context.Context, repository string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	assert.Equal(f.t, "acme/app", repository)
	assert.Equal(f.t, "open", opts.State)
	assert.Equal(f.t, 5, opts.PerPage)
	if f.pullRequests != nil {
		return f.pullRequests, nil, nil
	}
	return []*github.PullRequest{f.pullRequest, openHandOffPullRequest("other/app", 43, "branch")}, nil, nil
}
func (f *importSearchGitHub) SearchIssues(_ context.Context, query string, opts *github.SearchOptions) (*github.IssuesSearchResult, *github.Response, error) {
	assert.Equal(f.t, f.query, query)
	assert.Equal(f.t, 50, opts.PerPage)
	return &github.IssuesSearchResult{Issues: []*github.Issue{
		{Number: github.Ptr(42), Title: github.Ptr("Ready"), State: github.Ptr("open"), HTMLURL: github.Ptr("https://github.com/acme/app/pull/42"), Draft: github.Ptr(true), PullRequestLinks: &github.PullRequestLinks{}},
		{Number: github.Ptr(43), Title: github.Ptr("Wrong repository"), State: github.Ptr("open"), HTMLURL: github.Ptr("https://github.com/other/app/pull/43"), PullRequestLinks: &github.PullRequestLinks{}},
	}}, nil, nil
}

func TestSearchFactoryPullRequestImportItemsGitHub(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestImport))
	pr := openHandOffPullRequest("acme/app", 42, "branch")
	pr.Draft = github.Ptr(true)
	fake := &importSearchGitHub{fakeFactoryGitHub: &fakeFactoryGitHub{pullRequest: pr}, t: t, query: `repo:acme/app is:pr is:open in:title "fix \" repo:other/app"`}
	original := newFactoryGitHubAPI
	newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) { return fake, nil }
	t.Cleanup(func() { newFactoryGitHubAPI = original })
	for _, query := range []string{"", "42", "#42", pr.GetHTMLURL(), `fix " repo:other/app`} {
		t.Run(query, func(t *testing.T) {
			limit := int32(0)
			if strings.HasPrefix(query, "fix") {
				limit = 100
			}
			result, err := SearchFactoryPullRequestImportItems(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.SearchFactoryPullRequestImportItemsRequest{FactoryId: factory.ID.String(), Query: query, Limit: limit})
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.True(t, result.Items[0].Draft)
			assert.Equal(t, pr.GetHTMLURL(), result.Items[0].Url)
		})
	}
	_, err := SearchFactoryPullRequestImportItems(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.SearchFactoryPullRequestImportItemsRequest{FactoryId: factory.ID.String(), Query: "https://github.com/other/app/pull/42"})
	require.Error(t, err)
}

func TestSearchFactoryPullRequestImportItemsBatchesExistingTasks(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestImport))
	orders := make([]*models.FactoryWorkOrder, 5)
	prs := make([]*github.PullRequest, len(orders))
	for i := range orders {
		order, err := factory.CreateWorkOrder(db, fmt.Sprintf("Imported task %d", i), "", &r.User, nil, nil)
		require.NoError(t, err)
		orders[i] = order
		prs[i] = openHandOffPullRequest("acme/app", 100+i, "branch")
		_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: prs[i].GetHTMLURL()})
		require.NoError(t, err)
	}
	// Provider identities must match even when repository casing differs.
	require.NoError(t, db.Model(&models.FactoryPullRequest{}).Where("factory_id = ?", factory.ID).Update("repository", "ACME/App").Error)
	fake := &importSearchGitHub{fakeFactoryGitHub: &fakeFactoryGitHub{}, t: t}
	original := newFactoryGitHubAPI
	newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) { return fake, nil }
	t.Cleanup(func() { newFactoryGitHubAPI = original })

	// Count only SQL in this request, including queries in serialization goroutines.
	type queryCountKey struct{}
	const callback = "test:pull_request_import_query_count"
	require.NoError(t, db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if count, ok := tx.Statement.Context.Value(queryCountKey{}).(*atomic.Int64); ok {
			count.Add(1)
		}
	}))
	t.Cleanup(func() { require.NoError(t, db.Callback().Query().Remove(callback)) })
	search := func(prs []*github.PullRequest) (*pb.SearchFactoryPullRequestImportItemsResponse, int64) {
		t.Helper()
		fake.pullRequests = prs
		var count atomic.Int64
		ctx := context.WithValue(t.Context(), queryCountKey{}, &count)
		result, err := SearchFactoryPullRequestImportItems(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.SearchFactoryPullRequestImportItemsRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		require.Len(t, result.Items, len(prs))
		return result, count.Load()
	}
	_, singleQueries := search(prs[:1])
	result, batchQueries := search(prs)
	t.Logf("Database queries: one task = %d, five tasks = %d", singleQueries, batchQueries)
	require.Positive(t, singleQueries)
	assert.Equal(t, singleQueries, batchQueries, "database queries must not grow with matching imported tasks")
	for i, item := range result.Items {
		assert.Equal(t, orders[i].ID.String(), item.GetExistingTask().GetId())
		assert.Equal(t, factory.WorkOrderKey(orders[i].Number), item.GetExistingTask().GetKey())
	}
	otherFactory := seedHandOffFactory(t, r, db)
	for i, scenario := range []struct {
		name     string
		factory  *models.Factory
		provider string
		url      string
	}{
		{"another factory", otherFactory, models.ProviderGitHub, "https://github.com/acme/app/pull/200"},
		{"another repository", factory, models.ProviderGitHub, "https://github.com/other/app/pull/201"},
		{"another provider", factory, models.ProviderBitbucket, "https://bitbucket.org/acme/app/pull-requests/202"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			order, err := scenario.factory.CreateWorkOrder(db, "Unrelated task", "", &r.User, nil, nil)
			require.NoError(t, err)
			_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{Provider: scenario.provider, URL: scenario.url})
			require.NoError(t, err)
			result, _ := search([]*github.PullRequest{openHandOffPullRequest("acme/app", 200+i, "branch")})
			assert.Nil(t, result.Items[0].ExistingTask)
		})
	}
}

func TestImportFactoryPullRequestConcurrent(t *testing.T) {
	for _, archived := range []bool{false, true} {
		t.Run(fmt.Sprintf("archived predecessor %t", archived), func(t *testing.T) {
			r := support.Setup(t)
			ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
			db := database.DB(ctx)
			factory := seedHandOffFactory(t, r, db)
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestImport))
			app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "implement", "start")
			line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry}})
			require.NoError(t, err)
			if archived {
				previous, err := factory.CreateWorkOrder(db, "Previous import", "", &r.User, nil, nil)
				require.NoError(t, err)
				_, err = previous.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
				require.NoError(t, err)
				_, err = previous.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateClosed, Result: models.FactoryWorkOrderResultArchived, Actor: &r.User})
				require.NoError(t, err)
			}
			arrivals := make(chan struct{}, 2)
			release := make(chan struct{})
			fake := &concurrentImportGitHub{fakeFactoryGitHub: &fakeFactoryGitHub{pullRequest: openHandOffPullRequest("acme/app", 42, "branch")}, arrivals: arrivals, release: release}
			original := newFactoryGitHubAPI
			newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) { return fake, nil }
			t.Cleanup(func() { newFactoryGitHubAPI = original })
			req := &pb.ImportFactoryPullRequestRequest{FactoryId: factory.ID.String(), LineId: line.ID.String(), PullRequestUrl: "https://github.com/acme/app/pull/42"}
			type outcome struct {
				result *pb.ImportFactoryPullRequestResponse
				err    error
			}
			outcomes := make(chan outcome, 2)
			for range 2 {
				go func() {
					result, err := ImportFactoryPullRequest(ctx, IntakeDependencies{}, r.Organization.ID.String(), req)
					outcomes <- outcome{result, err}
				}()
			}
			for range 2 {
				select {
				case <-arrivals:
				case <-time.After(10 * time.Second):
					t.Fatal("concurrent imports did not reach provider lookup")
				}
			}
			close(release)
			first, second := <-outcomes, <-outcomes
			require.NoError(t, first.err)
			require.NoError(t, second.err)
			assert.Equal(t, first.result.Order.GetId(), second.result.Order.GetId())
			assert.NotEqual(t, first.result.Created, second.result.Created)
			orders, err := factory.ListWorkOrders(db, models.ListFactoryWorkOrdersFilters{})
			require.NoError(t, err)
			want := 1
			if archived {
				want = 2
			}
			assert.Len(t, orders, want)
		})
	}
}

type concurrentImportGitHub struct {
	*fakeFactoryGitHub
	arrivals chan struct{}
	release  chan struct{}
}

func (f *concurrentImportGitHub) GetPullRequest(ctx context.Context, repository string, number int) (*github.PullRequest, *github.Response, error) {
	f.arrivals <- struct{}{}
	select {
	case <-f.release:
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
	return f.fakeFactoryGitHub.GetPullRequest(ctx, repository, number)
}

func TestPullRequestImportRequiresFlagAndConnections(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(ctx)
	factory := seedHandOffFactory(t, r, db)
	search := &pb.SearchFactoryPullRequestImportItemsRequest{FactoryId: factory.ID.String()}
	request := &pb.ImportFactoryPullRequestRequest{FactoryId: factory.ID.String(), LineId: uuid.NewString(), PullRequestUrl: "https://github.com/acme/app/pull/42"}
	_, err := SearchFactoryPullRequestImportItems(ctx, IntakeDependencies{}, r.Organization.ID.String(), search)
	require.Error(t, err)
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, code)
	_, err = ImportFactoryPullRequest(ctx, IntakeDependencies{}, r.Organization.ID.String(), request)
	require.Error(t, err)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestImport))
	_, err = ImportFactoryPullRequest(ctx, IntakeDependencies{}, r.Organization.ID.String(), request)
	require.ErrorIs(t, err, errFactoryGitHubNotConnected)
	useGitHub(t, openHandOffPullRequest("acme/app", 42, "branch"))
	subscription := true
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
	_, err = ImportFactoryPullRequest(ctx, IntakeDependencies{}, r.Organization.ID.String(), request)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	require.Equal(t, "CODEX_CONNECTION_REQUIRED", status.Convert(err).Details()[0].(*errdetails.ErrorInfo).Reason)
	orders, err := factory.ListWorkOrders(db, models.ListFactoryWorkOrdersFilters{})
	require.NoError(t, err)
	assert.Empty(t, orders)
}

func TestPullRequestImportBitbucket(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(ctx)
	factory := seedBitbucketHandOffFactory(t, r, db)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestImport))
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "implement", "start")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry}})
	require.NoError(t, err)
	body := `{"id":42,"title":"Current title","description":"Current body","state":"OPEN","draft":true,"links":{"html":{"href":"https://bitbucket.org/acme/app/pull-requests/42"}},"source":{"branch":{"name":"feature"},"repository":{"full_name":"contributor/app"}},"destination":{"repository":{"full_name":"acme/app"}}}`
	for _, query := range []string{"", `fix " OR state="MERGED`, "42", "#42", "https://bitbucket.org/acme/app/pull-requests/42"} {
		t.Run(query, func(t *testing.T) {
			httpCtx := stubBitbucketMergeAPI(t, map[string][]*http.Response{
				"/2.0/repositories/acme/app/pullrequests":    {okBitbucketResponse(`{"values":[` + body + `]}`)},
				"/2.0/repositories/acme/app/pullrequests/42": {okBitbucketResponse(body)},
			})
			result, err := SearchFactoryPullRequestImportItems(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.SearchFactoryPullRequestImportItemsRequest{FactoryId: factory.ID.String(), Query: query})
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			assert.True(t, result.Items[0].Draft)
			assert.Equal(t, "Current body", result.Items[0].Description)
			if strings.HasPrefix(query, "fix") {
				require.Len(t, httpCtx.Requests, 1)
				assert.Equal(t, `title ~ "fix \" OR state=\"MERGED"`, httpCtx.Requests[0].URL.Query().Get("q"))
				assert.Equal(t, "OPEN", httpCtx.Requests[0].URL.Query().Get("state"))
			}
		})
	}
	stubBitbucketMergeAPI(t, map[string][]*http.Response{"/2.0/repositories/acme/app/pullrequests/42": {okBitbucketResponse(body)}})
	result, err := ImportFactoryPullRequest(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.ImportFactoryPullRequestRequest{FactoryId: factory.ID.String(), LineId: line.ID.String(), PullRequestUrl: "https://bitbucket.org/acme/app/pull-requests/42"})
	require.NoError(t, err)
	assert.True(t, result.Created)
	assert.Equal(t, "Current title", result.Order.GetTitle())
	assert.Equal(t, "Current body", result.Order.GetDescription())
	order, err := factory.FindWorkOrder(db, uuid.MustParse(result.Order.GetId()))
	require.NoError(t, err)
	artifacts, err := order.ListArtifacts(db)
	require.NoError(t, err)
	assert.True(t, hasBranchArtifact(artifacts, "feature", "contributor/app"))
}
