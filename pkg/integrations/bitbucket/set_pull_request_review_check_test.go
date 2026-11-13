package bitbucket

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestReviewCheckSetupSubscribesWithoutResolvingDecisionExpressions(t *testing.T) {
	integration := workspaceTokenIntegration()
	metadata := &contexts.MetadataContext{}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{bitbucketResponse(200, `{"values":[{"uuid":"{repo}","full_name":"acme/widgets","slug":"widgets"}]}`)}}
	err := (&SetPullRequestReviewCheck{}).Setup(core.SetupContext{Configuration: map[string]any{"repository": "acme/widgets", "pullRequestId": "{{ $.trigger.data.id }}", "reviewers": "{{ $.selection.data.reviewers }}"}, Integration: integration, Metadata: metadata, HTTP: httpCtx})
	require.NoError(t, err)
	require.Len(t, integration.WebhookRequests, 1)
	request := integration.WebhookRequests[0].(WebhookConfiguration)
	require.Contains(t, request.EventTypes, "pullrequest:approved")
	require.Contains(t, request.EventTypes, "pullrequest:changes_request_removed")
	require.Contains(t, request.EventTypes, "repo:push")
	// Invalid signatures cannot create commit evidence or wake publications.
	code, _, err := (&SetPullRequestReviewCheck{}).HandleWebhook(core.WebhookRequestContext{Headers: http.Header{}, Body: []byte(`{}`)})
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, code)
}
