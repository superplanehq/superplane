package splitio

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
				"workspaceId":   "ws-123",
				"environmentId": "env-456",
				"flagName":      "my-feature",
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing workspace returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"environmentId": "env-456",
				"flagName":      "my-feature",
			},
		})

		require.ErrorContains(t, err, "workspace is required")
	})

	t.Run("empty workspace returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"workspaceId":   "",
				"environmentId": "env-456",
				"flagName":      "my-feature",
			},
		})

		require.ErrorContains(t, err, "workspace is required")
	})

	t.Run("missing environment returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"workspaceId": "ws-123",
				"flagName":    "my-feature",
			},
		})

		require.ErrorContains(t, err, "environment is required")
	})

	t.Run("missing flag name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"workspaceId":   "ws-123",
				"environmentId": "env-456",
			},
		})

		require.ErrorContains(t, err, "feature flag name is required")
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

	flagResponse := `{"name":"my-feature","killed":false,"treatments":[{"name":"on"},{"name":"off"}],"defaultTreatment":"off","trafficAllocation":100,"creationTime":1704067200000}`

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
			Configuration:  map[string]any{"workspaceId": "ws-123", "environmentId": "env-456", "flagName": "my-feature"},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execStateCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "https://api.split.io/internal/api/v2/splits/ws/ws-123/my-feature/environments/env-456", req.URL.String())
		assert.True(t, execStateCtx.Passed)
		require.Len(t, execStateCtx.Payloads, 1)
		payload := execStateCtx.Payloads[0].(map[string]any)
		assert.Equal(t, "splitio.flag", payload["type"])
		assert.NotNil(t, payload["data"])
		data, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "my-feature", data["name"])
	})

	t.Run("missing workspace returns error before API call", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-api-key"},
		}

		err := component.Execute(core.ExecutionContext{
			ID:             uuid.New(),
			Configuration:  map[string]any{"environmentId": "env-456", "flagName": "my-feature"},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "workspace is required")
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("missing environment returns error before API call", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-api-key"},
		}

		err := component.Execute(core.ExecutionContext{
			ID:             uuid.New(),
			Configuration:  map[string]any{"workspaceId": "ws-123", "flagName": "my-feature"},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "environment is required")
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("missing flag name returns error before API call", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-api-key"},
		}

		err := component.Execute(core.ExecutionContext{
			ID:             uuid.New(),
			Configuration:  map[string]any{"workspaceId": "ws-123", "environmentId": "env-456"},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "feature flag name is required")
		assert.Empty(t, httpContext.Requests)
	})
}
