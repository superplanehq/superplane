package pricesync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestParseOpenRouterModels(t *testing.T) {
	body := []byte(`{
		"data": [
			{
				"id": "anthropic/claude-sonnet-4",
				"pricing": {
					"prompt": "0.000003",
					"completion": "0.000015",
					"input_cache_read": "0.0000003",
					"input_cache_write": "0.00000375"
				}
			},
			{"id": "", "pricing": {"prompt": "0.000001"}},
			{
				"id": "anthropic/claude-sonnet-4",
				"pricing": {"prompt": "0.000009"}
			}
		]
	}`)

	rates, err := parseOpenRouterModels(body)
	require.NoError(t, err)
	require.Len(t, rates, 1)
	assert.Equal(t, "anthropic/claude-sonnet-4", rates[0].MatchKey)
	assert.Equal(t, models.UsagePriceBookMatchExact, rates[0].MatchMode)
	assert.Equal(t, int64(300), rates[0].Rate.Input)
	assert.Equal(t, int64(1500), rates[0].Rate.Output)
	assert.Equal(t, int64(30), rates[0].Rate.CacheRead)
	assert.Equal(t, int64(375), rates[0].Rate.CacheWrite)
}

func TestOpenRouterSourceScanModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"openai/gpt-4o","pricing":{"prompt":"0.0000025","completion":"0.00001"}}]}`))
	}))
	t.Cleanup(server.Close)

	source := OpenRouterSource{HTTP: http.DefaultClient, URL: server.URL}
	rates, err := source.ScanModels(context.Background())
	require.NoError(t, err)
	require.Len(t, rates, 1)
	assert.Equal(t, "openai/gpt-4o", rates[0].MatchKey)
	assert.Equal(t, int64(250), rates[0].Rate.Input)
	assert.Equal(t, int64(1000), rates[0].Rate.Output)
}

func TestOpenRouterSourceScanModels_EmptyCatalog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	t.Cleanup(server.Close)

	source := OpenRouterSource{HTTP: http.DefaultClient, URL: server.URL}
	_, err := source.ScanModels(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no model rates")
}
