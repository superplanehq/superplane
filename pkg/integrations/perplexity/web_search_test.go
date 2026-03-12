package perplexity

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestWebSearch_Execute(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"id": "search-123",
					"results": [
						{"title": "Result 1", "url": "https://example.com/1", "snippet": "Snippet 1", "date": "2026-01-01"},
						{"title": "Result 2", "url": "https://example.com/2", "snippet": "Snippet 2", "date": "2026-01-02"}
					]
				}`)),
			},
		},
	}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "pplx-test"},
	}

	c := &webSearch{}
	err := c.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"query": "AI news"},
		ExecutionState: execState,
		HTTP:           httpCtx,
		Integration:    integrationCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, SearchPayloadType, execState.Type)
	require.Len(t, execState.Payloads, 1)

	wrapped := execState.Payloads[0].(map[string]any)
	data := wrapped["data"].(searchPayload)
	assert.Equal(t, "search-123", data.ID)
	assert.Equal(t, "AI news", data.Query)
	require.Len(t, data.Results, 2)
	assert.Equal(t, "Result 1", data.Results[0].Title)
	assert.Equal(t, "https://example.com/2", data.Results[1].URL)
}

func TestWebSearch_WithFilters(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"s1","results":[]}`)),
			},
		},
	}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "pplx-test"},
	}

	c := &webSearch{}
	err := c.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"query":         "golang news",
			"maxResults":    3,
			"domainFilter":  "go.dev, golang.org",
			"recencyFilter": "week",
		},
		ExecutionState: execState,
		HTTP:           httpCtx,
		Integration:    integrationCtx,
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)

	body, err := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, err)

	var sent SearchRequest
	require.NoError(t, json.Unmarshal(body, &sent))

	assert.Equal(t, "golang news", sent.Query)
	assert.Equal(t, 3, sent.MaxResults)
	assert.Equal(t, "week", sent.SearchRecencyFilter)
	require.Len(t, sent.SearchDomainFilter, 2)
	assert.Equal(t, "go.dev", sent.SearchDomainFilter[0])
	assert.Equal(t, "golang.org", sent.SearchDomainFilter[1])
}

func TestWebSearch_MissingQuery(t *testing.T) {
	c := &webSearch{}
	err := c.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"query": ""},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		HTTP:           &contexts.HTTPContext{},
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "query is required")
}

func TestWebSearch_APIError(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader(`{"error":"invalid query"}`)),
			},
		},
	}

	c := &webSearch{}
	err := c.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"query": "test"},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key"}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}
