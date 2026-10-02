package customllm

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestListModelsUsesBearerTokenForOpenAICompatible(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"kimi-k3","name":"Kimi K3"}]}`)),
			Header:     make(http.Header),
		}},
	}
	client := &Client{
		APIKey:  "token",
		BaseURL: "https://models.example/v1",
		APIType: APITypeOpenAICompatible,
		http:    httpCtx,
	}

	models, err := client.ListModels()
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, "kimi-k3", models[0].ID)
	assert.Equal(t, "Kimi K3", models[0].Name)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "https://models.example/v1/models", httpCtx.Requests[0].URL.String())
	assert.Equal(t, "Bearer token", httpCtx.Requests[0].Header.Get("Authorization"))
}

func TestListModelsUsesAnthropicHeaders(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"claude-sonnet-4-6","display_name":"Claude Sonnet"}]}`)),
			Header:     make(http.Header),
		}},
	}
	client := &Client{
		APIKey:  "token",
		BaseURL: "https://models.example/v1",
		APIType: APITypeAnthropic,
		http:    httpCtx,
	}

	models, err := client.ListModels()
	require.NoError(t, err)
	require.Equal(t, "Claude Sonnet", models[0].Name)
	assert.Equal(t, "token", httpCtx.Requests[0].Header.Get("x-api-key"))
	assert.Equal(t, anthropicVersion, httpCtx.Requests[0].Header.Get("anthropic-version"))
	assert.Empty(t, httpCtx.Requests[0].Header.Get("Authorization"))
}

func TestNormalizeAPIType(t *testing.T) {
	parsed, err := NormalizeAPIType(" OpenAI-Compatible ")
	require.NoError(t, err)
	assert.Equal(t, APITypeOpenAICompatible, parsed)

	_, err = NormalizeAPIType("other")
	require.Error(t, err)
}
