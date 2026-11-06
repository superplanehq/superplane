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
	for _, fundingSource := range []string{models.UsageFundingSourceBYOK, models.UsageFundingSourceHosted} {
		t.Run(fundingSource, func(t *testing.T) {
			for _, tc := range []struct {
				name   string
				model  string
				levels []string
			}{
				{name: "cached levels", model: "claude-sonnet-5-5", levels: []string{"medium", "high"}},
				{name: "new model", model: "claude-new-model", levels: []string{"low", "max"}},
				{name: "effort disabled", model: "claude-opus-5-5", levels: []string{}},
				{name: "missing metadata", model: "claude-opus-4-5-20251101", levels: []string{"low", "medium", "high"}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					assert.Equal(t, tc.levels, catalog.ThinkingLevels(models.UsageProviderAnthropic, tc.model, fundingSource))
				})
			}
		})
	}
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
