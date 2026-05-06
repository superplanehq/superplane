package common

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func Test__EnsureRepoInMetadata(t *testing.T) {
	t.Run("repository is required", func(t *testing.T) {
		integrationCtx := mocks.IntegrationContextForNewSetupFlow()

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"id": 123456,
					"name": "hello",
					"html_url": "https://github.com/testhq/hello"
				}`),
			},
		}

		err := EnsureRepoInMetadata(
			&contexts.MetadataContext{},
			integrationCtx,
			httpCtx,
			map[string]any{"repository": ""},
		)

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("repository is not accessible", func(t *testing.T) {
		integrationCtx := mocks.IntegrationContextForNewSetupFlow()
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusNotFound, `{"message":"Not Found"}`),
			},
		}

		err := EnsureRepoInMetadata(
			&contexts.MetadataContext{},
			integrationCtx,
			httpCtx,
			map[string]any{"repository": "world"},
		)

		require.ErrorContains(t, err, "failed to find repository")
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, "/repos/testhq/world", httpCtx.Requests[0].URL.Path)
		assert.Equal(t, "Bearer test-token", httpCtx.Requests[0].Header.Get("Authorization"))
	})

	t.Run("repository exists and is saved in metadata", func(t *testing.T) {
		integrationCtx := mocks.IntegrationContextForNewSetupFlow()
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"id": 123456,
					"name": "hello",
					"html_url": "https://github.com/testhq/hello"
				}`),
			},
		}

		nodeMetadataCtx := &contexts.MetadataContext{}
		err := EnsureRepoInMetadata(
			nodeMetadataCtx,
			integrationCtx,
			httpCtx,
			map[string]any{"repository": "hello"},
		)

		require.NoError(t, err)

		helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
		require.Equal(t, nodeMetadataCtx.Get(), NodeMetadata{Repository: &helloRepo})
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, "/repos/testhq/hello", httpCtx.Requests[0].URL.Path)
		assert.Equal(t, "Bearer test-token", httpCtx.Requests[0].Header.Get("Authorization"))
	})
}
