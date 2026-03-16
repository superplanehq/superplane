package perplexity

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

func TestSync_Success(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"object":"list","data":[]}`)),
			},
		},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "pplx-test-key"},
	}

	p := &Perplexity{}
	err := p.Sync(core.SyncContext{
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: map[string]any{"apiKey": "pplx-test-key"},
		HTTP:          httpCtx,
		Integration:   integrationCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, "ready", integrationCtx.State)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
	assert.Contains(t, httpCtx.Requests[0].URL.String(), "/v1/models")
}

func TestSync_InvalidKey(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"error":"invalid api key"}`)),
			},
		},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "bad-key"},
	}

	p := &Perplexity{}
	err := p.Sync(core.SyncContext{
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: map[string]any{"apiKey": "bad-key"},
		HTTP:          httpCtx,
		Integration:   integrationCtx,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
	assert.NotEqual(t, "ready", integrationCtx.State)
}

func TestSync_MissingAPIKey(t *testing.T) {
	p := &Perplexity{}
	err := p.Sync(core.SyncContext{
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: map[string]any{"apiKey": ""},
		HTTP:          &contexts.HTTPContext{},
		Integration:   &contexts.IntegrationContext{Configuration: map[string]any{}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "apiKey is required")
}

func TestListResources_Presets(t *testing.T) {
	p := &Perplexity{}
	resources, err := p.ListResources("agent-preset", core.ListResourcesContext{
		Logger:      logrus.NewEntry(logrus.New()),
		HTTP:        &contexts.HTTPContext{},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key"}},
	})

	require.NoError(t, err)
	require.Len(t, resources, 4)

	ids := make([]string, 0, len(resources))
	for _, r := range resources {
		ids = append(ids, r.ID)
	}
	assert.Contains(t, ids, "fast-search")
	assert.Contains(t, ids, "pro-search")
	assert.Contains(t, ids, "deep-research")
	assert.Contains(t, ids, "advanced-deep-research")
}

func TestListResources_Models(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"object": "list",
					"data": [
						{"id": "anthropic/claude-haiku-4-5", "owned_by": "anthropic"},
						{"id": "openai/gpt-5.2", "owned_by": "openai"}
					]
				}`)),
			},
		},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "pplx-key"},
	}

	p := &Perplexity{}
	resources, err := p.ListResources("agent-model", core.ListResourcesContext{
		Logger:      logrus.NewEntry(logrus.New()),
		HTTP:        httpCtx,
		Integration: integrationCtx,
	})

	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, "anthropic/claude-haiku-4-5", resources[0].ID)
	assert.Equal(t, "openai/gpt-5.2", resources[1].ID)
}

func TestListResources_UnknownType(t *testing.T) {
	p := &Perplexity{}
	resources, err := p.ListResources("unknown-type", core.ListResourcesContext{
		Logger:      logrus.NewEntry(logrus.New()),
		HTTP:        &contexts.HTTPContext{},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key"}},
	})

	require.NoError(t, err)
	assert.Empty(t, resources)
}
