package launchdarkly

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetFeatureFlag__Setup(t *testing.T) {
	component := &GetFeatureFlag{}

	t.Run("valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectKey": "default",
				"flagKey":    "my-feature",
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing project key returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"flagKey": "my-feature",
			},
		})

		require.ErrorContains(t, err, "project key is required")
	})

	t.Run("empty project key returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectKey": "",
				"flagKey":    "my-feature",
			},
		})

		require.ErrorContains(t, err, "project key is required")
	})

	t.Run("missing flag key returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectKey": "default",
			},
		})

		require.ErrorContains(t, err, "flag key is required")
	})

	t.Run("invalid configuration format -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})
}

func Test__GetFeatureFlag__Execute(t *testing.T) {
	component := &GetFeatureFlag{}

	flagResponse := `{"key":"my-feature","name":"My Feature","description":"A test flag","kind":"boolean","creationDate":1700000000000,"archived":false,"temporary":false}`

	t.Run("success gets flag and emits output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(flagResponse)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-api-key"},
		}

		execStateCtx := &contexts.ExecutionStateContext{}
		execID := uuid.New()

		err := component.Execute(core.ExecutionContext{
			ID:             execID,
			Configuration:  map[string]any{"projectKey": "default", "flagKey": "my-feature"},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execStateCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "https://app.launchdarkly.com/api/v2/flags/default/my-feature", req.URL.String())
		assert.True(t, execStateCtx.Passed)
		require.Len(t, execStateCtx.Payloads, 1)
		payload := execStateCtx.Payloads[0].(map[string]any)
		assert.Equal(t, "launchdarkly.flag", payload["type"])
		assert.NotNil(t, payload["data"])
		data, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "my-feature", data["key"])
		assert.Equal(t, "My Feature", data["name"])
	})

	t.Run("missing project key returns error before API call", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-api-key"},
		}

		err := component.Execute(core.ExecutionContext{
			ID:             uuid.New(),
			Configuration:  map[string]any{"flagKey": "my-feature"},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "project key is required")
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("missing flag key returns error before API call", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-api-key"},
		}

		err := component.Execute(core.ExecutionContext{
			ID:             uuid.New(),
			Configuration:  map[string]any{"projectKey": "default"},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "flag key is required")
		assert.Empty(t, httpContext.Requests)
	})
}
