package jenkins

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

func Test__Jenkins__Sync(t *testing.T) {
	j := &Jenkins{}

	t.Run("no url -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "",
				"username": "admin",
				"apiToken": "test-token",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "url is required")
	})

	t.Run("no username -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "",
				"apiToken": "test-token",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "username is required")
	})

	t.Run("no apiToken -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "admin",
				"apiToken": "",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "apiToken is required")
	})

	t.Run("successful sync -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"mode":"NORMAL","url":"https://jenkins.example.com/"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "admin",
				"apiToken": "test-token",
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
					Body:       io.NopCloser(strings.NewReader(`<html>Authentication required</html>`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "https://jenkins.example.com",
				"username": "admin",
				"apiToken": "invalid-token",
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

func Test__Jenkins__IntegrationInfo(t *testing.T) {
	j := &Jenkins{}

	assert.Equal(t, "jenkins", j.Name())
	assert.Equal(t, "Jenkins", j.Label())
	assert.Equal(t, "jenkins", j.Icon())
	assert.NotEmpty(t, j.Description())
}

func Test__Jenkins__Components(t *testing.T) {
	j := &Jenkins{}
	components := j.Components()

	require.Len(t, components, 1)
	assert.Equal(t, "jenkins.triggerBuild", components[0].Name())
}

func Test__Jenkins__Triggers(t *testing.T) {
	j := &Jenkins{}
	triggers := j.Triggers()

	require.Len(t, triggers, 1)
	assert.Equal(t, "jenkins.onBuildFinished", triggers[0].Name())
}
