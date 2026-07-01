package checks

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func Test__ListCheckRunsForRef__Execute(t *testing.T) {
	component := &ListCheckRunsForRef{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			mocks.GitHubResponse(http.StatusOK, `{
				"total_count": 2,
				"check_runs": [
					{"id": 1, "name": "DCO", "status": "completed", "conclusion": "success"},
					{"id": 2, "name": "Cloudflare Pages", "status": "completed", "conclusion": "success"}
				]
			}`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: ListCheckRunsForRefConfiguration{
			Repository: "hello",
			Ref:        "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44",
			CheckName:  "DCO",
			Status:     "completed",
			Filter:     "latest",
		},
		HTTP:           httpCtx,
		Integration:    mocks.IntegrationContextForNewSetupFlow(),
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "/repos/testhq/hello/commits/d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44/check-runs", httpCtx.Requests[0].URL.Path)
	assert.Equal(t, "DCO", httpCtx.Requests[0].URL.Query().Get("check_name"))
	assert.Equal(t, "completed", httpCtx.Requests[0].URL.Query().Get("status"))
	assert.Equal(t, "latest", httpCtx.Requests[0].URL.Query().Get("filter"))
	assert.Equal(t, "github.checkRuns", executionState.Type)
	assert.True(t, executionState.Passed)
}

func Test__ListCheckRunsForRef__Setup(t *testing.T) {
	component := &ListCheckRunsForRef{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			mocks.GitHubResponse(http.StatusOK, `{
				"id": 123456,
				"name": "hello",
				"html_url": "https://github.com/testhq/hello"
			}`),
		},
	}
	metadata := &contexts.MetadataContext{}

	err := component.Setup(core.SetupContext{
		Integration:   mocks.IntegrationContextForNewSetupFlow(),
		HTTP:          httpCtx,
		Metadata:      metadata,
		Configuration: map[string]any{"repository": "hello"},
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)
}
