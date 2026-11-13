package workers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	canvasactions "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/pulls"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
	"github.com/superplanehq/superplane/test/support/impl"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type reviewCheckIntegration struct {
	core.IntegrationContext
	store core.PublicationStore
}

func (c reviewCheckIntegration) Publications() core.PublicationStore { return c.store }

type reviewCheckHTTP struct {
	t            *testing.T
	mu           sync.Mutex
	head         string
	closed       bool
	reviews      []map[string]any
	failReviews  bool
	failWrite    bool
	failPR       bool
	requestCount int
	writes       []struct {
		SHA    string
		Status github.RepoStatus
	}
	onRead  func()
	onWrite func()
}

func (h *reviewCheckHTTP) Do(request *http.Request) (*http.Response, error) {
	if request.Method == http.MethodGet && strings.HasSuffix(request.URL.Path, "/pulls/42") && h.onRead != nil {
		h.onRead()
	}
	if request.Method == http.MethodPost && strings.Contains(request.URL.Path, "/statuses/") && h.onWrite != nil {
		h.onWrite()
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	path := request.URL.Path
	if request.Method == http.MethodPost && strings.Contains(path, "/statuses/") {
		if h.failWrite {
			return mocks.GitHubResponse(503, `{"message":"unavailable"}`), nil
		}
		var status github.RepoStatus
		require.NoError(h.t, json.NewDecoder(request.Body).Decode(&status))
		h.writes = append(h.writes, struct {
			SHA    string
			Status github.RepoStatus
		}{SHA: path[strings.LastIndex(path, "/")+1:], Status: status})
		return mocks.GitHubResponse(201, `{}`), nil
	}
	if request.Method == http.MethodPost && strings.HasSuffix(path, "/requested_reviewers") {
		h.requestCount++
		return mocks.GitHubResponse(201, `{}`), nil
	}
	if strings.HasSuffix(path, "/reviews") {
		if h.failReviews {
			return mocks.GitHubResponse(503, `{"message":"unavailable"}`), nil
		}
		// Split on two pages even for small histories to exercise pagination.
		reviews := h.reviews
		if request.URL.Query().Get("page") == "2" {
			reviews = reviews[min(1, len(reviews)):]
		} else {
			reviews = reviews[:min(1, len(reviews))]
		}
		body, _ := json.Marshal(reviews)
		if reviews == nil {
			body = []byte("[]")
		}
		response := mocks.GitHubResponse(200, string(body))
		if len(h.reviews) > 1 && request.URL.Query().Get("page") != "2" {
			response.Header.Set("Link", `<https://api.github.com/repos/testhq/hello/pulls/42/reviews?page=2>; rel="next"`)
		}
		return response, nil
	}
	if strings.HasSuffix(path, "/collaborators") {
		return mocks.GitHubResponse(200, `[{"id":2,"login":"first","type":"User","permissions":{"push":true}},{"id":3,"login":"second","type":"User","permissions":{"push":true}}]`), nil
	}
	if strings.HasSuffix(path, "/permission") {
		return mocks.GitHubResponse(200, `{"permission":"write"}`), nil
	}
	if strings.HasSuffix(path, "/pulls/42") {
		if h.failPR {
			return mocks.GitHubResponse(503, `{"message":"unavailable"}`), nil
		}
		state := "open"
		if h.closed {
			state = "closed"
		}
		return mocks.GitHubResponse(200, fmt.Sprintf(`{"state":%q,"user":{"id":1,"login":"author"},"head":{"sha":%q},"requested_reviewers":[]}`, state, h.head)), nil
	}
	return nil, fmt.Errorf("unexpected GitHub request: %s %s", request.Method, request.URL)
}

type reviewCheckFixture struct {
	activationNode string
	t              *testing.T
	r              *support.ResourceRegistry
	db             *gorm.DB
	canvas         *models.Canvas
	nodes          []models.CanvasNode
	root           *models.CanvasEvent
	installation   *models.Integration
	integration    reviewCheckIntegration
	http           *reviewCheckHTTP
	ctx            core.ExecutionContext
}

func newReviewCheckFixture(t *testing.T) *reviewCheckFixture {
	t.Helper()
	r := support.Setup(t)
	t.Cleanup(r.Close)
	db := database.DB(t.Context())
	installation, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "github", support.RandomName("github"), nil)
	require.NoError(t, err)
	canvas, nodes := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: "trigger", Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onBroadcast"}})},
		{NodeID: "ensure", Type: models.NodeTypeComponent, AppInstallationID: &installation.ID, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "github.ensurePullRequestReviewers"}})},
		{NodeID: "check", Type: models.NodeTypeComponent, AppInstallationID: &installation.ID, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "github.setPullRequestReviewCheck"}})},
	}, nil)
	for i := 1; i < len(nodes); i++ {
		nodes[i].AppInstallationID = &installation.ID
		require.NoError(t, db.Model(&nodes[i]).Update("app_installation_id", installation.ID).Error)
	}
	store := contexts.NewIntegrationContext(db, nil, installation, r.Encryptor, r.Registry, nil).Publications()
	f := &reviewCheckFixture{t: t, r: r, db: db, canvas: canvas, nodes: nodes, installation: installation, http: &reviewCheckHTTP{t: t, head: "abc"}, integration: reviewCheckIntegration{IntegrationContext: mocks.IntegrationContextForNewSetupFlow(), store: store}}
	f.root = support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	f.attempt(f.root)
	return f
}
func (f *reviewCheckFixture) attempt(root *models.CanvasEvent) {
	f.root = root
	execution := support.CreateCanvasNodeExecution(f.t, f.canvas.ID, "check", root.ID, root.ID)
	f.ctx = f.executionContext(execution, &f.nodes[2])
}

func (f *reviewCheckFixture) finishAction() {
	var request models.CanvasNodeRequest
	require.NoError(f.t, f.db.Where("execution_id = ? AND state = ?", f.ctx.ID, models.NodeExecutionRequestStatePending).First(&request).Error)
	worker := NewNodeRequestExecutor(f.r.Encryptor, f.r.Registry, "", nil)
	require.NoError(f.t, worker.RunNow(request))
	var execution models.CanvasNodeExecution
	require.NoError(f.t, f.db.First(&execution, "id = ?", f.ctx.ID).Error)
	require.Equal(f.t, models.CanvasNodeExecutionResultPassed, execution.Result)
}

func (f *reviewCheckFixture) processDue() error {
	provider, err := f.r.Registry.GetIntegration("github")
	require.NoError(f.t, err)
	f.r.Registry.Integrations["github"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		Hooks: provider.Hooks(),
		HandleHook: func(ctx core.IntegrationHookContext) error {
			ctx.HTTP, ctx.Integration = f.http, f.integration
			return provider.HandleHook(ctx)
		},
	})
	defer func() { f.r.Registry.Integrations["github"] = provider }()
	requests := pendingActionRequests(f.t, f.installation.ID, pulls.ReviewCheckHook)
	require.Len(f.t, requests, 1)
	require.NoError(f.t, f.db.Model(&requests[0]).Update("run_at", time.Now().Add(-time.Second)).Error)
	worker := NewIntegrationRequestWorker(f.r.Encryptor, f.r.Registry, nil, "", "")
	return worker.LockAndProcessRequest(requests[0])
}
func (f *reviewCheckFixture) executionContext(execution *models.CanvasNodeExecution, node *models.CanvasNode) core.ExecutionContext {
	execution.State = models.CanvasNodeExecutionStateStarted
	require.NoError(f.t, f.db.Save(execution).Error)
	return core.ExecutionContext{ID: execution.ID, RunID: execution.RunID, WorkflowID: f.canvas.ID.String(), OrganizationID: f.r.Organization.ID.String(), NodeID: node.NodeID, Integration: f.integration, HTTP: f.http,
		Publications: contexts.NewPublicationContext(f.db, f.r.Organization.ID, node, execution), Metadata: contexts.NewExecutionMetadataContext(f.db, execution), ExecutionState: contexts.NewExecutionStateContext(f.db, execution, nil), Requests: contexts.NewExecutionRequestContext(f.db, execution)}
}
func (f *reviewCheckFixture) apply(phase string, reviewers []pulls.ReviewerIdentity, outcome string) error {
	f.ctx.Configuration = map[string]any{"repository": "hello", "pullNumber": 42, "assessedSha": "abc", "context": "team/review", "phase": phase, "reason": "Review decision", "reviewers": reviewers, "reviewerRequestExecutionId": outcome}
	if f.activationNode != "" {
		f.ctx.Configuration.(map[string]any)["activationNodeId"] = f.activationNode
	}
	return (&pulls.SetPullRequestReviewCheck{}).Execute(f.ctx)
}
func (f *reviewCheckFixture) ensure(reviewers []pulls.ReviewerIdentity) string {
	execution := support.CreateCanvasNodeExecution(f.t, f.canvas.ID, "ensure", f.root.ID, f.root.ID)
	execution.Configuration = datatypes.NewJSONType(map[string]any{"repository": "hello", "pullNumber": 42, "expectedHeadSha": "abc", "reviewers": reviewers})
	require.NoError(f.t, f.db.Save(execution).Error)
	ctx := f.executionContext(execution, &f.nodes[1])
	ctx.Configuration = execution.Configuration.Data()
	require.NoError(f.t, (&pulls.EnsurePullRequestReviewers{}).Execute(ctx))
	return execution.ID.String()
}
func (f *reviewCheckFixture) refresh() error {
	integration, err := f.r.Registry.GetIntegration("github")
	require.NoError(f.t, err)
	return integration.HandleHook(core.IntegrationHookContext{Name: pulls.ReviewCheckHook, Parameters: map[string]any{"key": "github.review-check/testhq/hello/42/team/review"}, Integration: f.integration, HTTP: f.http})
}
func (f *reviewCheckFixture) record() *core.OrderedPublication {
	record, err := f.integration.store.Get("github.review-check/testhq/hello/42/team/review")
	require.NoError(f.t, err)
	require.NotNil(f.t, record)
	return record
}
func (f *reviewCheckFixture) state() string {
	f.t.Helper()
	require.NotEmpty(f.t, f.http.writes)
	return f.http.writes[len(f.http.writes)-1].Status.GetState()
}
func reviewDecision(id int, user int64, login, state, sha string) map[string]any {
	return map[string]any{"id": id, "user": map[string]any{"id": user, "login": login}, "state": state, "commit_id": sha, "submitted_at": time.Now().Add(time.Duration(id) * time.Second).UTC().Format(time.RFC3339)}
}

// The provider history and writes are real component operations; the fixture
// replaces only HTTP, not approval evaluation, persistence, or request scheduling.
func TestReviewCheckTracksCurrentApprovals(t *testing.T) {
	f := newReviewCheckFixture(t)
	reviewers := []pulls.ReviewerIdentity{{ID: 2, Login: "first"}, {ID: 3, Login: "second"}}
	f.http.reviews = []map[string]any{reviewDecision(1, 2, "first", "APPROVED", "abc"), reviewDecision(2, 3, "second", "APPROVED", "abc")}
	outcome := f.ensure(reviewers)
	require.Zero(t, f.http.requestCount, "existing approvals on all pages cover reviewer requests")
	require.NoError(t, f.apply("required_reviewers", reviewers, outcome))
	require.NoError(t, f.refresh())
	require.Equal(t, "success", f.state())
	f.finishAction()
	require.NotNil(t, f.record().WakeAt, "success must continue monitoring")

	for _, step := range []struct{ state, sha, want string }{
		{"COMMENTED", "abc", "success"}, {"CHANGES_REQUESTED", "abc", "failure"}, {"APPROVED", "abc", "success"}, {"DISMISSED", "abc", "pending"}, {"APPROVED", "old", "pending"}, {"APPROVED", "abc", "success"},
	} {
		id := 10 + len(f.http.reviews)
		decision := reviewDecision(id, 3, "second", step.state, step.sha)
		// Dismissal replaces the latest review, rather than creating a new review.
		if step.state == "DISMISSED" {
			f.http.reviews[len(f.http.reviews)-1]["state"] = "DISMISSED"
		} else {
			f.http.reviews = append(f.http.reviews, decision)
		}
		body := []byte(`{"repository":{"full_name":"testhq/hello"},"pull_request":{"number":42}}`)
		signature := hmac.New(sha256.New, []byte("review-hook-secret"))
		_, err := signature.Write(body)
		require.NoError(t, err)
		headers := http.Header{"X-Github-Event": {"pull_request_review"}, "X-Hub-Signature-256": {"sha256=" + hex.EncodeToString(signature.Sum(nil))}}
		code, _, err := (&pulls.SetPullRequestReviewCheck{}).HandleWebhook(core.WebhookRequestContext{Headers: headers, Body: body, Integration: f.integration, Webhook: &testcontexts.NodeWebhookContext{Secret: "review-hook-secret"}})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, code)
		require.NoError(t, f.processDue())
		require.Equal(t, step.want, f.state(), step.state)
	}
	var executions int64
	require.NoError(t, f.db.Model(&models.CanvasNodeExecution{}).Where("workflow_id = ?", f.canvas.ID).Count(&executions).Error)
	require.EqualValues(t, 2, executions, "review events must not start new executions")
	require.Zero(t, f.http.requestCount, "tracking must not request reviews a second time")
	f.http.head = "new-head"
	require.NoError(t, f.refresh())
	require.Equal(t, "pending", f.state())
	require.Equal(t, "new-head", f.http.writes[len(f.http.writes)-1].SHA)
	f.http.closed = true
	require.NoError(t, f.refresh())
	require.Nil(t, f.record().WakeAt)
	f.http.closed = false
	require.NoError(t, pulls.WakeReviewChecks(f.integration, "pull_request", []byte(`{"repository":{"full_name":"testhq/hello"},"pull_request":{"number":42}}`)))
	require.NotNil(t, f.record().WakeAt)
}

func TestReviewCheckPersistsDeliveryAndRecoversWithoutWebhooks(t *testing.T) {
	f := newReviewCheckFixture(t)
	reviewers := []pulls.ReviewerIdentity{{ID: 2, Login: "first"}}
	outcome := f.ensure(reviewers)
	require.Equal(t, 1, f.http.requestCount)
	require.NoError(t, f.apply("required_reviewers", reviewers, outcome))
	f.http.failWrite = true
	require.ErrorContains(t, f.refresh(), "publish review check")
	require.NotNil(t, f.record().WakeAt)
	var desired map[string]any
	require.NoError(t, json.Unmarshal(f.record().Payload, &desired))
	require.Equal(t, "pending", desired["state"])
	require.Empty(t, desired["delivered"])
	f.http.failWrite = false
	require.NoError(t, f.refresh())
	require.Equal(t, "pending", f.state())
	f.http.failReviews = true
	require.Error(t, f.refresh())
	require.Equal(t, "error", f.state())
	f.http.failReviews = false
	f.http.reviews = []map[string]any{reviewDecision(1, 2, "first", "APPROVED", "abc")}
	// A due integration request recovers the state without a webhook or new run.
	require.Len(t, pendingActionRequests(t, f.installation.ID, pulls.ReviewCheckHook), 1)
	require.NoError(t, f.processDue())
	require.Equal(t, "success", f.state())
	writes := len(f.http.writes)
	require.NoError(t, f.apply("required_reviewers", reviewers, outcome))
	require.NoError(t, f.refresh())
	require.Len(t, f.http.writes, writes, "duplicate application must not publish again")
	require.Len(t, pendingActionRequests(t, f.installation.ID, pulls.ReviewCheckHook), 1)
	require.Equal(t, 1, f.http.requestCount)
}

func TestReviewCheckReapplyRestartsExhaustedDelivery(t *testing.T) {
	f := newReviewCheckFixture(t)
	f.http.failPR = true
	require.NoError(t, f.apply("disabled", nil, ""))
	generation := f.record().Generation
	for i := 0; i < 10; i++ {
		require.Error(t, f.processDue())
	}
	require.Empty(t, pendingActionRequests(t, f.installation.ID, pulls.ReviewCheckHook))
	f.http.failPR = false
	f.attempt(f.root)
	require.NoError(t, f.apply("disabled", nil, ""))
	require.Equal(t, generation, f.record().Generation)
	require.Len(t, pendingActionRequests(t, f.installation.ID, pulls.ReviewCheckHook), 1)
	require.NoError(t, f.processDue())
	require.Equal(t, "success", f.state())
	require.Nil(t, f.record().WakeAt)
}

func TestReviewCheckContextUsesOneCaseInsensitiveRequirement(t *testing.T) {
	f := newReviewCheckFixture(t)
	require.NoError(t, f.apply("no_review", nil, ""))
	require.NoError(t, f.refresh())
	generation := f.record().Generation
	f.ctx.Configuration.(map[string]any)["context"] = "TEAM/REVIEW"
	require.NoError(t, (&pulls.SetPullRequestReviewCheck{}).Execute(f.ctx))
	records, err := f.integration.store.List("github.review-check/testhq/hello/42/")
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, generation, records[0].Generation)
}

func TestReviewCheckRejectsUnrecordedOrForeignReviewerOutcome(t *testing.T) {
	f := newReviewCheckFixture(t)
	reviewers := []pulls.ReviewerIdentity{{ID: 2, Login: "first"}}
	require.Error(t, f.apply("required_reviewers", reviewers, uuid.NewString()))
	outcome := f.ensure(reviewers)
	require.ErrorContains(t, f.apply("required_reviewers", []pulls.ReviewerIdentity{{ID: 3, Login: "second"}}, outcome), "missing from the recorded outcome")
	f.attempt(support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil))
	require.Error(t, f.apply("required_reviewers", reviewers, outcome))
}

func TestReviewCheckSelectionFailureAndDisableFence(t *testing.T) {
	f := newReviewCheckFixture(t)
	require.NoError(t, f.apply("pending_selection", nil, ""))
	require.NoError(t, f.refresh())
	require.Equal(t, "pending", f.state())
	// No failed-channel event is needed for cancellation to fail selection.
	require.NoError(t, f.db.Model(&models.CanvasRun{}).Where("id = ?", f.ctx.RunID).Update("state", models.CanvasRunStateCancelling).Error)
	require.NoError(t, f.refresh())
	require.Equal(t, "error", f.state())
	require.NoError(t, f.apply("disabled", nil, ""))
	f.http.failWrite = true
	require.Error(t, f.refresh())
	require.True(t, f.record().Disabled)
	require.NotNil(t, f.record().WakeAt)
	f.http.failWrite = false
	require.NoError(t, f.refresh())
	require.Equal(t, "success", f.state())
	require.Equal(t, "Review request check disabled", f.http.writes[len(f.http.writes)-1].Status.GetDescription())
	require.Nil(t, f.record().WakeAt)
	require.Empty(t, pendingActionRequests(t, f.installation.ID, pulls.ReviewCheckHook))
	require.NoError(t, f.apply("no_review", nil, ""))
	require.True(t, f.record().Disabled)
	// A newer source still belongs to the disabled activation unless explicitly enabled.
	f.attempt(support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil))
	require.NoError(t, f.apply("no_review", nil, ""))
	require.True(t, f.record().Disabled)
	f.attempt(support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil))
	require.NoError(t, f.apply("no_review", nil, ""))
	f.ctx.Configuration.(map[string]any)["activationGeneration"] = 2
	require.NoError(t, (&pulls.SetPullRequestReviewCheck{}).Execute(f.ctx))
	require.NoError(t, f.refresh())
	require.False(t, f.record().Disabled)
	require.Equal(t, int64(2), f.record().ActivationGeneration)
}

func TestReviewCheckRechecksGenerationAfterLookup(t *testing.T) {
	f := newReviewCheckFixture(t)
	require.NoError(t, f.apply("no_review", nil, ""))
	entered, resume := make(chan struct{}), make(chan struct{})
	var once sync.Once
	f.http.onRead = func() { once.Do(func() { close(entered); <-resume }) }
	done := make(chan error, 1)
	go func() { done <- f.refresh() }()
	<-entered
	require.NoError(t, f.apply("disabled", nil, ""))
	close(resume)
	require.NoError(t, <-done)
	require.Empty(t, f.http.writes, "an in-flight lookup cannot publish over disable")
	f.http.onRead = nil
	require.NoError(t, f.refresh())
	require.Equal(t, "success", f.state())
	require.True(t, f.record().Disabled)
}

func TestReviewCheckDeliveryDoesNotBlockReplacement(t *testing.T) {
	f := newReviewCheckFixture(t)
	require.NoError(t, f.apply("selection_error", nil, ""))
	entered, resume := make(chan struct{}), make(chan struct{})
	var once sync.Once
	f.http.onWrite = func() { once.Do(func() { close(entered); <-resume }) }
	done := make(chan error, 1)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		done <- f.refresh()
	}()
	<-entered
	var release sync.Once
	t.Cleanup(func() {
		release.Do(func() { close(resume) })
		<-finished
	})

	replaced := make(chan error, 1)
	go func() { replaced <- f.apply("disabled", nil, "") }()
	select {
	case err := <-replaced:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		release.Do(func() { close(resume) })
		<-done
		<-replaced
		t.Fatal("a GitHub status request must not block a newer decision")
	}
	require.NoError(t, f.refresh())
	require.Empty(t, f.http.writes, "concurrent refresh must wait for the active delivery")
	release.Do(func() { close(resume) })
	require.NoError(t, <-done)
	f.http.onWrite = nil
	var state map[string]any
	require.NoError(t, json.Unmarshal(f.record().Payload, &state))
	require.Empty(t, state["delivered"], "an old delivery cannot acknowledge the replacement")
	require.NotNil(t, f.record().WakeAt)
	require.NoError(t, f.processDue())
	require.Len(t, f.http.writes, 2)
	require.Equal(t, "success", f.state())
	require.Equal(t, "Review request check disabled", f.http.writes[1].Status.GetDescription())
	require.Nil(t, f.record().WakeAt)
}

func TestReviewCheckRecoversInterruptedDelivery(t *testing.T) {
	f := newReviewCheckFixture(t)
	require.NoError(t, f.apply("selection_error", nil, ""))
	key := f.record().Key
	expiresAt := time.Now().Add(time.Minute)
	setLease := func(expiry time.Time) {
		require.NoError(t, f.integration.store.Update(key, func(record *core.OrderedPublication) error {
			var state map[string]any
			if err := json.Unmarshal(record.Payload, &state); err != nil {
				return err
			}
			state["delivery"] = map[string]any{"id": uuid.NewString(), "expiresAt": expiry}
			var err error
			record.Payload, err = json.Marshal(state)
			return err
		}))
	}
	setLease(expiresAt)
	require.NoError(t, f.processDue())
	require.Empty(t, f.http.writes, "a live lease prevents overlapping status writes")
	requests := pendingActionRequests(t, f.installation.ID, pulls.ReviewCheckHook)
	require.Len(t, requests, 1, "skipping an active delivery must retain recovery work")
	require.WithinDuration(t, expiresAt, requests[0].RunAt, time.Second)

	setLease(time.Now().Add(-time.Second))
	require.NoError(t, f.processDue())
	require.Equal(t, "error", f.state())
	var state map[string]any
	require.NoError(t, json.Unmarshal(f.record().Payload, &state))
	require.NotEmpty(t, state["delivered"])
	require.Empty(t, state["delivery"], "the recovered delivery must release its lease")
}

func TestReviewCheckFencesLateDeliveryCompletion(t *testing.T) {
	for _, recoveryInFlight := range []bool{true, false} {
		t.Run(fmt.Sprintf("recovery_in_flight=%t", recoveryInFlight), func(t *testing.T) {
			f := newReviewCheckFixture(t)
			require.NoError(t, f.apply("selection_error", nil, ""))
			entered, resume := make(chan struct{}), make(chan struct{})
			recovering, finishRecovery := make(chan struct{}), make(chan struct{})
			var calls atomic.Int32
			f.http.onWrite = func() {
				call := calls.Add(1)
				if call == 1 {
					close(entered)
					<-resume
				}
				if call == 2 && recoveryInFlight {
					close(recovering)
					<-finishRecovery
				}
			}
			done := make(chan error, 1)
			finished := make(chan struct{})
			go func() {
				defer close(finished)
				done <- f.refresh()
			}()
			var release, releaseRecovery sync.Once
			t.Cleanup(func() {
				release.Do(func() { close(resume) })
				releaseRecovery.Do(func() { close(finishRecovery) })
				<-finished
			})
			<-entered
			require.NoError(t, f.apply("disabled", nil, ""))
			// Simulate lease expiry while the old provider response is delayed.
			require.NoError(t, f.integration.store.Update(f.record().Key, func(record *core.OrderedPublication) error {
				var state map[string]any
				if err := json.Unmarshal(record.Payload, &state); err != nil {
					return err
				}
				state["delivery"].(map[string]any)["expiresAt"] = time.Now().Add(-time.Second)
				var err error
				record.Payload, err = json.Marshal(state)
				return err
			}))

			recovered := make(chan error, 1)
			go func() { recovered <- f.refresh() }()
			if recoveryInFlight {
				<-recovering
			} else {
				require.NoError(t, <-recovered)
			}
			release.Do(func() { close(resume) })
			require.NoError(t, <-done)
			if recoveryInFlight {
				require.NoError(t, f.refresh())
				require.Len(t, f.http.writes, 1, "an expired owner cannot release the new delivery lease")
				releaseRecovery.Do(func() { close(finishRecovery) })
				require.NoError(t, <-recovered)
			}
			f.http.onWrite = nil
			require.NoError(t, f.processDue())
			require.Equal(t, "success", f.state(), "late responses must not leave an obsolete status on GitHub")
			require.Nil(t, f.record().WakeAt)
		})
	}
}

func TestReviewCheckDetectsAttemptFailureWithoutFailedChannel(t *testing.T) {
	f := newReviewCheckFixture(t)
	require.NoError(t, f.apply("pending_selection", nil, ""))
	failed := support.CreateCanvasNodeExecution(t, f.canvas.ID, "ensure", f.root.ID, f.root.ID)
	require.NoError(t, contexts.NewExecutionStateContext(f.db, failed, nil).Fail(models.CanvasNodeExecutionResultReasonError, "selection failed"))
	var count int64
	require.NoError(t, f.db.Model(&models.CanvasEvent{}).Where("run_id = ? AND channel = ?", f.ctx.RunID, "failed").Count(&count).Error)
	require.Zero(t, count)
	require.NoError(t, f.refresh())
	require.Equal(t, "error", f.state())
}

func TestReviewCheckPreservesSourceAcrossOrdinaryTriggerRetries(t *testing.T) {
	f := newReviewCheckFixture(t)
	old := f.root
	newer := support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil)
	f.attempt(newer)
	require.NoError(t, f.apply("no_review", nil, ""))
	require.NoError(t, f.refresh())
	current := f.record().Generation

	// Retrying an old event, including a retry of that retry, keeps its source.
	for range 2 {
		retry, err := canvasactions.ReemitTriggerEvent(t.Context(), f.db, f.canvas, "trigger", old.ID)
		require.NoError(t, err)
		old, err = models.FindCanvasEventInTransaction(f.db, uuid.MustParse(retry.EventId))
		require.NoError(t, err)
		f.attempt(old)
		require.NoError(t, f.apply("pending_selection", nil, ""))
		require.Equal(t, current, f.record().Generation, "an older source cannot replace the current decision")
	}

	// A retry of the current source can still replace its earlier attempt.
	retry, err := canvasactions.ReemitTriggerEvent(t.Context(), f.db, f.canvas, "trigger", newer.ID)
	require.NoError(t, err)
	retried, err := models.FindCanvasEventInTransaction(f.db, uuid.MustParse(retry.EventId))
	require.NoError(t, err)
	f.attempt(retried)
	require.NoError(t, f.apply("selection_error", nil, ""))
	require.NoError(t, f.refresh())
	require.Greater(t, f.record().Generation, current)
	require.Equal(t, "error", f.state())
}

func TestPublicationOrderingUsesBroadcastProvenanceAndOriginalRetry(t *testing.T) {
	f := newReviewCheckFixture(t)
	source, nodes := support.CreateCanvas(t, f.r.Organization.ID, f.r.User, []models.CanvasNode{{NodeID: "start", Type: models.NodeTypeTrigger}, {NodeID: "broadcast", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "broadcastMessage"}})}}, nil)
	require.NoError(t, f.db.Create(&models.CanvasSubscription{SourceCanvasID: source.ID, TargetCanvasID: f.canvas.ID, TargetNodeID: "trigger"}).Error)
	broadcast := func() *models.CanvasEvent {
		root := support.EmitCanvasEventForNode(t, source.ID, "start", "default", nil)
		execution := support.CreateCanvasNodeExecution(t, source.ID, "broadcast", root.ID, root.ID)
		require.NoError(t, contexts.NewAppExecutionContext(f.db, source, &nodes[1], execution).Broadcast(map[string]any{"timestamp": "2099-01-01", "text": "original report"}))
		messages, err := models.ListAppMessages(f.db)
		require.NoError(t, err)
		require.Len(t, messages, 1)
		require.NoError(t, NewAppMessageWorker(f.r.Registry).LockAndProcessMessage(messages[0]))
		var delivered models.CanvasEvent
		require.NoError(t, f.db.Where("workflow_id = ? AND source_run_id = ?", f.canvas.ID, execution.RunID).First(&delivered).Error)
		return &delivered
	}
	old := broadcast()
	newer := broadcast()
	f.attempt(newer)
	// The ordering capability works without creating a status or approval monitor.
	order, err := f.ctx.Publications.ResolveOrder(uuid.Nil, uuid.Nil)
	require.NoError(t, err)
	require.Equal(t, *newer.SourceRunID, order.SourceRunID)
	require.Empty(t, f.http.writes)
	require.Empty(t, pendingActionRequests(t, f.installation.ID, pulls.ReviewCheckHook))
	require.NoError(t, f.apply("no_review", nil, ""))
	require.NoError(t, f.refresh())
	require.Equal(t, "success", f.state())
	current := f.record().Generation
	retry, err := canvasactions.ReemitTriggerEvent(t.Context(), f.db, f.canvas, "trigger", old.ID)
	require.NoError(t, err)
	retried, err := models.FindCanvasEventInTransaction(f.db, uuid.MustParse(retry.EventId))
	require.NoError(t, err)
	require.Equal(t, old.Data.Data(), retried.Data.Data())
	f.attempt(retried)
	require.NoError(t, f.apply("pending_selection", nil, ""))
	require.Equal(t, current, f.record().Generation, "a newer retry cannot make an older source current")
	_, err = f.ctx.Publications.ResolveOrder(uuid.New(), uuid.Nil)
	require.ErrorContains(t, err, "authorized broadcast")
	_, err = f.ctx.Publications.ResolveOrder(f.ctx.RunID, uuid.Nil)
	require.ErrorContains(t, err, "authorized broadcast", "the consumer run cannot replace persisted source provenance")
	_, err = f.ctx.Publications.ResolveOrder(uuid.Nil, newer.ID)
	require.ErrorContains(t, err, "current execution root")
	// A retry of the current source supersedes its original attempt on the same SHA.
	retry, err = canvasactions.ReemitTriggerEvent(t.Context(), f.db, f.canvas, "trigger", newer.ID)
	require.NoError(t, err)
	retried, err = models.FindCanvasEventInTransaction(f.db, uuid.MustParse(retry.EventId))
	require.NoError(t, err)
	f.attempt(retried)
	require.NoError(t, f.apply("selection_error", nil, ""))
	require.NoError(t, f.refresh())
	require.Equal(t, "error", f.state())
	require.Greater(t, f.record().Generation, current)
	current = f.record().Generation
	f.attempt(newer)
	require.NoError(t, f.apply("no_review", nil, ""))
	f.ctx.Configuration.(map[string]any)["sourceCreatedAt"] = "2099-01-01T00:00:00Z"
	require.NoError(t, (&pulls.SetPullRequestReviewCheck{}).Execute(f.ctx))
	require.Equal(t, current, f.record().Generation)
	f.attempt(newer)
	require.NoError(t, f.apply("disabled", nil, ""))
	require.False(t, f.record().Disabled, "a stale attempt cannot disable a newer requirement")
	require.Equal(t, current, f.record().Generation)
}

// Publish actual versions so the attempt owner, rather than the test, supplies
// participation snapshots. Setup and durable integration requests stay real.
type reviewCheckSetupAction struct {
	core.Action
	integration core.IntegrationContext
	http        core.HTTPContext
}

func (a reviewCheckSetupAction) Setup(ctx core.SetupContext) error {
	ctx.Integration, ctx.HTTP = a.integration, a.http
	return a.Action.Setup(ctx)
}

func (f *reviewCheckFixture) saveActivation(enabled bool, generation int, pause ...bool) {
	f.r.Registry.Actions["github.setPullRequestReviewCheck"] = reviewCheckSetupAction{Action: &pulls.SetPullRequestReviewCheck{}, integration: f.integration, http: f.http}
	version, err := models.FindLiveCanvasVersionInTransaction(f.db, f.canvas.ID)
	require.NoError(f.t, err)
	for i := range version.Nodes {
		node := &version.Nodes[i]
		node.ErrorMessage = nil
		if node.ID == "trigger" && len(pause) > 0 {
			node.Configuration = map[string]any{"enabled": !pause[0]}
		}
		if node.ID != "check" {
			continue
		}
		integrationID := f.installation.ID.String()
		node.IntegrationID = &integrationID
		node.Configuration = map[string]any{"repository": "{{ root().data.repository }}", "pullNumber": "{{ root().data.pullNumber }}", "enabled": enabled, "activationGeneration": generation, "activationNodeId": "check", "assessedSha": "abc", "context": "team/review", "phase": "pending_selection", "reason": "Select required reviewers"}
	}
	require.NoError(f.t, canvasactions.PublishGeneratedCanvasNodesWithOwner(f.t.Context(), f.db, f.canvas, nil, "Save publication settings", version.Nodes, version.Edges, changesets.CanvasPublisherOptions{Registry: f.r.Registry, OrgID: f.r.Organization.ID, Encryptor: f.r.Encryptor, AuthService: f.r.AuthService, WebhookBaseURL: "http://localhost:8000"}))
	current, err := models.FindCanvasNode(f.db, f.canvas.ID, "check")
	require.NoError(f.t, err)
	if current.StateReason != nil {
		f.t.Fatalf("check setup failed: %s", *current.StateReason)
	}
	require.Equal(f.t, models.CanvasNodeStateReady, current.State)
	require.Equal(f.t, enabled, current.Configuration.Data()["enabled"])

}

func TestReviewCheckSavedActivationFencesInFlightAttempts(t *testing.T) {
	f := newReviewCheckFixture(t)
	f.activationNode = "check"
	f.saveActivation(false, 1)
	f.attempt(support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil))
	disabledAttempt := f.root
	f.saveActivation(true, 2)
	require.NoError(t, f.apply("pending_selection", nil, ""))
	require.Empty(t, pendingActionRequests(t, f.installation.ID, pulls.ReviewCheckHook), "enabling cannot attach checks to admitted attempts")
	f.attempt(support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil))
	enabledAttempt := f.root
	require.NoError(t, f.apply("pending_selection", nil, ""))
	require.NoError(t, f.processDue())
	require.Equal(t, "pending", f.state())
	// Pausing selection does not invalidate approval tracking.
	f.saveActivation(true, 2, true)
	require.False(t, f.record().Disabled)
	f.saveActivation(false, 2)
	require.True(t, f.record().Disabled)
	f.http.failWrite = true
	require.Error(t, f.processDue())
	f.http.failWrite = false
	require.NoError(t, f.processDue())
	require.Equal(t, "success", f.state())
	require.Nil(t, f.record().WakeAt)
	f.saveActivation(true, 5, false) // PRs can miss intervening activation generations.
	generation := f.record().Generation
	for _, old := range []*models.CanvasEvent{disabledAttempt, enabledAttempt} {
		f.attempt(old)
		require.NoError(t, f.apply("no_review", nil, ""))
		require.Equal(t, generation, f.record().Generation)
		require.True(t, f.record().Disabled)
	}
	retry, err := canvasactions.ReemitTriggerEvent(t.Context(), f.db, f.canvas, "trigger", enabledAttempt.ID)
	require.NoError(t, err)
	retried, err := models.FindCanvasEventInTransaction(f.db, uuid.MustParse(retry.EventId))
	require.NoError(t, err)
	f.attempt(retried)
	require.NoError(t, f.apply("no_review", nil, ""))
	require.NoError(t, f.processDue())
	require.False(t, f.record().Disabled)
	require.Equal(t, int64(5), f.record().ActivationGeneration)
}

func TestReviewCheckRemovalReleasesDormantChecks(t *testing.T) {
	for _, removeNode := range []bool{false, true} {
		t.Run(fmt.Sprintf("remove_node=%t", removeNode), func(t *testing.T) {
			f := newReviewCheckFixture(t)
			f.activationNode = "check"
			f.saveActivation(true, 4)
			f.attempt(support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil))
			require.NoError(t, f.apply("pending_selection", nil, ""))
			f.http.closed = true
			require.NoError(t, f.processDue())
			require.Nil(t, f.record().WakeAt)
			if removeNode {
				_, err := models.DeleteCanvasNodeWithResult(f.db, f.nodes[2])
				require.NoError(t, err)
			} else {
				_, err := canvasactions.DeleteCanvas(t.Context(), f.db, f.canvas)
				require.NoError(t, err)
			}
			require.NoError(t, f.processDue()) // Persist the disabled decision.
			require.True(t, f.record().Disabled)
			require.NoError(t, f.processDue()) // Deliver the release.
			require.Equal(t, "success", f.state())
			require.Equal(t, "Review request check disabled", f.http.writes[len(f.http.writes)-1].Status.GetDescription())
			require.Nil(t, f.record().WakeAt)
		})
	}
}

func TestReviewCheckTracksApprovalsAfterRunRetention(t *testing.T) {
	f := newReviewCheckFixture(t)
	f.activationNode = "check"
	f.saveActivation(true, 2)
	f.attempt(support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil))
	reviewers := []pulls.ReviewerIdentity{{ID: 2, Login: "first"}}
	outcome := f.ensure(reviewers)
	require.NoError(t, f.apply("required_reviewers", reviewers, outcome))
	require.NoError(t, f.refresh())
	f.finishAction()
	var run models.CanvasRun
	require.NoError(t, f.db.First(&run, "id = ?", f.ctx.RunID).Error)
	_, err := run.DeleteChain(f.db)
	require.NoError(t, err)
	for _, step := range []struct{ decision, want string }{{"APPROVED", "success"}, {"DISMISSED", "pending"}} {
		f.http.reviews = []map[string]any{reviewDecision(1, 2, "first", step.decision, "abc")}
		require.NoError(t, pulls.WakeReviewChecks(f.integration, "pull_request_review", []byte(`{"repository":{"full_name":"testhq/hello"},"pull_request":{"number":42}}`)))
		require.NoError(t, f.processDue())
		require.Equal(t, step.want, f.state())
	}
	f.saveActivation(false, 2)
	require.NoError(t, f.processDue())
	require.True(t, f.record().Disabled)
	require.Equal(t, "Review request check disabled", f.http.writes[len(f.http.writes)-1].Status.GetDescription())
}
