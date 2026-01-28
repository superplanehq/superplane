package daytona

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

func Test__Daytona__Sync(t *testing.T) {
	d := &Daytona{}

	t.Run("no apiKey -> error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiKey": "",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			AppInstallation: appCtx,
		})

		require.ErrorContains(t, err, "apiKey is required")
	})

	t.Run("successful connection test -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/sandbox")
		assert.Equal(t, "Bearer test-api-key", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("connection test failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"message":"unauthorized"}`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiKey": "invalid-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", appCtx.State)
	})

	t.Run("custom baseURL is used", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiKey":  "test-api-key",
				"baseURL": "https://custom.daytona.io/api",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "https://custom.daytona.io/api/sandbox")
	})
}

func Test__Daytona__Metadata(t *testing.T) {
	t.Run("metadata is set on successful sync", func(t *testing.T) {
		d := &Daytona{}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.NotNil(t, appCtx.Metadata)
		_, ok := appCtx.Metadata.(Metadata)
		assert.True(t, ok, "metadata should be of type Metadata")
	})
}

func Test__Daytona__Components(t *testing.T) {
	d := &Daytona{}

	components := d.Components()
	assert.Len(t, components, 4)

	componentNames := make([]string, len(components))
	for i, c := range components {
		componentNames[i] = c.Name()
	}

	assert.Contains(t, componentNames, "daytona.createSandbox")
	assert.Contains(t, componentNames, "daytona.executeCode")
	assert.Contains(t, componentNames, "daytona.executeCommand")
	assert.Contains(t, componentNames, "daytona.deleteSandbox")
}

func Test__Daytona__Triggers(t *testing.T) {
	d := &Daytona{}

	triggers := d.Triggers()
	assert.Empty(t, triggers)
}

func Test__Daytona__ApplicationInfo(t *testing.T) {
	d := &Daytona{}

	assert.Equal(t, "daytona", d.Name())
	assert.Equal(t, "Daytona", d.Label())
	assert.Equal(t, "daytona", d.Icon())
	assert.Equal(t, "Execute code in isolated sandbox environments", d.Description())
	assert.Empty(t, d.InstallationInstructions())
}

func Test__Daytona__Configuration(t *testing.T) {
	d := &Daytona{}

	config := d.Configuration()
	assert.Len(t, config, 3)

	fieldNames := make([]string, len(config))
	for i, f := range config {
		fieldNames[i] = f.Name
	}

	assert.Contains(t, fieldNames, "apiKey")
	assert.Contains(t, fieldNames, "baseURL")
	assert.Contains(t, fieldNames, "target")

	for _, f := range config {
		if f.Name == "apiKey" {
			assert.True(t, f.Required)
			assert.True(t, f.Sensitive)
		}
	}
}
