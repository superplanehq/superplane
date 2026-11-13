package bitbucket

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const reviewAuthor = "11111111-1111-1111-1111-111111111111"
const reviewHuman = "22222222-2222-2222-2222-222222222222"
const reviewOther = "33333333-3333-3333-3333-333333333333"

// Effective permissions, rather than explicit grants, must own reviewer eligibility.
func TestListEligibleReviewers(t *testing.T) {
	client, httpCtx := stubBitbucketClient(
		okResponse(`{"values":[{"permission":"write","user":{"uuid":"{`+reviewHuman+`}"}}],"next":"https://api.bitbucket.org/2.0/workspaces/acme/permissions/repositories/widgets?page=2"}`),
		okResponse(`{"values":[{"permission":"read","user":{"uuid":"{`+reviewOther+`}"}},{"permission":"admin","user":{"uuid":"{`+reviewAuthor+`}"}}]}`),
		okResponse(`{"type":"app_user","uuid":"{`+reviewAuthor+`}","account_status":"active"}`),
		okResponse(`{"type":"user","uuid":"{`+reviewHuman+`}","display_name":"Renamed Human","account_status":"active"}`),
	)
	reviewers, err := client.ListEligibleReviewers("acme/widgets")
	require.NoError(t, err)
	require.Len(t, reviewers, 1)
	require.Equal(t, reviewHuman, reviewers[0].UUID)
	require.Equal(t, "Renamed Human", reviewers[0].DisplayName)
	require.Equal(t, "/2.0/workspaces/acme/permissions/repositories/widgets", httpCtx.Requests[0].URL.Path)
}

func TestReviewerDiscoveryDoesNotHidePermissionErrors(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client, _ := stubBitbucketClient(bitbucketResponse(status, `{"error":{"message":"denied"}}`))
			_, err := client.ListEligibleReviewers("acme/widgets")
			require.ErrorContains(t, err, "repository admin")
		})
	}
	client, _ := stubBitbucketClient(okResponse(`{"values":[]}`))
	reviewers, err := client.ListEligibleReviewers("acme/widgets")
	require.NoError(t, err)
	require.Empty(t, reviewers)
}

func reviewSnapshot(head, state string, reviewers ...string) string {
	body := `{"id":42,"state":"` + state + `","author":{"uuid":"{` + reviewAuthor + `}"},"source":{"commit":{"hash":"` + head + `"}},"reviewers":[`
	for i, id := range reviewers {
		if i > 0 {
			body += ","
		}
		body += `{"uuid":"{` + id + `}"}`
	}
	return body + `],"participants":[]}`
}

func reviewPermissionPage() *http.Response {
	return okResponse(`{"values":[{"permission":"write","user":{"uuid":"{` + reviewHuman + `}"}}]}`)
}
func reviewUserResponse() *http.Response {
	return okResponse(`{"type":"user","account_status":"active","uuid":"{` + reviewHuman + `}","display_name":"Human"}`)
}

func TestEnsureReviewersPreservesUnrelatedRequestsAndRetries(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		t.Run(map[bool]string{false: "update", true: "conflict"}[conflict], func(t *testing.T) {
			responses := []*http.Response{okResponse(reviewSnapshot("abc", "OPEN", reviewOther)), reviewPermissionPage(), reviewUserResponse(), okResponse(reviewSnapshot("abc", "OPEN", reviewOther))}
			if conflict {
				responses = append(responses, bitbucketResponse(http.StatusConflict, `{}`), reviewPermissionPage(), reviewUserResponse(), okResponse(reviewSnapshot("abc", "OPEN", reviewOther)))
			}
			responses = append(responses, okResponse(`{}`), okResponse(reviewSnapshot("abc", "OPEN", reviewOther, reviewHuman)))
			responses[3].Header = http.Header{"Etag": []string{`"revision-1"`}}
			if conflict {
				responses[7].Header = http.Header{"Etag": []string{`"revision-2"`}}
			}
			client, httpCtx := stubBitbucketClient(responses...)
			result, err := client.ensurePullRequestReviewers("acme/widgets", 42, "abc", []ReviewerIdentity{{UUID: reviewHuman}})
			require.NoError(t, err)
			require.Equal(t, "requested", result.Status)
			for _, request := range httpCtx.Requests {
				if request.Method == http.MethodPut {
					require.NotEmpty(t, request.Header.Get("If-Match"))
					require.Equal(t, map[string]any{"reviewers": []any{map[string]any{"uuid": "{" + reviewHuman + "}"}, map[string]any{"uuid": "{" + reviewOther + "}"}}}, requestJSON(t, request))
				}
			}
		})
	}
}

func TestEnsureReviewersSkipsStalePullRequests(t *testing.T) {
	for _, snapshot := range []string{reviewSnapshot("new", "OPEN"), reviewSnapshot("abc", "MERGED"), reviewSnapshot("abc", "DECLINED")} {
		client, httpCtx := stubBitbucketClient(okResponse(snapshot))
		result, err := client.ensurePullRequestReviewers("acme/widgets", 42, "abc", []ReviewerIdentity{{UUID: reviewHuman}})
		require.NoError(t, err)
		require.Equal(t, "stale_pr", result.Status)
		require.Len(t, httpCtx.Requests, 1)
	}
}

func TestReviewerDiscoveryRejectsIncompleteResults(t *testing.T) {
	for _, page := range []string{
		`{}`,
		`{"values":[],"next":"https://untrusted.example/steal"}`,
		`{"values":[],"next":"https://api.bitbucket.org/2.0/workspaces/acme/permissions/repositories/widgets?pagelen=100"}`,
		`{"values":[{"permission":"write","user":{"uuid":"nickname"}}]}`,
		`{"values":[{"user":{"uuid":"{` + reviewHuman + `}"}}]}`,
	} {
		client, httpCtx := stubBitbucketClient(okResponse(page))
		_, err := client.ListEligibleReviewers("acme/widgets")
		require.Error(t, err)
		require.Len(t, httpCtx.Requests, 1)
	}
	client, _ := stubBitbucketClient(okResponse(`{"values":[],"next":"?page=2"}`), bitbucketResponse(http.StatusForbidden, `{}`))
	_, err := client.ListEligibleReviewers("acme/widgets")
	require.ErrorContains(t, err, "repository admin")
}

func TestReviewerAccountEligibility(t *testing.T) {
	cases := []struct {
		name, account string
		want          int
		errorText     string
	}{
		{"inactive", `{"type":"user","account_status":"inactive"}`, 0, ""},
		{"bot", `{"type":"app_user","account_status":"active"}`, 0, ""},
		{"named bot", `{"type":"user","account_status":"active","nickname":"build[bot]"}`, 0, ""},
		{"unknown activity", `{"type":"user"}`, 0, "active status"},
		{"unknown type", `{"account_status":"active"}`, 0, "account type"},
		{"active", `{"type":"user","account_status":"active","display_name":"Human"}`, 1, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			account := strings.TrimSuffix(tc.account, "}") + `,"uuid":"{` + reviewHuman + `}"}`
			client, _ := stubBitbucketClient(reviewPermissionPage(), okResponse(account))
			reviewers, err := client.ListEligibleReviewers("acme/widgets")
			if tc.errorText != "" {
				require.ErrorContains(t, err, tc.errorText)
				return
			}
			require.NoError(t, err)
			require.Len(t, reviewers, tc.want)
		})
	}
}

func TestReviewCandidatesUseRepositoryEvidenceAndExcludeAuthor(t *testing.T) {
	client, _ := stubBitbucketClient(
		okResponse(reviewSnapshot("abc", "OPEN")),
		okResponse(`{"values":[{"permission":"admin","user":{"uuid":"{`+reviewAuthor+`}"}},{"permission":"write","user":{"uuid":"{`+reviewHuman+`}"}}]}`),
		okResponse(`{"type":"user","account_status":"active","uuid":"{`+reviewAuthor+`}"}`), reviewUserResponse(),
		okResponse(`{"values":[],"next":"?page=2"}`), okResponse(`{"values":[{"reviewer_type":"project","user":{"uuid":"{`+reviewHuman+`}"}}]}`),
		okResponse(`{"values":[{"author":{"raw":"Unlinked author"}}],"next":"?page=2"}`), okResponse(`{"values":[{"author":{"user":{"uuid":"{`+reviewHuman+`}"}}}]}`),
		okResponse(reviewSnapshot("abc", "OPEN")),
	)
	candidates, err := client.ListPullRequestReviewCandidates("acme/widgets", 42)
	require.NoError(t, err)
	require.Equal(t, reviewAuthor, candidates.Author.UUID)
	require.Len(t, candidates.Candidates, 1)
	require.Equal(t, reviewHuman, candidates.Candidates[0].UUID)
	require.Equal(t, []string{"default_reviewer", "recent_contributor"}, candidates.Candidates[0].Evidence)
	require.Equal(t, 2, candidates.ContributorCommits)
	require.False(t, candidates.ContributorHistoryLimited)
}

func TestEnsureReviewersDoesNotClaimAnUncertainUpdate(t *testing.T) {
	cases := []struct {
		name      string
		responses []*http.Response
		errorText string
	}{
		{"provider failure", []*http.Response{bitbucketResponse(http.StatusInternalServerError, `{}`)}, "uncertain"},
		{"malformed accepted response", []*http.Response{okResponse(`invalid`)}, "uncertain"},
		{"verification failure", []*http.Response{okResponse(`{}`), bitbucketResponse(http.StatusForbidden, `{}`)}, "cannot verify"},
		{"malformed reviewer identity", []*http.Response{okResponse(`{}`), okResponse(reviewSnapshot("abc", "OPEN", reviewHuman, reviewOther, "nickname"))}, "cannot verify"},
		{"head changed during write", []*http.Response{okResponse(`{}`), okResponse(reviewSnapshot("new", "OPEN", reviewHuman, reviewOther))}, "changed during"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			responses := []*http.Response{okResponse(reviewSnapshot("abc", "OPEN", reviewOther)), reviewPermissionPage(), reviewUserResponse(), okResponse(reviewSnapshot("abc", "OPEN", reviewOther))}
			responses = append(responses, tc.responses...)
			client, _ := stubBitbucketClient(responses...)
			outcome, err := client.ensurePullRequestReviewers("acme/widgets", 42, "abc", []ReviewerIdentity{{UUID: reviewHuman}})
			require.ErrorContains(t, err, tc.errorText)
			require.Empty(t, outcome.Status)
		})
	}
}

func TestEnsureReviewersRechecksStateAndEligibilityBeforeWrite(t *testing.T) {
	cases := []struct {
		name              string
		permissions       *http.Response
		profile           *http.Response
		snapshot          string
		status, errorText string
	}{
		{"push during discovery", reviewPermissionPage(), reviewUserResponse(), reviewSnapshot("new", "OPEN"), "stale_pr", ""},
		{"closed during discovery", reviewPermissionPage(), reviewUserResponse(), reviewSnapshot("abc", "DECLINED"), "stale_pr", ""},
		{"lost permission", okResponse(`{"values":[]}`), nil, reviewSnapshot("abc", "OPEN"), "", "not an eligible"},
		{"author", okResponse(`{"values":[{"permission":"write","user":{"uuid":"{` + reviewAuthor + `}"}}]}`), okResponse(`{"type":"user","account_status":"active","uuid":"{` + reviewAuthor + `}"}`), reviewSnapshot("abc", "OPEN"), "", "not an eligible"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			responses := []*http.Response{okResponse(reviewSnapshot("abc", "OPEN")), tc.permissions}
			if tc.profile != nil {
				responses = append(responses, tc.profile)
			}
			responses = append(responses, okResponse(tc.snapshot))
			client, httpCtx := stubBitbucketClient(responses...)
			desired := reviewHuman
			if tc.name == "author" {
				desired = reviewAuthor
			}
			outcome, err := client.ensurePullRequestReviewers("acme/widgets", 42, "abc", []ReviewerIdentity{{UUID: desired}})
			if tc.errorText != "" {
				require.ErrorContains(t, err, tc.errorText)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.status, outcome.Status)
			}
			for _, request := range httpCtx.Requests {
				require.NotEqual(t, http.MethodPut, request.Method)
			}
		})
	}
}

func TestEnsureReviewersPreservesApprovalsWithoutInventingCommitEvidence(t *testing.T) {
	for _, retained := range []bool{false, true} {
		t.Run(map[bool]string{false: "approval lost", true: "approval retained"}[retained], func(t *testing.T) {
			withApproval := func(snapshot string) string {
				return strings.Replace(snapshot, `"participants":[]`, `"participants":[{"approved":true,"user":{"uuid":"{`+reviewOther+`}"}}]`, 1)
			}
			current := withApproval(reviewSnapshot("abc", "OPEN", reviewOther))
			verified := reviewSnapshot("abc", "OPEN", reviewHuman, reviewOther)
			if retained {
				verified = withApproval(verified)
			}
			client, _ := stubBitbucketClient(okResponse(current), reviewPermissionPage(), reviewUserResponse(), okResponse(current), okResponse(`{}`), okResponse(verified))
			outcome, err := client.ensurePullRequestReviewers("acme/widgets", 42, "abc", []ReviewerIdentity{{UUID: reviewHuman}})
			if !retained {
				require.ErrorContains(t, err, "approval changed")
				return
			}
			require.NoError(t, err)
			require.Equal(t, "requested", outcome.Status)
			require.Equal(t, []ReviewerIdentity{{UUID: reviewOther}}, outcome.UnresolvedApprovals)
		})
	}
}

func TestEnsureReviewersReconcilesConcurrentReviewerReplacement(t *testing.T) {
	client, httpCtx := stubBitbucketClient(
		okResponse(reviewSnapshot("abc", "OPEN", reviewOther)), reviewPermissionPage(), reviewUserResponse(), okResponse(reviewSnapshot("abc", "OPEN", reviewOther)), okResponse(`{}`),
		okResponse(reviewSnapshot("abc", "OPEN", reviewHuman)), // External replacement lost the unrelated reviewer.
		reviewPermissionPage(), reviewUserResponse(), okResponse(reviewSnapshot("abc", "OPEN", reviewHuman)), okResponse(`{}`), okResponse(reviewSnapshot("abc", "OPEN", reviewHuman, reviewOther)),
	)
	outcome, err := client.ensurePullRequestReviewers("acme/widgets", 42, "abc", []ReviewerIdentity{{UUID: reviewHuman}})
	require.NoError(t, err)
	require.Equal(t, "requested", outcome.Status)
	writes := 0
	for _, request := range httpCtx.Requests {
		if request.Method == http.MethodPut {
			writes++
			require.Len(t, requestJSON(t, request)["reviewers"], 2)
		}
	}
	require.Equal(t, 2, writes)
}

func TestEnsureReviewersBoundsConflictsAndDoesNotRewriteExistingRequests(t *testing.T) {
	t.Run("bounded conflicts", func(t *testing.T) {
		responses := []*http.Response{okResponse(reviewSnapshot("abc", "OPEN", reviewOther))}
		for range 3 {
			responses = append(responses, reviewPermissionPage(), reviewUserResponse(), okResponse(reviewSnapshot("abc", "OPEN", reviewOther)), bitbucketResponse(http.StatusPreconditionFailed, `{}`))
		}
		client, _ := stubBitbucketClient(responses...)
		_, err := client.ensurePullRequestReviewers("acme/widgets", 42, "abc", []ReviewerIdentity{{UUID: reviewHuman}})
		require.ErrorContains(t, err, "3 attempts")
	})
	t.Run("already requested", func(t *testing.T) {
		client, httpCtx := stubBitbucketClient(okResponse(reviewSnapshot("abc", "OPEN", reviewHuman, reviewOther)), reviewPermissionPage(), reviewUserResponse(), okResponse(reviewSnapshot("abc", "OPEN", reviewHuman, reviewOther)))
		outcome, err := client.ensurePullRequestReviewers("acme/widgets", 42, "abc", []ReviewerIdentity{{UUID: "{" + reviewHuman + "}"}})
		require.NoError(t, err)
		require.Equal(t, "already_covered", outcome.Status)
		require.Empty(t, outcome.Requested)
		require.Equal(t, reviewHuman, outcome.AlreadyCovered[0].UUID)
		for _, request := range httpCtx.Requests {
			require.NotEqual(t, http.MethodPut, request.Method)
		}
	})
}

type reviewerPublications struct {
	core.PublicationContext
	records map[string]core.OrderedPublication
}

func (p *reviewerPublications) Update(key string, apply func(*core.OrderedPublication) error) error {
	record := p.records[key]
	if err := apply(&record); err != nil {
		return err
	}
	p.records[key] = record
	return nil
}

type reviewerHTTPFunc func(*http.Request) (*http.Response, error)

func (f reviewerHTTPFunc) Do(request *http.Request) (*http.Response, error) { return f(request) }

func reviewerAPIIntegration() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{IntegrationID: reviewOther, Configuration: map[string]any{"token": "api-token", "email": "admin@example.com"}, Metadata: Metadata{AuthType: AuthTypeAPIToken, Workspace: &WorkspaceMetadata{Slug: "acme"}}}
}

func TestReviewerActionSerializesWritesAndEmitsScopedOutcome(t *testing.T) {
	integration := reviewerAPIIntegration()
	publications := &reviewerPublications{records: map[string]core.OrderedPublication{}}
	state := &contexts.ExecutionStateContext{}
	component := &EnsurePullRequestReviewers{}
	config := map[string]any{"repository": "acme/widgets", "pullNumber": 42, "expectedHeadSha": "abc", "reviewers": []any{map[string]any{"uuid": "{" + reviewHuman + "}"}}}
	sequence := []*http.Response{okResponse(`{"uuid":"{` + reviewOther + `}"}`), okResponse(reviewSnapshot("abc", "OPEN", reviewOther)), reviewPermissionPage(), reviewUserResponse(), okResponse(reviewSnapshot("abc", "OPEN", reviewOther)), okResponse(`{}`), okResponse(reviewSnapshot("abc", "OPEN", reviewOther, reviewHuman))}
	calls := 0
	httpCtx := reviewerHTTPFunc(func(request *http.Request) (*http.Response, error) {
		email, token, ok := request.BasicAuth()
		require.True(t, ok)
		require.Equal(t, "admin@example.com", email)
		require.Equal(t, "api-token", token)
		if calls == 1 {
			// A second execution reaches the same durable write boundary while the first reads the PR.
			otherState := &contexts.ExecutionStateContext{}
			err := component.Execute(core.ExecutionContext{Configuration: config, Integration: integration, Publications: publications, ExecutionState: otherState, HTTP: &contexts.HTTPContext{Responses: []*http.Response{okResponse(`{"uuid":"{` + reviewOther + `}"}`)}}})
			require.ErrorIs(t, err, core.ErrPublicationBusy)
			require.Empty(t, otherState.Payloads)
		}
		if request.Method == http.MethodPut {
			_, ok := request.Context().Deadline()
			require.True(t, ok)
		}
		response := sequence[calls]
		calls++
		return response, nil
	})
	err := component.Execute(core.ExecutionContext{ID: uuid.New(), Configuration: config, Integration: integration, Publications: publications, ExecutionState: state, HTTP: httpCtx})
	require.NoError(t, err)
	require.Equal(t, reviewersEnsuredPayload, state.Type)
	require.Len(t, state.Payloads, 1)
	encoded, err := json.Marshal(state.Payloads[0].(map[string]any)["data"])
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(encoded, &payload))
	require.Equal(t, "bitbucket", payload["provider"])
	require.Equal(t, reviewOther, payload["integrationId"])
	require.Equal(t, reviewOther, payload["repositoryId"])
	require.Equal(t, float64(42), payload["pullNumber"])
	require.Equal(t, "requested", payload["status"])
	// A retry after release observes the existing assignment and does not write again.
	retryHTTP := &contexts.HTTPContext{Responses: []*http.Response{okResponse(`{"uuid":"{` + reviewOther + `}"}`), okResponse(reviewSnapshot("abc", "OPEN", reviewOther, reviewHuman)), reviewPermissionPage(), reviewUserResponse(), okResponse(reviewSnapshot("abc", "OPEN", reviewOther, reviewHuman))}}
	require.NoError(t, component.Execute(core.ExecutionContext{Configuration: config, Integration: integration, Publications: publications, ExecutionState: &contexts.ExecutionStateContext{}, HTTP: retryHTTP}))
	for _, request := range retryHTTP.Requests {
		require.NotEqual(t, http.MethodPut, request.Method)
	}
}

func TestReviewCandidatesActionEmitsEmptyDiscovery(t *testing.T) {
	state := &contexts.ExecutionStateContext{}
	err := (&ListPullRequestReviewCandidates{}).Execute(core.ExecutionContext{Configuration: map[string]any{"repository": "acme/widgets", "pullNumber": "42"}, Integration: reviewerAPIIntegration(), ExecutionState: state, HTTP: &contexts.HTTPContext{Responses: []*http.Response{
		okResponse(`{"uuid":"{` + reviewOther + `}"}`), okResponse(reviewSnapshot("abc", "OPEN")), okResponse(`{"values":[]}`), okResponse(`{"values":[]}`), okResponse(`{"values":[]}`), okResponse(reviewSnapshot("abc", "OPEN")),
	}}})
	require.NoError(t, err)
	require.Equal(t, reviewCandidatesPayload, state.Type)
	encoded, err := json.Marshal(state.Payloads[0].(map[string]any)["data"])
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(encoded, &payload))
	require.Equal(t, []any{}, payload["candidates"])
}

func TestReviewerSetupAndResourceCredentialCapabilities(t *testing.T) {
	t.Run("API token checks effective permissions during setup", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{okResponse(`{"values":[{"uuid":"{` + reviewOther + `}","full_name":"acme/widgets"}]}`), okResponse(`{"values":[]}`)}}
		err := (&ListPullRequestReviewCandidates{}).Setup(core.SetupContext{Configuration: map[string]any{"repository": "acme/widgets", "pullNumber": "42"}, Integration: reviewerAPIIntegration(), HTTP: httpCtx, Metadata: &contexts.MetadataContext{}})
		require.NoError(t, err)
		require.Equal(t, "/2.0/workspaces/acme/permissions/repositories/widgets", httpCtx.Requests[len(httpCtx.Requests)-1].URL.Path)
	})
	for _, authType := range []string{AuthTypeForgeApp, AuthTypeRepositoryAccessToken, AuthTypeWorkspaceAccessToken} {
		t.Run(authType, func(t *testing.T) {
			integration := reviewerAPIIntegration()
			integration.Metadata = Metadata{AuthType: authType, Workspace: &WorkspaceMetadata{Slug: "acme"}, Repository: &RepositoryMetadata{FullName: "acme/widgets"}}
			httpCtx := &contexts.HTTPContext{}
			err := (&ListPullRequestReviewCandidates{}).Setup(core.SetupContext{Configuration: map[string]any{"repository": "acme/widgets", "pullNumber": "42"}, Integration: integration, HTTP: httpCtx})
			require.ErrorContains(t, err, "API token")
			_, err = (&Bitbucket{}).ListResources(resourceTypeReviewer, core.ListResourcesContext{Integration: integration, HTTP: httpCtx, Parameters: map[string]string{"repository": "acme/widgets"}})
			require.ErrorContains(t, err, "API token")
			require.Empty(t, httpCtx.Requests)
		})
	}
	t.Run("reviewer picker stores UUID and a separate label", func(t *testing.T) {
		resources, err := (&Bitbucket{}).ListResources(resourceTypeReviewer, core.ListResourcesContext{Integration: reviewerAPIIntegration(), HTTP: &contexts.HTTPContext{Responses: []*http.Response{reviewPermissionPage(), reviewUserResponse()}}, Parameters: map[string]string{"repository": "acme/widgets"}})
		require.NoError(t, err)
		require.Equal(t, []core.IntegrationResource{{Type: resourceTypeReviewer, Name: "Human", ID: reviewHuman}}, resources)
	})
}

func TestEnsureReviewerSetupRejectsInvalidSelection(t *testing.T) {
	for _, reviewers := range []any{[]any{}, []any{map[string]any{"uuid": "nickname"}}, []any{map[string]any{"uuid": reviewHuman}, map[string]any{"uuid": "{" + reviewHuman + "}"}}} {
		err := (&EnsurePullRequestReviewers{}).Setup(core.SetupContext{Configuration: map[string]any{"repository": "acme/widgets", "pullNumber": "42", "expectedHeadSha": "abc", "reviewers": reviewers}, Integration: reviewerAPIIntegration()})
		require.Error(t, err)
	}
}

func TestEnsureReviewersKeepsPostWriteStalenessVisibleDuringReconciliation(t *testing.T) {
	client, _ := stubBitbucketClient(
		okResponse(reviewSnapshot("abc", "OPEN", reviewOther)), reviewPermissionPage(), reviewUserResponse(), okResponse(reviewSnapshot("abc", "OPEN", reviewOther)), okResponse(`{}`), okResponse(reviewSnapshot("abc", "OPEN", reviewHuman)),
		reviewPermissionPage(), reviewUserResponse(), okResponse(reviewSnapshot("new", "OPEN", reviewHuman)),
	)
	outcome, err := client.ensurePullRequestReviewers("acme/widgets", 42, "abc", []ReviewerIdentity{{UUID: reviewHuman}})
	require.ErrorContains(t, err, "changed during reviewer update")
	require.Empty(t, outcome.Status)
}
