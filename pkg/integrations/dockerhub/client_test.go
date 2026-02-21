package dockerhub

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DockerHub__NewClient(t *testing.T) {
	t.Run("missing access token secret -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}

		_, err := NewClient(&contexts.HTTPContext{}, integrationCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access token not configured")
	})

	t.Run("valid configuration -> client created", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				accessTokenSecretName: {Name: accessTokenSecretName, Value: []byte("token")},
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, integrationCtx)
		require.NoError(t, err)
		assert.Equal(t, "token", client.AccessToken)
	})
}

func Test__DockerHub__ListRepositories(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
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
		Secrets: map[string]core.IntegrationSecret{
			accessTokenSecretName: {Name: accessTokenSecretName, Value: []byte("token")},
		},
	}

	client, err := NewClient(httpCtx, integrationCtx)
	require.NoError(t, err)

	repos, err := client.ListRepositories("superplane")
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "demo", repos[0].Name)

	require.Len(t, httpCtx.Requests, 1)
	assert.Contains(t, httpCtx.Requests[0].URL.String(), "/v2/namespaces/superplane/repositories")
	assert.Equal(t, "Bearer token", httpCtx.Requests[0].Header.Get("Authorization"))
}
