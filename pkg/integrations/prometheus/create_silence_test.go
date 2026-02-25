package prometheus

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateSilence__Setup(t *testing.T) {
	component := &CreateSilence{}

	t.Run("matchers are required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"matchers":  []any{},
				"duration":  "1h",
				"createdBy": "SuperPlane",
				"comment":   "Test",
			},
		})
		require.ErrorContains(t, err, "at least one matcher is required")
	})

	t.Run("matcher name is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"matchers": []any{
					map[string]any{"name": "", "value": "test"},
				},
				"duration":  "1h",
				"createdBy": "SuperPlane",
				"comment":   "Test",
			},
		})
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("matcher value is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"matchers": []any{
					map[string]any{"name": "alertname", "value": ""},
				},
				"duration":  "1h",
				"createdBy": "SuperPlane",
				"comment":   "Test",
			},
		})
		require.ErrorContains(t, err, "value is required")
	})

	t.Run("duration is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"matchers": []any{
					map[string]any{"name": "alertname", "value": "test"},
				},
				"duration":  "",
				"createdBy": "SuperPlane",
				"comment":   "Test",
			},
		})
		require.ErrorContains(t, err, "duration is required")
	})

	t.Run("invalid duration returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"matchers": []any{
					map[string]any{"name": "alertname", "value": "test"},
				},
				"duration":  "invalid",
				"createdBy": "SuperPlane",
				"comment":   "Test",
			},
		})
		require.ErrorContains(t, err, "invalid duration")
	})

	t.Run("createdBy is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"matchers": []any{
					map[string]any{"name": "alertname", "value": "test"},
				},
				"duration":  "1h",
				"createdBy": "",
				"comment":   "Test",
			},
		})
		require.ErrorContains(t, err, "createdBy is required")
	})

	t.Run("comment is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"matchers": []any{
					map[string]any{"name": "alertname", "value": "test"},
				},
				"duration":  "1h",
				"createdBy": "SuperPlane",
				"comment":   "",
			},
		})
		require.ErrorContains(t, err, "comment is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"matchers": []any{
					map[string]any{"name": "alertname", "value": "HighLatency"},
				},
				"duration":  "1h",
				"createdBy": "SuperPlane",
				"comment":   "Maintenance window",
			},
		})
		require.NoError(t, err)
	})
}

func Test__CreateSilence__Execute(t *testing.T) {
	component := &CreateSilence{}

	t.Run("silence is created and emitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"silenceID":"abc123"}`)),
				},
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"matchers": []any{
					map[string]any{"name": "alertname", "value": "HighLatency", "isRegex": false, "isEqual": true},
				},
				"duration":  "1h",
				"createdBy": "SuperPlane",
				"comment":   "Maintenance window",
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeNone,
			}},
			Metadata:       metadataCtx,
			ExecutionState: executionCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionCtx.Finished)
		assert.True(t, executionCtx.Passed)
		assert.Equal(t, "prometheus.silence", executionCtx.Type)
		require.Len(t, executionCtx.Payloads, 1)

		wrappedPayload := executionCtx.Payloads[0].(map[string]any)
		payload := wrappedPayload["data"].(map[string]any)
		assert.Equal(t, "abc123", payload["silenceID"])
		assert.Equal(t, "active", payload["status"])
		assert.Equal(t, "SuperPlane", payload["createdBy"])
		assert.Equal(t, "Maintenance window", payload["comment"])

		matchers := payload["matchers"].([]map[string]any)
		require.Len(t, matchers, 1)
		assert.Equal(t, "alertname", matchers[0]["name"])
		assert.Equal(t, "HighLatency", matchers[0]["value"])
		assert.Equal(t, false, matchers[0]["isRegex"])
		assert.Equal(t, true, matchers[0]["isEqual"])

		assert.NotEmpty(t, payload["startsAt"])
		assert.NotEmpty(t, payload["endsAt"])

		startsAt, err := time.Parse(time.RFC3339, payload["startsAt"].(string))
		require.NoError(t, err)
		endsAt, err := time.Parse(time.RFC3339, payload["endsAt"].(string))
		require.NoError(t, err)
		assert.True(t, endsAt.After(startsAt))

		metadata := metadataCtx.Metadata.(CreateSilenceNodeMetadata)
		assert.Equal(t, "abc123", metadata.SilenceID)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error":"invalid matchers"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"matchers": []any{
					map[string]any{"name": "alertname", "value": "test"},
				},
				"duration":  "1h",
				"createdBy": "SuperPlane",
				"comment":   "Test",
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeNone,
			}},
			Metadata:       &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "failed to create silence")
	})

	t.Run("execute sanitizes configuration", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"silenceID":"abc123"}`)),
				},
			},
		}

		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"matchers": []any{
					map[string]any{"name": "  alertname  ", "value": "  test  "},
				},
				"duration":  "  1h  ",
				"createdBy": "  SuperPlane  ",
				"comment":   "  Test  ",
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeNone,
			}},
			Metadata:       &contexts.MetadataContext{},
			ExecutionState: executionCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionCtx.Passed)
		require.Len(t, executionCtx.Payloads, 1)
		wrappedPayload := executionCtx.Payloads[0].(map[string]any)
		payload := wrappedPayload["data"].(map[string]any)
		matchers := payload["matchers"].([]map[string]any)
		assert.Equal(t, "alertname", matchers[0]["name"])
		assert.Equal(t, "test", matchers[0]["value"])
		assert.Equal(t, "SuperPlane", payload["createdBy"])
		assert.Equal(t, "Test", payload["comment"])
	})
}
