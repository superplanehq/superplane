package workers

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bitbucketintegration "github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	supportcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func stubBitbucketVelocityClient(responses ...*http.Response) (*bitbucketintegration.Client, *supportcontexts.HTTPContext) {
	httpCtx := &supportcontexts.HTTPContext{Responses: responses}
	client, err := bitbucketintegration.NewClient(
		bitbucketintegration.AuthTypeWorkspaceAccessToken,
		httpCtx,
		&supportcontexts.IntegrationContext{Configuration: map[string]any{"token": "token"}},
	)
	if err != nil {
		panic(err)
	}
	return client, httpCtx
}

func bitbucketVelocityPR() *bitbucketintegration.MergedBitbucketPullRequest {
	return &bitbucketintegration.MergedBitbucketPullRequest{
		ID:         42,
		State:      "MERGED",
		AuthorUUID: "11111111-1111-1111-1111-111111111111",
		AuthorNick: "ada",
		AuthorName: "Ada Lovelace",
		SourceHash: "src42",
		MergeHash:  "m42",
		UpdatedOn:  time.Now(),
	}
}

func TestToBitbucketRepositoryMerge(t *testing.T) {
	now := time.Now()
	from := now.Add(-30 * 24 * time.Hour)
	to := now.Add(time.Hour)

	mergeDate := now.Add(-time.Hour).Format(time.RFC3339)

	t.Run("dates the merge from the pull request, not the commit", func(t *testing.T) {
		client, _ := stubBitbucketVelocityClient(okJSONResponse(`{
			"hash": "m42", "date": "2020-01-01T10:00:00+00:00", "message": "Merged in feat/x"
		}`))

		merge, ok, err := toBitbucketRepositoryMerge(client, "acme/widgets", bitbucketVelocityPR(), from, to)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, int64(42), merge.Number)
		assert.Equal(t, models.FactoryVelocityMergeSourcePeople, merge.Source)
		assert.Equal(t, "11111111-1111-1111-1111-111111111111", merge.AuthorUUID)
		assert.Equal(t, "ada", merge.AuthorLogin)
		assert.WithinDuration(t, now, merge.MergedAt, time.Minute)
	})

	t.Run("classifies agent merges by the co-author trailer", func(t *testing.T) {
		client, _ := stubBitbucketVelocityClient(okJSONResponse(`{
			"hash": "m42", "date": "` + mergeDate + `",
			"message": "Squashed commit\n\nCo-authored-by: SuperPlane Agent <superplaneagent@superplane.com>"
		}`))

		merge, ok, err := toBitbucketRepositoryMerge(client, "acme/widgets", bitbucketVelocityPR(), from, to)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, models.FactoryVelocityMergeSourceAgent, merge.Source)
	})

	t.Run("falls back to updated_on when the commit is unreadable", func(t *testing.T) {
		client, _ := stubBitbucketVelocityClient(&http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		})
		pr := bitbucketVelocityPR()
		pr.UpdatedOn = now.Add(-time.Hour)

		merge, ok, err := toBitbucketRepositoryMerge(client, "acme/widgets", pr, from, to)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, models.FactoryVelocityMergeSourcePeople, merge.Source)
	})

	t.Run("returns temporary commit read errors", func(t *testing.T) {
		client, _ := stubBitbucketVelocityClient(&http.Response{
			StatusCode: http.StatusTooManyRequests,
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"Rate limit exceeded"}}`)),
		})

		_, ok, err := toBitbucketRepositoryMerge(client, "acme/widgets", bitbucketVelocityPR(), from, to)
		require.Error(t, err)
		assert.False(t, ok)
		assert.Contains(t, err.Error(), "Rate limit exceeded")
	})

	t.Run("keeps a fast-forward merge when the commit predates the window", func(t *testing.T) {
		client, _ := stubBitbucketVelocityClient(okJSONResponse(`{
			"hash": "m42", "date": "2020-01-01T10:00:00+00:00",
			"message": "Fast-forward\n\nCo-authored-by: SuperPlane Agent <superplaneagent@superplane.com>"
		}`))
		pr := bitbucketVelocityPR()
		pr.SourceHash = "m42"
		pr.UpdatedOn = now.Add(-time.Hour)

		merge, ok, err := toBitbucketRepositoryMerge(client, "acme/widgets", pr, from, to)
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, models.FactoryVelocityMergeSourceAgent, merge.Source)
		assert.WithinDuration(t, now.Add(-time.Hour), merge.MergedAt, time.Minute)
	})

	t.Run("excludes merges outside the window", func(t *testing.T) {
		client, _ := stubBitbucketVelocityClient(okJSONResponse(`{
			"hash": "m42", "date": "` + mergeDate + `", "message": "Merged in feat/x"
		}`))
		pr := bitbucketVelocityPR()
		pr.UpdatedOn = from.Add(-time.Hour)

		_, ok, err := toBitbucketRepositoryMerge(client, "acme/widgets", pr, from, to)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestListBitbucketMergesKeepsAShortPageWalkIncomplete(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	body := `{"values":[{"id":1,"state":"MERGED","updated_on":"` + now + `","author":{},"source":{},"merge_commit":{"hash":"m1"}}],"next":"https://api.bitbucket.org/2.0/next"}`
	responses := make([]*http.Response, velocitySyncMaxPages)
	for i := range responses {
		responses[i] = okJSONResponse(body)
	}
	client, _ := stubBitbucketVelocityClient(responses...)

	_, err := listBitbucketMerges(context.Background(), client, "acme/widgets", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeded")
}

func okJSONResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}
}
