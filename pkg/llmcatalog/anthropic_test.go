package llmcatalog_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/llmcatalog"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestAnthropicRefreshUsesAPILevelsAndKeepsOtherKeysModels(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		catalogResponse(http.StatusOK, `{"data":[
			{"id":"claude-sonnet-5-5","capabilities":{"effort":{"supported":true,"high":{"supported":true}}}},
			{"id":"claude-new-model","capabilities":{"effort":{"supported":true,"low":{"supported":true},"max":{"supported":true}}}},
			{"id":"claude-opus-5-5","capabilities":{"effort":{"supported":false}}}
		]}`),
		catalogResponse(http.StatusOK, `{"data":[
			{"id":"claude-sonnet-5-5","capabilities":{"effort":{"supported":true,"medium":{"supported":true},"high":{"supported":true}}}}
		]}`),
	}}
	svc := llmcatalog.New(nil, httpCtx)
	err := svc.RefreshAnthropicIntegration(t.Context(), db, &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key-one"}})
	require.NoError(t, err)
	err = svc.RefreshAnthropicIntegration(t.Context(), db, &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key-two"}})
	require.NoError(t, err)
	catalog, err := llmcatalog.LoadThinkingCatalog(db)
	require.NoError(t, err)
	assert.Equal(t, []string{"medium", "high"}, catalog.ThinkingLevels(models.UsageProviderAnthropic, "claude-sonnet-5-5", models.UsageFundingSourceBYOK))
	assert.Equal(t, []string{"low", "max"}, catalog.ThinkingLevels(models.UsageProviderAnthropic, "claude-new-model", models.UsageFundingSourceBYOK))
	assert.Empty(t, catalog.ThinkingLevels(models.UsageProviderAnthropic, "claude-opus-5-5", models.UsageFundingSourceBYOK))
	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, "key-two", httpCtx.Requests[1].Header.Get("x-api-key"))
}

func TestAnthropicRefreshRetainsCapabilitiesWhenUnavailable(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		catalogResponse(http.StatusOK, `{"data":[{"id":"claude-sonnet-5-5","capabilities":{"effort":{"supported":true,"high":{"supported":true}}}}]}`),
		catalogResponse(http.StatusServiceUnavailable, `{"error":{"message":"unavailable"}}`),
		catalogResponse(http.StatusOK, `{"data":[]}`),
		catalogResponse(http.StatusOK, `{"data":[{"id":"claude-sonnet-5-5"}]}`),
	}}
	svc := llmcatalog.New(nil, httpCtx)
	integration := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key-one"}}
	err := svc.RefreshAnthropicIntegration(t.Context(), db, integration)
	require.NoError(t, err)
	before, err := models.ListLLMModelCapabilities(db, models.UsageProviderAnthropic)
	require.NoError(t, err)
	require.Len(t, before, 1)

	for range 2 {
		err := svc.RefreshAnthropicIntegration(t.Context(), db, integration)
		require.Error(t, err)
	}
	err = svc.RefreshAnthropicIntegration(t.Context(), db, integration)
	require.NoError(t, err)
	after, err := models.ListLLMModelCapabilities(db, models.UsageProviderAnthropic)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}
