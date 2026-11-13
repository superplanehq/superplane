package bitbucket

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const reviewUser = "{11111111-1111-4111-8111-111111111111}"
const reviewRepository = "{22222222-2222-4222-8222-222222222222}"

func TestReviewStateRequiresCommitEvidence(t *testing.T) {
	date := time.Now().UTC().Truncate(time.Microsecond)
	sha := strings.Repeat("a", 40)
	for _, test := range []struct {
		name, participant, evidenceSHA, want string
		evidenceDate                         time.Time
	}{
		{"unattributed approval", `"approved":true,"state":"approved"`, "", "approval_commit_unverified", date},
		{"current approval", `"approved":true,"state":"approved"`, sha, "approved", date},
		{"approval from old head", `"approved":true,"state":"approved"`, strings.Repeat("b", 40), "approval_commit_unverified", date},
		{"approval removed", `"approved":false`, sha, "approval_missing", date},
		{"changes requested", `"approved":false,"state":"changes_requested"`, sha, "changes_requested", date},
		{"reapproval without event", `"approved":true,"state":"approved"`, sha, "approval_commit_unverified", date.Add(-time.Second)},
	} {
		t.Run(test.name, func(t *testing.T) {
			httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
				bitbucketResponse(200, fmt.Sprintf(`{"id":42,"state":"OPEN","source":{"commit":{"hash":%q},"branch":{"name":"topic"}},"participants":[{"user":{"uuid":%q},%s}]}`, sha, reviewUser, test.participant)),
				bitbucketResponse(200, `{"values":[{"comment":{"created_on":"2099-01-01T00:00:00Z"}}],"next":"https://api.bitbucket.org/2.0/repositories/acme/widgets/pullrequests/42/activity?page=2"}`),
				bitbucketResponse(200, fmt.Sprintf(`{"values":[{"approval":{"date":%q,"user":{"uuid":%q}}}]}`, date.Format(time.RFC3339Nano), reviewUser)),
			}}
			client := &Client{HTTP: httpCtx}
			evidence := map[string]ApprovalEvidence{}
			if test.evidenceSHA != "" {
				evidence[bitbucketUUID(reviewUser)] = ApprovalEvidence{SHA: test.evidenceSHA, Date: test.evidenceDate}
			}
			state, err := client.ReadPullRequestReviewState(context.Background(), "acme/widgets", 42, sha, []ReviewerIdentity{{UUID: reviewUser}}, evidence)
			require.NoError(t, err)
			require.Equal(t, test.want, state.Reviewers[0].Reason)
			require.Len(t, httpCtx.Requests, 3, "read the complete activity, including approvals beyond comments")
		})
	}
}

func TestReviewActivityRejectsForeignPagination(t *testing.T) {
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		bitbucketResponse(200, `{"values":[],"next":"https://attacker.invalid/collect"}`),
	}}
	_, err := (&Client{HTTP: httpCtx}).ListPullRequestReviewActivity(context.Background(), "acme/widgets", 42)
	require.Error(t, err)
	require.Len(t, httpCtx.Requests, 1)
}

func TestReviewActivityKeepsRepositoryIdentityAcrossCanonicalLinks(t *testing.T) {
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		bitbucketResponse(200, `{"values":[],"next":"https://api.bitbucket.org/2.0/repositories/acme/widgets/pullrequests/42/activity?page=2"}`),
		bitbucketResponse(200, `{"values":[]}`),
	}}
	_, err := (&Client{HTTP: httpCtx}).ListPullRequestReviewActivity(context.Background(), "acme/"+reviewRepository, 42)
	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 2)
	for _, request := range httpCtx.Requests {
		require.Contains(t, request.URL.Path, reviewRepository)
	}
}
