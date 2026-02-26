package prometheus

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

func Test__GetSilence__Setup(t *testing.T) {
	component := &GetSilence{}

	t.Run("silence is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"silence": "",
			},
		})
		require.ErrorContains(t, err, "silence is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"silence": "abc123",
			},
		})
		require.NoError(t, err)
	})

	t.Run("legacy silenceID setup still works", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"silenceID": "abc123",
			},
		})
		require.NoError(t, err)
	})
}

func Test__GetSilence__Execute(t *testing.T) {
	component := &GetSilence{}

	silenceJSON := `{
		"id": "abc123",
		"status": {"state": "active"},
		"matchers": [{"name": "alertname", "value": "HighLatency", "isRegex": false, "isEqual": true}],
		"startsAt": "2026-02-12T16:30:00Z",
		"endsAt": "2026-02-12T17:30:00Z",
		"createdBy": "SuperPlane",
		"comment": "Maintenance window",
		"updatedAt": "2026-02-12T16:30:00Z"
	}`

	t.Run("silence is retrieved and emitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(silenceJSON)),
				},
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"silence": "abc123",
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
		assert.Equal(t, "2026-02-12T16:30:00Z", payload["startsAt"])
		assert.Equal(t, "2026-02-12T17:30:00Z", payload["endsAt"])

		metadata := metadataCtx.Metadata.(GetSilenceNodeMetadata)
		assert.Equal(t, "abc123", metadata.SilenceID)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error":"silence not found"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"silence": "notexist",
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeNone,
			}},
			Metadata:       &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "failed to get silence")
	})
}
