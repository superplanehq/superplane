package pulls

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func TestSetPullRequestReviewCheckSetupAcceptsExpressions(t *testing.T) {
	integration := mocks.IntegrationContextForNewSetupFlow()
	err := (&SetPullRequestReviewCheck{}).Setup(core.SetupContext{
		Integration: integration,
		Metadata:    &contexts.MetadataContext{},
		HTTP: &contexts.HTTPContext{Responses: []*http.Response{
			mocks.GitHubResponse(http.StatusOK, `{"id":123,"name":"hello","full_name":"testhq/hello"}`),
		}},
		Configuration: map[string]any{
			"repository":           "testhq/hello",
			"pullNumber":           "{{ $.trigger.data.number }}",
			"reviewers":            "{{ $.selection.data.reviewers }}",
			"activationGeneration": "{{ $.trigger.data.generation }}",
		},
	})
	require.NoError(t, err)
	require.Equal(t, []any{common.WebhookConfiguration{
		Repository: "testhq/hello",
		EventTypes: []string{"pull_request", "pull_request_review"},
	}}, integration.WebhookRequests)
}
