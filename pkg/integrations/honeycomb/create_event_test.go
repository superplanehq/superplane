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
				"fields":  map[string]any{"key": "value"},
			},
		})
		require.ErrorContains(t, err, "dataset is required")
	})

	t.Run("missing fields -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"dataset": "test-dataset",
				"fields":  map[string]any{},
			},
		})
		require.ErrorContains(t, err, "fields json is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"dataset": "test-dataset",
				"fields":  map[string]any{"message": "hello", "severity": "info"},
			},
		})
		require.NoError(t, err)
	})
}

func Test__CreateEvent__Execute(t *testing.T) {
	component := &CreateEvent{}

	t.Run("missing managementKey -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site": "api.honeycomb.io",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration: integrationCtx,
			Configuration: map[string]any{
				"dataset": "test-dataset",
				"fields":  map[string]any{"key": "value"},
			},
			HTTP: &contexts.HTTPContext{},
		})

		require.ErrorContains(t, err, "managementKey is required")
	})

	t.Run("missing ingest key secret -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"managementKey": "keyid:secret",
				"site":          "api.honeycomb.io",
			},
			Secrets: map[string]core.IntegrationSecret{},
		}

		err := component.Execute(core.ExecutionContext{
			Integration: integrationCtx,
			HTTP:        &contexts.HTTPContext{},
			Configuration: map[string]any{
				"dataset": "test-dataset",
				"fields":  map[string]any{"key": "value"},
			},
		})

		require.ErrorContains(t, err, "ingest key not found")
	})

	t.Run("API returns error -> Execute fails", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error":"unauthorized"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"managementKey": "keyid:secret",
				"site":          "api.honeycomb.io",
			},
			Secrets: map[string]core.IntegrationSecret{
				secretNameIngestKey: {Name: secretNameIngestKey, Value: []byte("test-ingest-key")},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration: integrationCtx,
			HTTP:        httpCtx,
			Configuration: map[string]any{
				"dataset": "test-dataset",
				"fields":  map[string]any{"key": "value"},
			},
		})

		require.ErrorContains(t, err, "401")
	})

	t.Run("successful event creation without time field -> emits payload and sets header", func(t *testing.T) {
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
				"managementKey": "keyid:secret",
				"site":          "api.honeycomb.io",
			},
			Secrets: map[string]core.IntegrationSecret{
				secretNameIngestKey: {Name: secretNameIngestKey, Value: []byte("test-ingest-key")},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			HTTP:           httpCtx,
			Configuration: map[string]any{
				"dataset": "test-dataset",
				"fields":  map[string]any{"message": "deployment", "version": "1.2.3"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, "honeycomb.event.created", execState.Type)

		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Contains(t, req.URL.String(), "https://api.honeycomb.io/1/events/test-dataset")
		assert.Equal(t, "test-ingest-key", req.Header.Get("X-Honeycomb-Team"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

		bodyBytes, _ := io.ReadAll(req.Body)
		bodyStr := strings.TrimSpace(string(bodyBytes))
		assert.True(t, strings.HasPrefix(bodyStr, "{"), "payload should be a JSON object")
		assert.Contains(t, bodyStr, `"message":"deployment"`)
		assert.Contains(t, bodyStr, `"version":"1.2.3"`)

		assert.NotEmpty(t, req.Header.Get("X-Honeycomb-Event-Time"), "event time header should be set when time field is not provided")
	})

	t.Run("successful event creation with time field -> emits payload without header", func(t *testing.T) {
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
				"managementKey": "keyid:secret",
				"site":          "api.honeycomb.io",
			},
			Secrets: map[string]core.IntegrationSecret{
				secretNameIngestKey: {Name: secretNameIngestKey, Value: []byte("test-ingest-key")},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: execState,
			HTTP:           httpCtx,
			Configuration: map[string]any{
				"dataset": "test-dataset",
				"fields":  map[string]any{"message": "deployment", "version": "1.2.3", "time": "2024-01-15T10:30:00Z"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, "honeycomb.event.created", execState.Type)

		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Contains(t, req.URL.String(), "https://api.honeycomb.io/1/events/test-dataset")
		assert.Equal(t, "test-ingest-key", req.Header.Get("X-Honeycomb-Team"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

		bodyBytes, _ := io.ReadAll(req.Body)
		bodyStr := strings.TrimSpace(string(bodyBytes))
		assert.True(t, strings.HasPrefix(bodyStr, "{"), "payload should be a JSON object")
		assert.Contains(t, bodyStr, `"message":"deployment"`)
		assert.Contains(t, bodyStr, `"version":"1.2.3"`)
		assert.Contains(t, bodyStr, `"time":"2024-01-15T10:30:00Z"`)

		assert.Empty(t, req.Header.Get("X-Honeycomb-Event-Time"), "event time header should not be set when time field is provided")
	})
}
