package dockerhub

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DockerHub__NewClient(t *testing.T) {
	t.Run("missing username -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"accessToken": "pat",
			},
		}

		_, err := NewClient(&contexts.HTTPContext{}, integrationCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "username")
	})

	t.Run("missing access token -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"username": "superplane",
			},
		}

		_, err := NewClient(&contexts.HTTPContext{}, integrationCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "accessToken")
	})

	t.Run("valid configuration -> client created", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"username":    "superplane",
				"accessToken": "pat",
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, integrationCtx)
		require.NoError(t, err)
		assert.Equal(t, "superplane", client.Identifier)
		assert.Equal(t, "pat", client.Secret)
	})
}

func Test__DockerHub__ListRepositories(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"token"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`
					{
						"next": null,
						"results": [
							{"name": "demo", "namespace": "superplane"}
						]
					}
				`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"username":    "superplane",
			"accessToken": "pat",
		},
	}

	client, err := NewClient(httpCtx, integrationCtx)
	require.NoError(t, err)

	repos, err := client.ListRepositories("superplane")
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "demo", repos[0].Name)

	require.Len(t, httpCtx.Requests, 2)
	assert.Contains(t, httpCtx.Requests[0].URL.String(), "/v2/auth/token")
	assert.Contains(t, httpCtx.Requests[1].URL.String(), "/v2/namespaces/superplane/repositories")
	assert.Equal(t, "Bearer token", httpCtx.Requests[1].Header.Get("Authorization"))
}
