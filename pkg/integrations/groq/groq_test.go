package groq

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestGroqSync(t *testing.T) {
	t.Run("marks the integration ready after key verification", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
			}},
		}
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "gsk-test"},
		}

		err := (&Groq{}).Sync(core.SyncContext{
			Configuration: integrationContext.Configuration,
			HTTP:          httpContext,
			Integration:   integrationContext,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationContext.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.groq.com/openai/v1/models", httpContext.Requests[0].URL.String())
	})

	t.Run("rejects an invalid key", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"error":"invalid api key"}`)),
			}},
		}
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "bad-key"},
		}

		err := (&Groq{}).Sync(core.SyncContext{
			Logger:        logrus.NewEntry(logrus.New()),
			Configuration: integrationContext.Configuration,
			HTTP:          httpContext,
			Integration:   integrationContext,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
		assert.NotEqual(t, "ready", integrationContext.State)
	})

	t.Run("requires an API key", func(t *testing.T) {
		err := (&Groq{}).Sync(core.SyncContext{
			Configuration: map[string]any{},
			HTTP:          &contexts.HTTPContext{},
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{}},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "apiKey is required")
	})
}

func TestGroqListResources(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"data": [
					{"id":"llama-3.3-70b-versatile","active":true},
					{"id":"whisper-large-v3","active":true},
					{"id":"canopylabs/orpheus-v1-english","active":true},
					{"id":"llama-3.1-8b-instant","active":false},
					{"id":"","active":true}
				]
			}`)),
		}},
	}
	integrationContext := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "gsk-test"}}

	resources, err := (&Groq{}).ListResources("model", core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: integrationContext,
	})

	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "model", resources[0].Type)
	assert.Equal(t, "llama-3.3-70b-versatile", resources[0].ID)
	assert.Equal(t, "llama-3.3-70b-versatile", resources[0].Name)
}

func TestGroqListResourcesUnknownType(t *testing.T) {
	resources, err := (&Groq{}).ListResources("unknown", core.ListResourcesContext{
		HTTP:        &contexts.HTTPContext{},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "gsk-test"}},
	})

	require.NoError(t, err)
	assert.Empty(t, resources)
}
