package jfrogartifactory

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

func Test__JFrogArtifactory__Sync(t *testing.T) {
	j := &JFrogArtifactory{}

	t.Run("no url -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":         "",
				"accessToken": "test-token",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "url is required")
	})

	t.Run("no accessToken -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":         "https://mycompany.jfrog.io",
				"accessToken": "",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "accessToken is required")
	})

	t.Run("successful sync -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("OK")),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":         "https://mycompany.jfrog.io",
				"accessToken": "test-token",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
	})

	t.Run("auth failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"errors":[{"status":401,"message":"Bad credentials"}]}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":         "https://mycompany.jfrog.io",
				"accessToken": "invalid-token",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", appCtx.State)
	})
}

func Test__JFrogArtifactory__IntegrationInfo(t *testing.T) {
	j := &JFrogArtifactory{}

	assert.Equal(t, "jfrogArtifactory", j.Name())
	assert.Equal(t, "JFrog Artifactory", j.Label())
	assert.Equal(t, "jfrogArtifactory", j.Icon())
	assert.NotEmpty(t, j.Description())
}

func Test__JFrogArtifactory__Triggers(t *testing.T) {
	j := &JFrogArtifactory{}
	triggers := j.Triggers()

	require.Len(t, triggers, 1)
	assert.Equal(t, "jfrogArtifactory.onArtifactUploaded", triggers[0].Name())
}

func Test__JFrogArtifactory__ListResources(t *testing.T) {
	j := &JFrogArtifactory{}

	t.Run("unsupported resource type -> empty", func(t *testing.T) {
		resources, err := j.ListResources("unsupported", core.ListResourcesContext{})
		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("repository -> returns repos", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"key":"libs-release-local","type":"LOCAL","packageType":"maven"},
						{"key":"libs-snapshot-local","type":"LOCAL","packageType":"maven"}
					]`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":         "https://mycompany.jfrog.io",
				"accessToken": "test-token",
			},
		}

		resources, err := j.ListResources("repository", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: appCtx,
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "libs-release-local", resources[0].Name)
		assert.Equal(t, "libs-release-local", resources[0].ID)
		assert.Equal(t, "libs-snapshot-local", resources[1].Name)
	})
}
