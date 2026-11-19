package pulls

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

type reviewerPublications struct {
	core.PublicationContext
	records map[string]core.OrderedPublication
	order   core.PublicationOrder
}

func (s *reviewerPublications) ResolveOrder(uuid.UUID, uuid.UUID) (core.PublicationOrder, error) {
	return s.order, nil
}
func (s *reviewerPublications) Get(key string) (*core.OrderedPublication, error) {
	r, ok := s.records[key]
	if !ok {
		return nil, nil
	}
	return &r, nil
}
func (s *reviewerPublications) Update(key string, apply func(*core.OrderedPublication) error) error {
	r := s.records[key]
	r.Key = key
	if err := apply(&r); err != nil {
		if errors.Is(err, core.ErrPublicationUnchanged) {
			return nil
		}
		return err
	}
	s.records[key] = r
	return nil
}

type reviewerHost struct {
	pending                           []ReviewerIdentity
	teams                             []string
	mutations                         []string
	failAdd, failRemove, appliedError bool
	moved, approved                   bool
	commented                         bool
	onAdd                             func()
}

func (h *reviewerHost) Do(req *http.Request) (*http.Response, error) {
	path := req.URL.Path
	body := any(map[string]any{})
	switch {
	case strings.HasSuffix(path, "/requested_reviewers"):
		var request struct {
			Reviewers     []string `json:"reviewers"`
			TeamReviewers []string `json:"team_reviewers"`
		}
		if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
			return nil, err
		}
		if req.Method == http.MethodDelete {
			h.teams = slices.DeleteFunc(h.teams, func(team string) bool { return slices.Contains(request.TeamReviewers, team) })
		}
		for _, login := range request.Reviewers {
			h.mutations = append(h.mutations, req.Method+" "+login)
			if req.Method == http.MethodPost {
				if h.failAdd && !h.appliedError {
					return mocks.GitHubResponse(503, `{"message":"failed"}`), nil
				}
				id := int64(2)
				if login == "second" {
					id = 3
				}
				h.pending = append(h.pending, ReviewerIdentity{ID: id, Login: login})
				if h.onAdd != nil {
					callback := h.onAdd
					h.onAdd = nil
					callback()
				}
				if h.failAdd {
					return mocks.GitHubResponse(503, `{"message":"response lost"}`), nil
				}
			} else {
				if h.failRemove && !h.appliedError {
					return mocks.GitHubResponse(503, `{"message":"failed"}`), nil
				}
				h.pending = slices.DeleteFunc(h.pending, func(r ReviewerIdentity) bool { return r.Login == login })
				if h.failRemove {
					return mocks.GitHubResponse(503, `{"message":"response lost"}`), nil
				}
			}
		}
	case strings.HasSuffix(path, "/collaborators"):
		body = []map[string]any{{"id": 2, "login": "first", "type": "User", "permissions": map[string]bool{"push": true}}, {"id": 3, "login": "second", "type": "User", "permissions": map[string]bool{"push": true}}}
	case strings.HasSuffix(path, "/permission"):
		body = map[string]any{"permission": "write"}
	case strings.HasSuffix(path, "/reviews"):
		body = []any{}
		if h.approved {
			body = []any{map[string]any{"id": 1, "user": ReviewerIdentity{ID: 2, Login: "first"}, "state": "APPROVED", "commit_id": "abc", "submitted_at": time.Now().Format(time.RFC3339)}}
		}
		if h.commented {
			body = []any{map[string]any{"id": 2, "user": ReviewerIdentity{ID: 2, Login: "first"}, "state": "COMMENTED", "commit_id": "abc", "submitted_at": time.Now().Format(time.RFC3339)}}
		}
	case strings.HasSuffix(path, "/pulls/42"):
		sha := "abc"
		if h.moved {
			sha = "moved"
		}
		body = map[string]any{"state": "open", "head": map[string]string{"sha": sha}, "user": map[string]any{"id": 1, "login": "author"}, "requested_reviewers": h.pending, "requested_teams": mapTeams(h.teams)}
	default:
		return nil, fmt.Errorf("unexpected request %s %s", req.Method, path)
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return mocks.GitHubResponse(200, string(encoded)), nil
}
func mapTeams(slugs []string) []map[string]string {
	teams := []map[string]string{}
	for _, slug := range slugs {
		teams = append(teams, map[string]string{"slug": slug})
	}
	return teams
}
func reviewerExecution(store *reviewerPublications, host *reviewerHost, reviewers ...ReviewerIdentity) core.ExecutionContext {
	return core.ExecutionContext{ID: uuid.New(), WorkflowID: "11111111-1111-1111-1111-111111111111", NodeID: "ensure", Configuration: map[string]any{"repository": "hello", "pullNumber": 42, "managed": true, "expectedHeadSha": "abc", "reviewers": reviewers}, Publications: store, Metadata: &contexts.MetadataContext{}, Requests: &contexts.RequestContext{}, Integration: mocks.IntegrationContextForNewSetupFlow(), HTTP: host, ExecutionState: &contexts.ExecutionStateContext{}}
}
func newReviewerPublications() *reviewerPublications {
	return &reviewerPublications{records: map[string]core.OrderedPublication{}, order: core.PublicationOrder{SourceRunID: uuid.New(), SourceCreatedAt: time.Now(), AttemptRootEventID: uuid.New(), AttemptCreatedAt: time.Now()}}
}
func TestManagedReviewersReplaceOnlyOwnedRequests(t *testing.T) {
	store := newReviewerPublications()
	host := &reviewerHost{pending: []ReviewerIdentity{{ID: 9, Login: "manual"}}, teams: []string{"manual-team"}}
	first := ReviewerIdentity{ID: 2, Login: "first"}
	second := ReviewerIdentity{ID: 3, Login: "second"}
	run := func(reviewers ...ReviewerIdentity) {
		t.Helper()
		require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host, reviewers...)))
	}
	run(first)
	run(first)
	require.Equal(t, []string{"POST first"}, host.mutations)
	run(second)
	require.Equal(t, []string{"POST first", "POST second", "DELETE first"}, host.mutations)
	require.ElementsMatch(t, []ReviewerIdentity{{ID: 9, Login: "manual"}, second}, host.pending)
	run()
	require.Equal(t, []ReviewerIdentity{{ID: 9, Login: "manual"}}, host.pending)
	require.Equal(t, []string{"manual-team"}, host.teams)
}
func TestManagedReviewersDoNotAdoptExistingRequestsOrApprovals(t *testing.T) {
	for _, approved := range []bool{false, true} {
		t.Run(fmt.Sprint(approved), func(t *testing.T) {
			store := newReviewerPublications()
			host := &reviewerHost{approved: approved}
			first := ReviewerIdentity{ID: 2, Login: "first"}
			if !approved {
				host.pending = []ReviewerIdentity{first}
			}
			require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host, first)))
			require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host)))
			require.Empty(t, host.mutations)
		})
	}
}

func TestManagedReviewersRetryPartialGitHubFailures(t *testing.T) {
	for _, failure := range []string{"add rejected", "add response lost", "withdraw failed"} {
		t.Run(failure, func(t *testing.T) {
			store := newReviewerPublications()
			host := &reviewerHost{}
			first := ReviewerIdentity{ID: 2, Login: "first"}
			second := ReviewerIdentity{ID: 3, Login: "second"}
			require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host, first)))
			host.failAdd = failure != "withdraw failed"
			host.appliedError = failure == "add response lost"
			host.failRemove = failure == "withdraw failed"
			failed := reviewerExecution(store, host, second)
			require.Error(t, (&EnsurePullRequestReviewers{}).Execute(failed))
			require.Empty(t, failed.ExecutionState.(*contexts.ExecutionStateContext).Payloads)
			require.Contains(t, host.pending, first, "a failed replacement must retain the previous request")
			host.failAdd, host.failRemove = false, false
			require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host, second)))
			if failure == "withdraw failed" {
				require.ElementsMatch(t, []ReviewerIdentity{first, second}, host.pending)
			} else {
				require.Equal(t, []ReviewerIdentity{second}, host.pending)
			}
			err := (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host))
			if failure == "withdraw failed" {
				require.ErrorContains(t, err, "uncertain withdrawal")
			} else {
				require.NoError(t, err)
			}
			if failure == "withdraw failed" {
				require.Equal(t, []ReviewerIdentity{first}, host.pending)
			} else {
				require.Empty(t, host.pending, "retry must preserve ownership of additions whose response was lost")
			}
		})
	}
}

func TestManagedReviewersFenceOverlappingRunsAndOldReportRetries(t *testing.T) {
	store := newReviewerPublications()
	host := &reviewerHost{}
	first := ReviewerIdentity{ID: 2, Login: "first"}
	second := ReviewerIdentity{ID: 3, Login: "second"}
	originalOrder := store.order
	var overlapErr error
	var newer core.ExecutionContext
	host.onAdd = func() {
		store.order.SourceRunID = uuid.New()
		store.order.SourceCreatedAt = store.order.SourceCreatedAt.Add(time.Minute)
		newer = reviewerExecution(store, host, second)
		overlapErr = (&EnsurePullRequestReviewers{}).Execute(newer)
	}
	older := reviewerExecution(store, host, first)
	require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(older))
	require.NoError(t, overlapErr)
	require.NotEmpty(t, newer.Requests.(*contexts.RequestContext).Action)
	require.Empty(t, newer.ExecutionState.(*contexts.ExecutionStateContext).Payloads)
	envelope := older.ExecutionState.(*contexts.ExecutionStateContext).Payloads[0].(map[string]any)
	require.Equal(t, "skipped", envelope["data"].(ensureReviewersOutcome).Status)
	require.NoError(t, resumeReviewers(newer))
	require.Equal(t, []ReviewerIdentity{second}, host.pending)
	mutations := slices.Clone(host.mutations)
	store.order = originalOrder
	require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host, first)))
	require.Equal(t, mutations, host.mutations, "an old report must not restore its reviewer")
}

func TestManagedReviewersPreserveRequestsAtStaleCommit(t *testing.T) {
	for _, duringAdd := range []bool{false, true} {
		t.Run(fmt.Sprint(duringAdd), func(t *testing.T) {
			store := newReviewerPublications()
			host := &reviewerHost{}
			first := ReviewerIdentity{ID: 2, Login: "first"}
			second := ReviewerIdentity{ID: 3, Login: "second"}
			require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host, first)))
			if duringAdd {
				host.onAdd = func() { host.moved = true }
			} else {
				host.moved = true
			}
			require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host, second)))
			require.Contains(t, host.pending, first)
			require.NotContains(t, host.mutations, "DELETE first")
		})
	}
}

func resumeReviewers(ctx core.ExecutionContext) error {
	return (&EnsurePullRequestReviewers{}).HandleHook(core.ActionHookContext{Name: ctx.Requests.(*contexts.RequestContext).Action, Configuration: ctx.Configuration, Metadata: ctx.Metadata, Requests: ctx.Requests, Publications: ctx.Publications, Integration: ctx.Integration, HTTP: ctx.HTTP, ExecutionState: ctx.ExecutionState})
}

func TestManagedReviewersLeaveManualRerequestsAfterSubmittedReviews(t *testing.T) {
	for _, state := range []string{"APPROVED", "COMMENTED"} {
		t.Run(state, func(t *testing.T) {
			store := newReviewerPublications()
			host := &reviewerHost{}
			first := ReviewerIdentity{ID: 2, Login: "first"}
			require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host, first)))
			host.approved, host.commented = state == "APPROVED", state == "COMMENTED"
			require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host)))
			require.Equal(t, []ReviewerIdentity{first}, host.pending)
			require.Equal(t, []string{"POST first"}, host.mutations)
		})
	}
}

func TestManagedReviewersPreserveRequestsAfterUncertainWithdrawal(t *testing.T) {
	for _, applied := range []bool{false, true} {
		t.Run(fmt.Sprint(applied), func(t *testing.T) {
			store := newReviewerPublications()
			host := &reviewerHost{}
			first := ReviewerIdentity{ID: 2, Login: "first"}
			require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host, first)))
			host.failRemove, host.appliedError = true, applied
			require.Error(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host)))
			// A person re-requests the reviewer after GitHub applied the DELETE.
			host.pending = []ReviewerIdentity{first}
			host.failRemove = false
			require.ErrorContains(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host)), "uncertain withdrawal")
			require.Equal(t, []ReviewerIdentity{first}, host.pending)
			require.Equal(t, []string{"POST first", "DELETE first"}, host.mutations)
		})
	}
}

func TestReviewCandidateDiscoveryIncludesPendingAndPreviousSelection(t *testing.T) {
	store := newReviewerPublications()
	host := &reviewerHost{pending: []ReviewerIdentity{{ID: 9, Login: "manual"}}, teams: []string{"manual-team"}}
	first := ReviewerIdentity{ID: 2, Login: "first", Reason: "Owns this code."}
	require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host, first)))
	ctx := reviewerExecution(store, host)
	require.NoError(t, (&ListPullRequestReviewCandidates{}).Execute(ctx))
	envelope := ctx.ExecutionState.(*contexts.ExecutionStateContext).Payloads[0].(map[string]any)
	result := envelope["data"].(map[string]any)
	require.Equal(t, []ReviewerIdentity{first}, result["previousSelection"])
	require.Len(t, result["requestedReviewers"], 2)
	require.NotEmpty(t, result["requestedTeams"])
}

func TestManagedReviewersRespectAnotherAutomationsOwnership(t *testing.T) {
	store := newReviewerPublications()
	host := &reviewerHost{}
	first := ReviewerIdentity{ID: 2, Login: "first"}
	require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host, first)))
	other := reviewerExecution(store, host, first)
	other.WorkflowID = uuid.NewString()
	require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(other))
	other.Configuration.(map[string]any)["reviewers"] = []ReviewerIdentity{}
	require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(other))
	require.Equal(t, []ReviewerIdentity{first}, host.pending)
	require.Equal(t, []string{"POST first"}, host.mutations)
}

func TestManagedReviewersExpiredWriterCannotRestoreOwnership(t *testing.T) {
	store := newReviewerPublications()
	host := &reviewerHost{}
	first := ReviewerIdentity{ID: 2, Login: "first"}
	second := ReviewerIdentity{ID: 3, Login: "second"}
	host.onAdd = func() {
		// Simulate a paused writer resuming after its leases expired and a
		// successor reconciled. Its late response must not overwrite new state.
		for key, record := range store.records {
			var payload managedReviewerPayload
			require.NoError(t, json.Unmarshal(record.Payload, &payload))
			if payload.Lease == nil {
				continue
			}
			payload.Lease.ExpiresAt = time.Now().Add(-time.Second)
			record.Payload, _ = json.Marshal(payload)
			store.records[key] = record
		}
		store.order.SourceRunID = uuid.New()
		store.order.SourceCreatedAt = store.order.SourceCreatedAt.Add(time.Minute)
		require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host, second)))
	}
	older := reviewerExecution(store, host, first)
	require.ErrorIs(t, (&EnsurePullRequestReviewers{}).Execute(older), core.ErrPublicationSuperseded)
	require.Empty(t, older.ExecutionState.(*contexts.ExecutionStateContext).Payloads)
	require.Equal(t, []ReviewerIdentity{second}, host.pending)
	require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host)))
	require.Empty(t, host.pending, "the superseded writer must not lose the successor's ownership")
}

func TestManagedReviewersRetrySharedLeaseContention(t *testing.T) {
	for _, superseded := range []bool{false, true} {
		t.Run(fmt.Sprint(superseded), func(t *testing.T) {
			store := newReviewerPublications()
			host := &reviewerHost{}
			key := "github.review-requests-lock/testhq/hello/42"
			lease, err := core.BeginPublicationLease(store, key, time.Minute)
			require.NoError(t, err)
			first := ReviewerIdentity{ID: 2, Login: "first"}
			waiting := reviewerExecution(store, host, first)
			require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(waiting))
			require.NotEmpty(t, waiting.Requests.(*contexts.RequestContext).Action)
			require.NoError(t, resumeReviewers(waiting))
			require.Empty(t, host.mutations)
			require.Empty(t, waiting.ExecutionState.(*contexts.ExecutionStateContext).Payloads)
			require.NoError(t, core.FinishPublicationLease(store, key, lease))
			if superseded {
				store.order.SourceRunID = uuid.New()
				store.order.SourceCreatedAt = store.order.SourceCreatedAt.Add(time.Minute)
				require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host)))
			}
			require.NoError(t, resumeReviewers(waiting))
			outcome := waiting.ExecutionState.(*contexts.ExecutionStateContext).Payloads[0].(map[string]any)["data"].(ensureReviewersOutcome)
			require.Equal(t, waiting.ID.String(), outcome.ExecutionID)
			if superseded {
				require.Empty(t, host.pending)
				require.Equal(t, "superseded", outcome.Reason)
			} else {
				require.Equal(t, []ReviewerIdentity{first}, host.pending)
				require.Equal(t, "requested", outcome.Status)
			}
		})
	}
}

func TestManagedReviewersRecoverWithdrawalExplanation(t *testing.T) {
	store := newReviewerPublications()
	host := &reviewerHost{}
	first := ReviewerIdentity{ID: 2, Login: "first"}
	require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host, first)))
	host.failRemove, host.appliedError = true, true
	require.Error(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host)))
	host.failRemove = false
	for range 2 {
		ctx := reviewerExecution(store, host)
		require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(ctx))
		outcome := ctx.ExecutionState.(*contexts.ExecutionStateContext).Payloads[0].(map[string]any)["data"].(ensureReviewersOutcome)
		require.Equal(t, []ReviewerIdentity{first}, outcome.Removed)
		require.Len(t, outcome.Withdrawals, 1)
		require.Equal(t, "abc", outcome.Withdrawals[0].AssessedSHA)
	}
	require.Equal(t, []string{"POST first", "DELETE first"}, host.mutations)
}

func TestManagedReviewersRecoverUncertainWithdrawalAfterSubmission(t *testing.T) {
	for _, manualRequest := range []bool{false, true} {
		t.Run(fmt.Sprintf("manual_request=%t", manualRequest), func(t *testing.T) {
			store := newReviewerPublications()
			host := &reviewerHost{}
			first := ReviewerIdentity{ID: 2, Login: "first"}
			require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host, first)))
			host.failRemove = true
			require.Error(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host)))
			host.failRemove = false
			host.commented = true
			host.pending = nil
			if manualRequest {
				host.pending = []ReviewerIdentity{first}
				require.ErrorContains(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host)), "uncertain withdrawal")
				require.Equal(t, []ReviewerIdentity{first}, host.pending)
				host.pending = nil
			}
			for range 2 {
				ctx := reviewerExecution(store, host)
				require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(ctx))
				outcome := ctx.ExecutionState.(*contexts.ExecutionStateContext).Payloads[0].(map[string]any)["data"].(ensureReviewersOutcome)
				require.Empty(t, outcome.Withdrawals, "submission does not prove an automation withdrawal")
			}
			// A later manual request remains outside managed ownership.
			host.pending = []ReviewerIdentity{first}
			require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host, first)))
			require.NoError(t, (&EnsurePullRequestReviewers{}).Execute(reviewerExecution(store, host)))
			require.Equal(t, []ReviewerIdentity{first}, host.pending)
			require.Equal(t, []string{"POST first", "DELETE first"}, host.mutations)
		})
	}
}
