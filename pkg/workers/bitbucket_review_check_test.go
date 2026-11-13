package workers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

const bbReviewRepository = "{22222222-2222-4222-8222-222222222222}"
const bbReviewUser = "{11111111-1111-4111-8111-111111111111}"
const bbReviewKey = "bitbucket.review-check/22222222-2222-4222-8222-222222222222/42"
const bbReviewHook = "refreshPullRequestReviewCheck"

type bitbucketReviewHTTP struct {
	t           *testing.T
	head        string
	heads       []string
	state       string
	participant string
	approval    time.Time
	writes      []map[string]any
	beforeWrite func()
	beforeRead  func()
	failWrite   bool
	failRead    bool
}

func (h *bitbucketReviewHTTP) Do(request *http.Request) (*http.Response, error) {
	if request.Method == http.MethodPost {
		if h.beforeWrite != nil {
			callback := h.beforeWrite
			h.beforeWrite = nil
			callback()
		}
		if err := request.Context().Err(); err != nil {
			return nil, err
		}
		if h.failWrite {
			return nil, fmt.Errorf("provider unavailable")
		}
		var status map[string]any
		require.NoError(h.t, json.NewDecoder(request.Body).Decode(&status))
		status["path"] = request.URL.Path
		h.writes = append(h.writes, status)
		return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	}
	if h.beforeRead != nil {
		callback := h.beforeRead
		h.beforeRead = nil
		callback()
	}
	if h.failRead {
		return nil, fmt.Errorf("lookup unavailable")
	}
	body := fmt.Sprintf(`{"uuid":%q,"full_name":"acme/widgets"}`, bbReviewRepository)
	if strings.Contains(request.URL.Path, "/pullrequests/") && !strings.HasSuffix(request.URL.Path, "/activity") {
		if len(h.heads) > 0 {
			h.head = h.heads[0]
			h.heads = h.heads[1:]
		}
		body = fmt.Sprintf(`{"id":42,"author":{"uuid":"11111111-1111-4111-8111-111111111111"},"reviewers":[],"state":%q,"source":{"commit":{"hash":%q},"branch":{"name":"topic"}},"participants":[{"user":{"uuid":%q},%s}]}`, h.state, h.head, bbReviewUser, h.participant)
	}
	if strings.HasSuffix(request.URL.Path, "/activity") {
		body = fmt.Sprintf(`{"values":[{"approval":{"date":%q,"user":{"uuid":%q}}}]}`, h.approval.Format(time.RFC3339Nano), bbReviewUser)
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
}

type bitbucketReviewFixture struct {
	*reviewCheckFixture
	http *bitbucketReviewHTTP
}

func newBitbucketReviewFixture(t *testing.T) *bitbucketReviewFixture {
	f := newPullRequestCheckFixture(t, "bitbucket.setPullRequestReviewCheck")
	integration := reviewCheckIntegration{IntegrationContext: &testcontexts.IntegrationContext{IntegrationID: f.installation.ID.String(), Configuration: map[string]any{"token": "token"}, Metadata: bitbucket.Metadata{AuthType: bitbucket.AuthTypeWorkspaceAccessToken, Workspace: &bitbucket.WorkspaceMetadata{Slug: "acme"}}}, store: f.integration.store}
	httpCtx := &bitbucketReviewHTTP{t: t, head: strings.Repeat("a", 40), state: "OPEN", participant: `"approved":false`, approval: time.Now().UTC().Truncate(time.Microsecond)}
	f.integration = integration
	f.ctx.Integration, f.ctx.HTTP = integration, httpCtx
	f.ctx.BaseURL = "https://app.superplane.com"
	f.ctx.Configuration = map[string]any{"repository": "acme/widgets", "pullRequestId": "42", "assessedSha": httpCtx.head, "phase": "pending_selection", "reason": "Review required"}
	return &bitbucketReviewFixture{reviewCheckFixture: f, http: httpCtx}
}

func (f *bitbucketReviewFixture) apply(phase string) error {
	f.ctx.Configuration.(map[string]any)["phase"] = phase
	return (&bitbucket.SetPullRequestReviewCheck{}).Execute(f.ctx)
}
func (f *bitbucketReviewFixture) refresh() error {
	return (&bitbucket.Bitbucket{}).HandleHook(core.IntegrationHookContext{Name: bbReviewHook, Parameters: map[string]any{"key": bbReviewKey}, Integration: f.integration, HTTP: f.http})
}
func (f *bitbucketReviewFixture) record() *core.OrderedPublication {
	record, err := f.integration.store.Get(bbReviewKey)
	require.NoError(f.t, err)
	require.NotNil(f.t, record)
	return record
}
func (f *bitbucketReviewFixture) state() string {
	require.NotEmpty(f.t, f.http.writes)
	return f.http.writes[len(f.http.writes)-1]["state"].(string)
}
func (f *bitbucketReviewFixture) ensure() {
	execution := support.CreateCanvasNodeExecution(f.t, f.canvas.ID, "ensure", f.root.ID, f.root.ID)
	execution.Configuration = datatypes.NewJSONType(map[string]any{"repository": "acme/widgets", "pullNumber": f.ctx.Configuration.(map[string]any)["pullRequestId"]})
	require.NoError(f.t, f.db.Save(execution).Error)
	ctx := f.executionContext(execution, &f.nodes[1])
	require.NoError(f.t, ctx.ExecutionState.Emit("default", "bitbucket.pullRequestReviewersEnsured", []any{map[string]any{"executionId": execution.ID.String(), "provider": "bitbucket", "integrationId": f.installation.ID.String(), "repositoryId": strings.Trim(bbReviewRepository, "{}"), "repository": "acme/widgets", "pullNumber": 42, "status": "requested", "headSha": f.http.head, "requested": []bitbucket.ReviewerIdentity{{UUID: bbReviewUser, DisplayName: "Reviewer"}}}}))
	f.ctx.Configuration.(map[string]any)["reviewers"] = []bitbucket.ReviewerIdentity{{UUID: bbReviewUser, DisplayName: "Reviewer"}}
	f.ctx.Configuration.(map[string]any)["reviewerRequestExecutionId"] = execution.ID.String()
}
func (f *bitbucketReviewFixture) event(eventKey, sha string, date time.Time) {
	body := []byte(fmt.Sprintf(`{"repository":{"uuid":%q},"pullrequest":{"id":42,"source":{"commit":{"hash":%q}}},"actor":{"uuid":%q},"approval":{"user":{"uuid":%q},"date":%q}}`, bbReviewRepository, sha, bbReviewUser, bbReviewUser, date.Format(time.RFC3339Nano)))
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write(body)
	headers := http.Header{}
	headers.Set("X-Hub-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	headers.Set("X-Event-Key", eventKey)
	code, _, err := (&bitbucket.SetPullRequestReviewCheck{}).HandleWebhook(core.WebhookRequestContext{Headers: headers, Body: body, Integration: f.integration, Webhook: &testcontexts.NodeWebhookContext{Secret: "secret"}})
	require.NoError(f.t, err)
	require.Equal(f.t, 200, code)
}

func (f *bitbucketReviewFixture) saveActivation(enabled bool, generation int) {
	version, err := models.FindLiveCanvasVersionInTransaction(f.db, f.canvas.ID)
	require.NoError(f.t, err)
	version.ID = uuid.New()
	for i := range version.Nodes {
		if version.Nodes[i].ID == "check" {
			version.Nodes[i].Configuration = map[string]any{"enabled": enabled, "activationGeneration": generation}
		}
	}
	require.NoError(f.t, f.db.Create(version).Error)
	require.NoError(f.t, f.db.Model(f.canvas).Update("live_version_id", version.ID).Error)
	require.NoError(f.t, f.db.Model(&f.nodes[2]).Update("configuration", datatypes.NewJSONType(map[string]any{"enabled": enabled, "activationGeneration": generation})).Error)
	if !enabled {
		require.NoError(f.t, (&bitbucket.SetPullRequestReviewCheck{}).Setup(core.SetupContext{WorkflowID: f.canvas.ID.String(), NodeID: "check", Configuration: map[string]any{"enabled": false}, Integration: f.integration}))
	}
}

func (f *bitbucketReviewFixture) processDue() error {
	provider, err := f.r.Registry.GetIntegration("bitbucket")
	require.NoError(f.t, err)
	f.r.Registry.Integrations["bitbucket"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{Actions: provider.Actions(), Hooks: provider.Hooks(), HandleHook: func(ctx core.IntegrationHookContext) error {
		ctx.Integration, ctx.HTTP = f.integration, f.http
		return provider.HandleHook(ctx)
	}})
	defer func() { f.r.Registry.Integrations["bitbucket"] = provider }()
	requests := pendingActionRequests(f.t, f.installation.ID, bbReviewHook)
	require.Len(f.t, requests, 1)
	require.NoError(f.t, f.db.Model(&requests[0]).Update("run_at", time.Now().Add(-time.Second)).Error)
	return NewIntegrationRequestWorker(f.r.Encryptor, f.r.Registry, nil, "", "").LockAndProcessRequest(requests[0])
}

func TestBitbucketReviewCheckTracksApprovalsAfterSelection(t *testing.T) {
	f := newBitbucketReviewFixture(t)
	require.NoError(t, f.apply("pending_selection"))
	require.NoError(t, f.refresh())
	require.Equal(t, "INPROGRESS", f.state())
	f.ensure()
	// The authenticated approval predates selection completion.
	f.event("pullrequest:approved", f.http.head, f.http.approval)
	require.NoError(t, f.apply("required_reviewers"))
	f.http.participant = `"approved":true,"state":"approved"`
	require.NoError(t, f.refresh())
	require.Equal(t, "SUCCESSFUL", f.state())
	write := f.http.writes[len(f.http.writes)-1]
	require.Equal(t, "superplane/review-request", write["key"])
	require.Equal(t, "topic", write["refname"])
	require.Contains(t, write["path"], f.http.head)
	require.Contains(t, write["path"], bbReviewRepository)
	require.Contains(t, write["url"], f.ctx.RunID.String())
	// Missing events must not hide revocation or changes requests.
	for _, test := range []struct{ participant, want string }{
		{`"approved":false`, "INPROGRESS"},
		{`"approved":false,"state":"changes_requested"`, "FAILED"},
		{`"approved":false`, "INPROGRESS"},
		{`"approved":true,"state":"approved"`, "SUCCESSFUL"},
	} {
		f.http.participant = test.participant
		require.NoError(t, f.refresh())
		require.Equal(t, test.want, f.state())
	}
	// Comments cannot revoke approval or generate another status write.
	count := len(f.http.writes)
	f.event("pullrequest:comment_created", f.http.head, time.Now())
	require.NoError(t, f.refresh())
	require.Len(t, f.http.writes, count)
	oldSHA := f.http.head
	f.http.head = strings.Repeat("b", 40)
	f.event("repo:push", f.http.head, time.Now())
	require.NoError(t, f.refresh())
	require.Equal(t, "INPROGRESS", f.state())
	f.http.head = oldSHA
	require.NoError(t, f.refresh())
	require.Equal(t, "INPROGRESS", f.state(), "returning to the old commit cannot restore an invalidated decision")
	f.http.head = strings.Repeat("b", 40)
	// A new attempt on the new head still cannot use the old approval.
	root := support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil)
	f.attempt(root)
	f.ctx.Integration, f.ctx.HTTP = f.integration, f.http
	f.ctx.Configuration = map[string]any{"repository": "acme/widgets", "pullRequestId": "42", "assessedSha": f.http.head, "phase": "pending_selection", "reason": "Review required"}
	require.NoError(t, f.apply("pending_selection"))
	f.ensure()
	require.NoError(t, f.apply("required_reviewers"))
	f.event("pullrequest:approved", oldSHA, f.http.approval)
	require.NoError(t, f.refresh()) // Release the previous commit before applying the new check.
	require.NoError(t, f.refresh())
	require.Equal(t, "INPROGRESS", f.state())
	f.http.approval = f.http.approval.Add(time.Second)
	f.event("pullrequest:approved", f.http.head, f.http.approval)
	f.event("pullrequest:approved", oldSHA, f.http.approval.Add(-time.Second))
	require.NoError(t, f.refresh())
	require.Equal(t, "SUCCESSFUL", f.state())
}

func TestBitbucketReviewCheckRecoversDeliveryAndDisable(t *testing.T) {
	f := newBitbucketReviewFixture(t)
	require.NoError(t, f.apply("no_review"))
	f.http.failWrite = true
	require.Error(t, f.refresh())
	require.NotNil(t, f.record().WakeAt)
	f.http.failWrite = false
	require.NoError(t, f.refresh())
	require.Equal(t, "SUCCESSFUL", f.state())
	f.finishAction() // Run the real action hook and finish the ordinary-canvas step.
	// An interruption leaves a committed lease and durable recovery request.
	require.NoError(t, f.integration.store.Update(bbReviewKey, func(record *core.OrderedPublication) error {
		var payload map[string]any
		require.NoError(t, json.Unmarshal(record.Payload, &payload))
		payload["delivered"] = ""
		payload["delivery"] = core.PublicationLease{ID: uuid.New(), ExpiresAt: time.Now().Add(-time.Minute)}
		record.Payload, _ = json.Marshal(payload)
		return nil
	}))
	require.NoError(t, f.refresh())
	require.Equal(t, "SUCCESSFUL", f.state())
	require.NoError(t, (&bitbucket.SetPullRequestReviewCheck{}).Cleanup(core.SetupContext{WorkflowID: f.canvas.ID.String(), NodeID: "check", Integration: f.integration}))
	require.True(t, f.record().Disabled)
	f.http.failRead = true // Release does not require the removed PR.
	require.NoError(t, f.refresh())
	require.Equal(t, "SUCCESSFUL", f.state())
	require.Nil(t, f.record().WakeAt)
	f.http.failRead = false
	require.NoError(t, f.apply("no_review"))
	require.True(t, f.record().Disabled, "obsolete callback cannot reopen the disabled activation")
}

func TestBitbucketReviewCheckRechecksAfterLookupAndWrite(t *testing.T) {
	f := newBitbucketReviewFixture(t)
	require.NoError(t, f.apply("no_review"))
	f.http.beforeRead = func() { require.NoError(t, f.apply("disabled")) }
	require.NoError(t, f.refresh())
	require.Empty(t, f.http.writes, "replacement wins before a write is claimed")
	require.NoError(t, f.refresh())
	require.Equal(t, "SUCCESSFUL", f.state())
}

func TestBitbucketReviewCheckFencesActivationAndInFlightDisable(t *testing.T) {
	f := newBitbucketReviewFixture(t)
	require.NoError(t, f.apply("pending_selection"))
	require.NoError(t, f.refresh())
	require.NoError(t, f.apply("selection_error"))
	f.http.beforeWrite = func() { f.saveActivation(false, 1); require.NoError(t, f.refresh()) }
	require.NoError(t, f.refresh())
	require.NotNil(t, f.record().WakeAt, "late completion must retain release delivery")
	require.NoError(t, f.refresh())
	require.Equal(t, "SUCCESSFUL", f.state())
	old := f.ctx
	f.saveActivation(true, 2)
	require.NoError(t, f.apply("no_review"))
	require.True(t, f.record().Disabled, "reenabling cannot admit an old attempt")
	root := support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil)
	f.attempt(root)
	f.ctx.Integration, f.ctx.HTTP = f.integration, f.http
	f.ctx.Configuration = map[string]any{"repository": "acme/widgets", "pullRequestId": "42", "assessedSha": f.http.head, "reason": "No review needed"}
	require.NoError(t, f.apply("no_review"))
	require.NoError(t, f.refresh())
	require.EqualValues(t, 2, f.record().ActivationGeneration)
	generation := f.record().Generation
	require.NoError(t, (&bitbucket.SetPullRequestReviewCheck{}).Execute(old))
	require.Equal(t, generation, f.record().Generation)
}

func TestBitbucketReviewCheckRetriesFailedHooksAndBoundsFailures(t *testing.T) {
	f := newBitbucketReviewFixture(t)
	require.NoError(t, f.apply("no_review"))
	f.http.failWrite = true
	require.Error(t, f.processDue())
	require.Len(t, pendingActionRequests(t, f.installation.ID, bbReviewHook), 1, "failed integration hooks retain recovery work")
	f.http.failWrite = false
	require.NoError(t, f.processDue())
	require.Equal(t, "SUCCESSFUL", f.state())
	// Repeated lookup failure after a delivered status must also be bounded.
	f.http.failRead = true
	for i := 0; i < 10; i++ {
		_ = f.refresh()
	}
	var state struct {
		Error  string
		Reason string
	}
	require.NoError(t, json.Unmarshal(f.record().Payload, &state))
	require.Equal(t, "lookup_error", state.Reason)
	require.NotEmpty(t, state.Error)
	require.Nil(t, f.record().WakeAt)
}

func TestBitbucketReviewCheckEndsFailedSelectionAndReopensTracking(t *testing.T) {
	f := newBitbucketReviewFixture(t)
	require.NoError(t, f.apply("pending_selection"))
	require.NoError(t, f.refresh())
	failed := support.CreateCanvasNodeExecution(t, f.canvas.ID, "ensure", f.root.ID, f.root.ID)
	require.NoError(t, workercontexts.NewExecutionStateContext(f.db, failed, nil).Fail(models.CanvasNodeExecutionResultReasonError, "Selection failed without a failed output"))
	require.NoError(t, f.refresh())
	require.Equal(t, "FAILED", f.state())
	// Head changes cannot hide an attempt that ended without a decision.
	f.http.head = strings.Repeat("b", 40)
	require.NoError(t, f.refresh())
	require.Equal(t, "FAILED", f.state())
	// A separate decision can complete, close, and resume without another agent.
	root := support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil)
	f.attempt(root)
	f.ctx.Integration, f.ctx.HTTP = f.integration, f.http
	f.ctx.Configuration = map[string]any{"repository": "acme/widgets", "pullRequestId": "42", "assessedSha": f.http.head, "reason": "No review needed"}
	require.NoError(t, f.apply("no_review"))
	f.http.state = "DECLINED"
	require.NoError(t, f.refresh())
	require.Equal(t, "SUCCESSFUL", f.state())
	require.NotNil(t, f.record().WakeAt, "a missed reopening event needs periodic recovery")
	f.http.state = "OPEN"
	f.event("pullrequest:updated", f.http.head, time.Now())
	require.NoError(t, f.refresh())
	require.NotNil(t, f.record().WakeAt)
}

func TestBitbucketReviewCheckExplicitDisableUsesSavedSource(t *testing.T) {
	f := newBitbucketReviewFixture(t)
	require.NoError(t, f.apply("pending_selection"))
	require.NoError(t, f.refresh())
	require.Equal(t, "INPROGRESS", f.state())
	publishedSHA := f.http.head
	f.ctx.Configuration.(map[string]any)["assessedSha"] = strings.Repeat("b", 40)
	require.NoError(t, f.apply("disabled"))
	f.http.failRead = true
	require.NoError(t, f.refresh())
	require.Equal(t, "SUCCESSFUL", f.state())
	write := f.http.writes[len(f.http.writes)-1]
	require.Contains(t, write["path"], publishedSHA)
	require.Equal(t, "topic", write["refname"])
	require.Nil(t, f.record().WakeAt)
}

func TestBitbucketReviewCheckNormalizesPullRequestIdentity(t *testing.T) {
	for _, id := range []string{"042", "42 ", " 0042 "} {
		t.Run(id, func(t *testing.T) {
			f := newBitbucketReviewFixture(t)
			f.ctx.Configuration.(map[string]any)["pullRequestId"] = id
			require.NoError(t, f.apply("pending_selection"))
			require.NoError(t, f.refresh())
			f.ensure()
			f.ctx.Configuration.(map[string]any)["pullRequestId"] = "42"
			require.NoError(t, f.apply("required_reviewers"))
			records, err := f.integration.store.List("bitbucket.review-check/")
			require.NoError(t, err)
			require.Len(t, records, 1)
			require.EqualValues(t, 2, f.record().Generation)
			require.NoError(t, f.integration.store.Update(bbReviewKey, func(record *core.OrderedPublication) error {
				record.WakeAt = nil
				return nil
			}))
			f.event("pullrequest:approved", f.http.head, f.http.approval)
			require.NotNil(t, f.record().WakeAt)
			f.http.participant = `"approved":true,"state":"approved"`
			require.NoError(t, f.refresh())
			require.Equal(t, "SUCCESSFUL", f.state())
		})
	}
}

func TestBitbucketReviewCheckKeepsWritesOnAssessedCommit(t *testing.T) {
	f := newBitbucketReviewFixture(t)
	require.NoError(t, f.apply("no_review"))
	assessed := f.http.head
	f.http.head = strings.Repeat("b", 40)
	require.NoError(t, f.refresh())
	require.Equal(t, "INPROGRESS", f.state())
	require.Contains(t, f.http.writes[len(f.http.writes)-1]["path"], assessed)
}

func TestBitbucketReviewCheckReleasesSupersededPendingCommit(t *testing.T) {
	f := newBitbucketReviewFixture(t)
	require.NoError(t, f.apply("pending_selection"))
	require.NoError(t, f.refresh())
	previous := f.http.head
	root := support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil)
	f.attempt(root)
	f.http.head = strings.Repeat("b", 40)
	f.ctx.Integration, f.ctx.HTTP = f.integration, f.http
	f.ctx.Configuration = map[string]any{"repository": "acme/widgets", "pullRequestId": "42", "assessedSha": f.http.head, "reason": "Review required"}
	require.NoError(t, f.apply("pending_selection"))
	require.NoError(t, f.refresh())
	write := f.http.writes[len(f.http.writes)-1]
	require.Equal(t, "SUCCESSFUL", write["state"], "the old pending check must be released")
	require.Contains(t, write["path"], previous)
	require.NoError(t, f.refresh())
	require.Equal(t, "INPROGRESS", f.state())
	require.Contains(t, f.http.writes[len(f.http.writes)-1]["path"], f.http.head)
}

func TestBitbucketReviewCheckPublishesInitialLookupFailure(t *testing.T) {
	f := newBitbucketReviewFixture(t)
	require.NoError(t, f.apply("pending_selection"))
	f.http.failRead = true
	require.Error(t, f.refresh())
	require.Equal(t, "FAILED", f.state())
	require.NotNil(t, f.record().WakeAt)
}

func TestBitbucketReviewCheckInvalidatesHeadObservedDuringReviewReads(t *testing.T) {
	f := newBitbucketReviewFixture(t)
	f.ensure()
	require.NoError(t, f.apply("required_reviewers"))
	f.http.participant = `"approved":true,"state":"approved"`
	original := f.http.head
	f.event("pullrequest:approved", original, f.http.approval)
	f.http.heads = []string{original, strings.Repeat("b", 40), original}
	require.NoError(t, f.refresh())
	require.Equal(t, "INPROGRESS", f.state())
	require.NoError(t, f.refresh())
	require.Equal(t, "INPROGRESS", f.state(), "an observed head change requires another decision even after the old commit returns")
}

func TestBitbucketReviewCheckRejectsConflictingApprovalReceipts(t *testing.T) {
	f := newBitbucketReviewFixture(t)
	f.ensure()
	require.NoError(t, f.apply("required_reviewers"))
	f.http.participant = `"approved":true,"state":"approved"`
	f.event("pullrequest:approved", f.http.head, f.http.approval)
	f.event("pullrequest:approved", strings.Repeat("b", 40), f.http.approval)
	require.NoError(t, f.refresh())
	require.Equal(t, "INPROGRESS", f.state())
	f.event("pullrequest:approved", f.http.head, f.http.approval)
	require.NoError(t, f.refresh())
	require.Equal(t, "INPROGRESS", f.state(), "a duplicate cannot clear conflicting commit evidence")
}

func TestBitbucketReviewCheckDoesNotReleaseTheCurrentCommit(t *testing.T) {
	f := newBitbucketReviewFixture(t)
	original := f.http.head
	require.NoError(t, f.apply("pending_selection"))
	require.NoError(t, f.refresh())
	for _, head := range []string{strings.Repeat("b", 40), original} {
		root := support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil)
		f.attempt(root)
		f.ctx.Integration, f.ctx.HTTP = f.integration, f.http
		f.http.head = head
		f.ctx.Configuration = map[string]any{"repository": "acme/widgets", "pullRequestId": "42", "assessedSha": head, "reason": "Review required"}
		require.NoError(t, f.apply("pending_selection"))
	}
	writes := len(f.http.writes)
	for range 3 {
		require.NoError(t, f.refresh())
	}
	require.Greater(t, len(f.http.writes), writes)
	require.Equal(t, "INPROGRESS", f.state())
	for _, write := range f.http.writes[writes:] {
		if strings.Contains(write["path"].(string), original) {
			require.Equal(t, "INPROGRESS", write["state"], "an active commit must not receive a superseded release")
		}
	}
}

func TestBitbucketReviewCheckDisableReleasesQueuedCommits(t *testing.T) {
	f := newBitbucketReviewFixture(t)
	original := f.http.head
	require.NoError(t, f.apply("pending_selection"))
	require.NoError(t, f.refresh())
	other := strings.Repeat("b", 40)
	for _, head := range []string{other, original} {
		root := support.EmitCanvasEventForNode(t, f.canvas.ID, "trigger", "default", nil)
		f.attempt(root)
		f.ctx.Integration, f.ctx.HTTP = f.integration, f.http
		f.http.head = head
		f.ctx.Configuration = map[string]any{"repository": "acme/widgets", "pullRequestId": "42", "assessedSha": head, "reason": "Review required"}
		require.NoError(t, f.apply("pending_selection"))
		if head == other {
			require.NoError(t, f.refresh())
			require.NoError(t, f.refresh())
		}
	}
	f.ctx.Configuration.(map[string]any)["assessedSha"] = other
	require.NoError(t, f.apply("disabled"))
	f.http.failRead = true
	for range 3 {
		require.NoError(t, f.refresh())
	}
	lastState := map[string]any{}
	for _, write := range f.http.writes {
		lastState[write["path"].(string)] = write["state"]
	}
	require.Len(t, lastState, 2)
	for _, state := range lastState {
		require.Equal(t, "SUCCESSFUL", state)
	}
	require.Nil(t, f.record().WakeAt)
}

func TestBitbucketReviewCheckRejectsAReusedRepositoryName(t *testing.T) {
	f := newBitbucketReviewFixture(t)
	f.ensure()
	// The repository name can stay the same while the provider UUID changes.
	var event models.CanvasEvent
	require.NoError(t, f.db.Where("execution_id = ?", f.ctx.Configuration.(map[string]any)["reviewerRequestExecutionId"]).First(&event).Error)
	envelope := event.Data.Data().(map[string]any)
	envelope["data"].(map[string]any)["repositoryId"] = uuid.NewString()
	event.Data = models.NewJSONValue(envelope)
	require.NoError(t, f.db.Save(&event).Error)
	require.ErrorContains(t, f.apply("required_reviewers"), "different publication scope")
	record, err := f.integration.store.Get(bbReviewKey)
	require.NoError(t, err)
	require.Nil(t, record, "an outcome for a different repository cannot register a check")
}
