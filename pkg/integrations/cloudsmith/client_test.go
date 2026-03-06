package cloudsmith

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Cloudsmith__NewClient(t *testing.T) {
	t.Run("missing API key -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}

		_, err := NewClient(&contexts.HTTPContext{}, integrationCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key not configured")
	})

	t.Run("valid configuration -> client created", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":    "test-api-key",
				"workspace": "my-org",
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, integrationCtx)
		require.NoError(t, err)
		assert.Equal(t, "test-api-key", client.APIKey)
		assert.Equal(t, "my-org", client.Workspace)
	})
}

func Test__Cloudsmith__ListRepositories(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"name":"my-repo","slug":"my-repo"}]`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey":    "test-api-key",
			"workspace": "my-org",
		},
	}

	client, err := NewClient(httpCtx, integrationCtx)
	require.NoError(t, err)

	repos, err := client.ListRepositories("my-org")
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "my-repo", repos[0].Name)

	require.Len(t, httpCtx.Requests, 1)
	assert.Contains(t, httpCtx.Requests[0].URL.String(), "/repos/my-org/")
	assert.Equal(t, "test-api-key", httpCtx.Requests[0].Header.Get("X-Api-Key"))
}
