package bitbucket

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func stubBitbucketClient(responses ...*http.Response) (*Client, *contexts.HTTPContext) {
	httpCtx := &contexts.HTTPContext{Responses: responses}
	return &Client{AuthType: AuthTypeWorkspaceAccessToken, Token: "token", HTTP: httpCtx}, httpCtx
}

func okResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}
}

func Test__PullRequestMergeStrategy(t *testing.T) {
	assert.Equal(t, "squash", PullRequestMergeStrategy("squash"))
	assert.Equal(t, "fast_forward", PullRequestMergeStrategy("rebase"))
	assert.Equal(t, "merge_commit", PullRequestMergeStrategy("merge"))
	assert.Equal(t, "merge_commit", PullRequestMergeStrategy("unknown"))
	assert.Equal(t, "merge_commit", PullRequestMergeStrategy(""))
}

func Test__GetPullRequest(t *testing.T) {
	client, httpCtx := stubBitbucketClient(okResponse(`{
		"id": 42, "state": "OPEN",
		"source": {"branch": {"name": "feat/x"}, "commit": {"hash": "abc123"}},
		"destination": {"branch": {"name": "main", "merge_strategies": ["squash", "merge_commit", "fast_forward"]}, "commit": {"hash": "def456"}},
		"merge_commit": {"hash": "merge789"}
	}`))

	pr, err := client.GetPullRequest("acme/widgets", 42)
	require.NoError(t, err)
	assert.Equal(t, int64(42), pr.ID)
	assert.Equal(t, "OPEN", pr.State)
	assert.Equal(t, "abc123", pr.SourceHash)
	assert.Equal(t, "feat/x", pr.SourceBr)
	assert.Equal(t, "main", pr.DestBranch)
	assert.Equal(t, []string{"squash", "merge_commit", "fast_forward"}, pr.DestMergeStrategies)

	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "/2.0/repositories/acme/widgets/pullrequests/42", httpCtx.Requests[0].URL.Path)
}

func Test__ListMergeabilityChecks(t *testing.T) {
	t.Run("follows pagination", func(t *testing.T) {
		client, _ := stubBitbucketClient(
			okResponse(`{"values": [{"type": "git_mergeability_check", "status": "PASSED", "required": true, "blocking": false}], "next": "https://api.bitbucket.org/2.0/next-page"}`),
			okResponse(`{"values": [{"type": "current_user_permission_check", "status": "FAILED", "required": true, "blocking": true}]}`),
		)

		checks, err := client.ListMergeabilityChecks("acme/widgets", 42)
		require.NoError(t, err)
		require.Len(t, checks, 2)
		assert.Equal(t, "git_mergeability_check", checks[0].Type)
		assert.False(t, checks[0].Blocking)
		assert.Equal(t, "current_user_permission_check", checks[1].Type)
		assert.True(t, checks[1].Blocking)
	})

	t.Run("api failures surface the status", func(t *testing.T) {
		client, _ := stubBitbucketClient(&http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader(`{"error": {"message": "boom"}}`)),
		})

		_, err := client.ListMergeabilityChecks("acme/widgets", 42)
		require.Error(t, err)
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
	})
}

func Test__MergePullRequest(t *testing.T) {
	t.Run("sends the merge strategy", func(t *testing.T) {
		client, httpCtx := stubBitbucketClient(okResponse(`{"id": 42, "state": "MERGED"}`))

		merged, err := client.MergePullRequest("acme/widgets", 42, "squash")
		require.NoError(t, err)
		assert.Equal(t, "MERGED", merged.State)

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "/2.0/repositories/acme/widgets/pullrequests/42/merge", request.URL.Path)
		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"type": "pullrequest", "merge_strategy": "squash"}`, string(body))
	})

	t.Run("asynchronous merges do not count as merged", func(t *testing.T) {
		client, _ := stubBitbucketClient(&http.Response{
			StatusCode: http.StatusAccepted,
			Body:       io.NopCloser(strings.NewReader(`{"type": "error", "error": {"message": "pending"}}`)),
		})

		_, err := client.MergePullRequest("acme/widgets", 42, "merge_commit")
		require.ErrorIs(t, err, errBitbucketMergeAsync)
	})
}

func Test__DeclinePullRequest(t *testing.T) {
	client, httpCtx := stubBitbucketClient(okResponse(`{"id": 42, "state": "DECLINED"}`))

	declined, err := client.DeclinePullRequest("acme/widgets", 42)
	require.NoError(t, err)
	assert.Equal(t, "DECLINED", declined.State)

	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "/2.0/repositories/acme/widgets/pullrequests/42/decline", httpCtx.Requests[0].URL.Path)
}

func Test__ListMergedPullRequests(t *testing.T) {
	now := time.Now()
	from := now.Add(-30 * 24 * time.Hour)

	t.Run("stops at the window start", func(t *testing.T) {
		client, httpCtx := stubBitbucketClient(okResponse(`{"values": [
			{"id": 9, "state": "MERGED", "title": "new", "updated_on": "` + now.Add(-time.Hour).Format(time.RFC3339) + `",
			 "author": {"uuid": "{11111111-1111-1111-1111-111111111111}", "nickname": "ada", "display_name": "Ada"},
			 "source": {"commit": {"hash": "src9"}}, "merge_commit": {"hash": "m9"}},
			{"id": 8, "state": "MERGED", "title": "old", "updated_on": "` + now.Add(-60*24*time.Hour).Format(time.RFC3339) + `",
			 "author": {"uuid": "{22222222-2222-2222-2222-222222222222}", "nickname": "grace"},
			 "source": {"commit": {"hash": "src8"}}, "merge_commit": {"hash": "m8"}}
		]}`))

		prs, truncated, err := client.ListMergedPullRequests("acme/widgets", from, time.Time{}, 50)
		require.NoError(t, err)
		require.Len(t, prs, 1)
		assert.Equal(t, int64(9), prs[0].ID)
		assert.Equal(t, "11111111-1111-1111-1111-111111111111", prs[0].AuthorUUID)
		assert.Equal(t, "ada", prs[0].AuthorNick)
		assert.Equal(t, "m9", prs[0].MergeHash)
		assert.False(t, truncated)
		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, httpCtx.Requests[0].URL.RawQuery, "state=MERGED")
	})

	t.Run("page cap reports a short walk", func(t *testing.T) {
		client, _ := stubBitbucketClient(okResponse(`{"values": [
			{"id": 9, "state": "MERGED", "updated_on": "` + now.Format(time.RFC3339) + `", "author": {}, "source": {}, "merge_commit": {}}
		], "next": "https://api.bitbucket.org/2.0/repositories/acme/widgets/pullrequests?page=2"}`))

		prs, truncated, err := client.ListMergedPullRequests("acme/widgets", from, time.Time{}, 1)
		require.NoError(t, err)
		require.Len(t, prs, 1)
		assert.True(t, truncated)
	})

	t.Run("until asks for pulls older than the saved slice", func(t *testing.T) {
		until := now.Add(-2 * time.Hour)
		client, httpCtx := stubBitbucketClient(okResponse(`{"values": [
			{"id": 8, "state": "MERGED", "updated_on": "` + now.Add(-time.Hour).Format(time.RFC3339) + `", "author": {}, "source": {}, "merge_commit": {}},
			{"id": 9, "state": "MERGED", "updated_on": "` + now.Add(-3*time.Hour).Format(time.RFC3339) + `", "author": {}, "source": {}, "merge_commit": {}}
		]}`))

		prs, truncated, err := client.ListMergedPullRequests("acme/widgets", from, until, 50)
		require.NoError(t, err)
		require.Len(t, prs, 1)
		assert.Equal(t, int64(9), prs[0].ID)
		assert.False(t, truncated)
		assert.Contains(t, httpCtx.Requests[0].URL.Query().Get("q"), "updated_on < "+until.UTC().Format(time.RFC3339Nano))
	})

	t.Run("until keeps fractional seconds", func(t *testing.T) {
		until := time.Now().UTC().Truncate(time.Second).Add(500 * time.Millisecond)
		client, httpCtx := stubBitbucketClient(okResponse(`{"values": []}`))

		_, _, err := client.ListMergedPullRequests("acme/widgets", from, until, 1)
		require.NoError(t, err)
		assert.Equal(t, "updated_on < "+until.Format(time.RFC3339Nano), httpCtx.Requests[0].URL.Query().Get("q"))
	})

	t.Run("unparseable timestamps are skipped", func(t *testing.T) {
		client, _ := stubBitbucketClient(okResponse(`{"values": [
			{"id": 9, "state": "MERGED", "updated_on": "not-a-time", "author": {}, "source": {}, "merge_commit": {}}
		]}`))

		prs, _, err := client.ListMergedPullRequests("acme/widgets", from, time.Time{}, 50)
		require.NoError(t, err)
		assert.Empty(t, prs)
	})
}

func Test__ListMergedPullRequestsThroughBoundary(t *testing.T) {
	now := time.Now().UTC()
	from := now.Add(-30 * 24 * time.Hour)
	newer := now.Truncate(time.Second)
	boundary := newer.Add(-time.Second)
	older := boundary.Add(-time.Second)
	next := "https://api.bitbucket.org/2.0/repositories/acme/widgets/pullrequests?page=2"

	t.Run("finishes a timestamp the page cap splits", func(t *testing.T) {
		client, _ := stubBitbucketClient(
			okResponse(mergedPullPage([]mergedPullFixture{{2, newer}, {1, boundary}}, next)),
			okResponse(mergedPullPage([]mergedPullFixture{{3, boundary}, {4, older}}, next)),
		)

		prs, truncated, err := client.ListMergedPullRequestsThroughBoundary("acme/widgets", from, time.Time{}, 1, 1)
		require.NoError(t, err)
		require.Len(t, prs, 3)
		assert.Equal(t, int64(2), prs[0].ID)
		assert.Equal(t, int64(1), prs[1].ID)
		assert.Equal(t, int64(3), prs[2].ID)
		assert.True(t, truncated, "older pulls remain, but the shared instant is fully read")
	})

	t.Run("leaves an unfinished timestamp for the next slice", func(t *testing.T) {
		client, _ := stubBitbucketClient(
			okResponse(mergedPullPage([]mergedPullFixture{{2, newer}, {1, boundary}}, next)),
			okResponse(mergedPullPage([]mergedPullFixture{{3, boundary}}, next)),
		)

		prs, truncated, err := client.ListMergedPullRequestsThroughBoundary("acme/widgets", from, time.Time{}, 1, 1)
		require.NoError(t, err)
		require.Len(t, prs, 1)
		assert.Equal(t, int64(2), prs[0].ID)
		assert.True(t, truncated)
	})

	t.Run("does not skip a timestamp that fills the page cap", func(t *testing.T) {
		client, _ := stubBitbucketClient(
			okResponse(mergedPullPage([]mergedPullFixture{{1, boundary}}, next)),
			okResponse(mergedPullPage([]mergedPullFixture{{2, boundary}}, next)),
		)

		_, _, err := client.ListMergedPullRequestsThroughBoundary("acme/widgets", from, time.Time{}, 1, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "updated_on")
	})
}

type mergedPullFixture struct {
	id      int64
	updated time.Time
}

func mergedPullPage(pulls []mergedPullFixture, next string) string {
	values := ""
	for i, pull := range pulls {
		if i > 0 {
			values += ","
		}
		values += `{"id":` + strconv.FormatInt(pull.id, 10) + `,"state":"MERGED","updated_on":"` + pull.updated.Format(time.RFC3339Nano) + `","author":{},"source":{},"merge_commit":{}}`
	}
	page := `{"values":[` + values + `]`
	if next != "" {
		page += `,"next":"` + next + `"`
	}
	return page + `}`
}

func Test__RateLimitedResponsesSurfaceStatus(t *testing.T) {
	client, _ := stubBitbucketClient(&http.Response{
		StatusCode: http.StatusTooManyRequests,
		Body:       io.NopCloser(strings.NewReader(`{"error": {"message": "Rate limit exceeded"}}`)),
	})

	_, _, err := client.ListMergedPullRequests("acme/widgets", time.Now().Add(-time.Hour), time.Time{}, 50)
	require.Error(t, err)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
}

func Test__GetMergeCommit(t *testing.T) {
	client, httpCtx := stubBitbucketClient(okResponse(`{
		"hash": "m9", "date": "2026-04-22T10:00:00+00:00", "message": "Merged in feat/x\n\nCo-authored-by: SuperPlane Agent <superplaneagent@superplane.com>"
	}`))

	commit, err := client.GetMergeCommit("acme/widgets", "m9")
	require.NoError(t, err)
	assert.Equal(t, "m9", commit.Hash)
	assert.Equal(t, 2026, commit.Date.Year())
	assert.Contains(t, commit.Message, "Co-authored-by")

	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "/2.0/repositories/acme/widgets/commit/m9", httpCtx.Requests[0].URL.Path)
}

func Test__ListCommitStatuses(t *testing.T) {
	t.Run("follows pagination", func(t *testing.T) {
		client, _ := stubBitbucketClient(
			okResponse(`{"values": [{"key": "build-a", "state": "SUCCESSFUL", "url": "https://ci.example/a"}], "next": "https://api.bitbucket.org/2.0/next-page"}`),
			okResponse(`{"values": [{"key": "build-b", "state": "INPROGRESS", "url": "https://ci.example/b"}]}`),
		)

		statuses, err := client.ListCommitStatuses("acme/widgets", "abc123")
		require.NoError(t, err)
		require.Len(t, statuses, 2)
		assert.Equal(t, "build-a", statuses[0].Key)
		assert.Equal(t, CommitStatusSuccessful, statuses[0].State)
		assert.Equal(t, "build-b", statuses[1].Key)
		assert.Equal(t, CommitStatusInProgress, statuses[1].State)
	})

	t.Run("empty status list is not an error", func(t *testing.T) {
		client, _ := stubBitbucketClient(okResponse(`{"values": []}`))

		statuses, err := client.ListCommitStatuses("acme/widgets", "abc123")
		require.NoError(t, err)
		assert.Empty(t, statuses)
	})

	t.Run("missing sha is rejected", func(t *testing.T) {
		client, _ := stubBitbucketClient()

		_, err := client.ListCommitStatuses("acme/widgets", "  ")
		require.ErrorContains(t, err, "commit sha is required")
	})
}
