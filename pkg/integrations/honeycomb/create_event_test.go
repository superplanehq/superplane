package honeycomb

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

func Test__CreateEvent__Setup(t *testing.T) {
	component := &CreateEvent{}

	t.Run("missing dataset -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"dataset": "",
				"fields":  `{"key":"value"}`,
			},
		})
		require.ErrorContains(t, err, "dataset is required")
	})

	t.Run("invalid JSON fields -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"dataset": "test-dataset",
				"fields":  `{invalid json}`,
			},
		})
		require.ErrorContains(t, err, "invalid fields json")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"dataset": "test-dataset",
				"fields":  `{"message":"hello","severity":"info"}`,
			},
		})
		require.NoError(t, err)
	})
}

func Test__CreateEvent__Execute(t *testing.T) {
	component := &CreateEvent{}

	t.Run("missing API key -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}

		err := component.Execute(core.ExecutionContext{
			Integration: integrationCtx,
			Configuration: map[string]any{
				"dataset": "test-dataset",
				"fields":  `{"key":"value"}`,
			},
			HTTP: &contexts.HTTPContext{},
		})

		require.ErrorContains(t, err, "api key is required")
	})

	t.Run("successful event creation -> emits payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
				"site":   "api.honeycomb.io",
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			HTTP:           httpCtx,
			Configuration: map[string]any{
				"dataset": "test-dataset",
				"fields":  `{"message":"deployment","version":"1.2.3"}`,
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, "honeycomb.event.created", execState.Type)

		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Contains(t, req.URL.String(), "https://api.honeycomb.io/1/events/test-dataset")
		assert.Equal(t, "test-api-key", req.Header.Get("X-Honeycomb-Team"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

		bodyBytes, _ := io.ReadAll(req.Body)
		bodyStr := strings.TrimSpace(string(bodyBytes))

		assert.True(t, strings.HasPrefix(bodyStr, "{"), "payload should be a JSON object")
		assert.Contains(t, bodyStr, `"message":"deployment"`)
		assert.Contains(t, bodyStr, `"version":"1.2.3"`)

		assert.NotEmpty(t, req.Header.Get("X-Honeycomb-Event-Time"), "event time is sent via header")

	})
}
