package llmcatalog_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/llm"
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

func TestConfiguredHostedAnthropicRefreshPopulatesAndPreservesCapabilities(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())
	_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider: models.UsageProviderAnthropic, APIKey: []byte("hosted-key"),
	})
	require.NoError(t, err)
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		catalogResponse(http.StatusOK, `{"data":[{"id":"public/model"}]}`),
		catalogResponse(http.StatusOK, `{"data":[{"id":"claude-haiku-5-5","capabilities":{"effort":{"supported":true,"low":{"supported":true},"high":{"supported":true}}}}]}`),
		catalogResponse(http.StatusOK, `{"data":[{"id":"public/model"}]}`),
		catalogResponse(http.StatusServiceUnavailable, `{"error":{"message":"unavailable"}}`),
	}}
	svc := llmcatalog.New(crypto.NewNoOpEncryptor(), httpCtx)
	require.NoError(t, svc.RefreshConfigured(t.Context(), db))
	catalog, err := llmcatalog.LoadThinkingCatalog(db)
	require.NoError(t, err)
	assert.Equal(t, []string{"low", "high"}, catalog.ThinkingLevels(models.UsageProviderAnthropic, "claude-haiku-5-5", models.UsageFundingSourceHosted))
	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, llm.DefaultAnthropicBaseURL+"/models", httpCtx.Requests[1].URL.String())
	assert.Equal(t, "hosted-key", httpCtx.Requests[1].Header.Get("x-api-key"))
	require.Error(t, svc.RefreshConfigured(t.Context(), db))
	catalog, err = llmcatalog.LoadThinkingCatalog(db)
	require.NoError(t, err)
	assert.Equal(t, []string{"low", "high"}, catalog.ThinkingLevels(models.UsageProviderAnthropic, "claude-haiku-5-5", models.UsageFundingSourceHosted))
}

func TestAnthropicRefreshKeepsCatalogSourcesIndependent(t *testing.T) {
	for _, hostedFirst := range []bool{true, false} {
		name := "BYOK first"
		if hostedFirst {
			name = "hosted first"
		}
		t.Run(name, func(t *testing.T) {
			_ = support.Setup(t)
			db := database.DB(t.Context())
			provider, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
				Provider: models.UsageProviderAnthropic, APIKey: []byte("hosted-key"),
				BaseURL: " https://anthropic-proxy.example/v1/// ",
			})
			require.NoError(t, err)
			hostedResponse := catalogResponse(http.StatusOK, `{"data":[
				{"id":"claude-opus-5-5","capabilities":{"effort":{"supported":false}}},
				{"id":"claude-sonnet-5-5","capabilities":{"effort":{"supported":true,"high":{"supported":true}}}},
				{"id":"claude-hosted-model","capabilities":{"effort":{"supported":true,"max":{"supported":true}}}}
			]}`)
			byokResponse := catalogResponse(http.StatusOK, `{"data":[
				{"id":"claude-opus-5-5","capabilities":{"effort":{"supported":true,"max":{"supported":true}}}},
				{"id":"claude-sonnet-5-5","capabilities":{"effort":{"supported":true,"low":{"supported":true}}}},
				{"id":"claude-byok-model","capabilities":{"effort":{"supported":true,"medium":{"supported":true}}}}
			]}`)
			httpCtx := &contexts.HTTPContext{}
			svc := llmcatalog.New(crypto.NewNoOpEncryptor(), httpCtx)
			refreshHosted := func() {
				httpCtx.Responses = append(httpCtx.Responses, hostedResponse)
				require.NoError(t, svc.Refresh(t.Context(), db, *provider))
				assert.Equal(t, "https://anthropic-proxy.example/v1/models", httpCtx.Requests[len(httpCtx.Requests)-1].URL.String())
			}
			refreshBYOK := func() {
				httpCtx.Responses = append(httpCtx.Responses, byokResponse)
				require.NoError(t, svc.RefreshAnthropicIntegration(t.Context(), db, &contexts.IntegrationContext{
					Configuration: map[string]any{"apiKey": "byok-key"},
				}))
				assert.Equal(t, llm.DefaultAnthropicBaseURL+"/models", httpCtx.Requests[len(httpCtx.Requests)-1].URL.String())
			}
			if hostedFirst {
				refreshHosted()
				refreshBYOK()
			} else {
				refreshBYOK()
				refreshHosted()
			}
			catalog, err := llmcatalog.LoadThinkingCatalog(db)
			require.NoError(t, err)
			for _, tc := range []struct {
				model        string
				hosted, byok []string
			}{
				{model: "claude-opus-5-5", hosted: []string{}, byok: []string{"max"}},
				{model: "claude-sonnet-5-5", hosted: []string{"high"}, byok: []string{"low"}},
				{model: "claude-hosted-model", hosted: []string{"max"}},
				{model: "claude-byok-model", byok: []string{"medium"}},
			} {
				assert.Equal(t, tc.hosted, catalog.ThinkingLevels(models.UsageProviderAnthropic, tc.model, models.UsageFundingSourceHosted), tc.model)
				assert.Equal(t, tc.byok, catalog.ThinkingLevels(models.UsageProviderAnthropic, tc.model, models.UsageFundingSourceBYOK), tc.model)
			}

			// Switching the hosted endpoint must not reuse the previous gateway's metadata.
			provider.BaseURL = "https://other-anthropic-proxy.example/v1"
			_, err = models.UpsertHostedLLMProvider(db, *provider)
			require.NoError(t, err)
			catalog, err = llmcatalog.LoadThinkingCatalog(db)
			require.NoError(t, err)
			assert.Empty(t, catalog.ThinkingLevels(models.UsageProviderAnthropic, "claude-hosted-model", models.UsageFundingSourceHosted))
			assert.Equal(t, []string{"low"}, catalog.ThinkingLevels(models.UsageProviderAnthropic, "claude-sonnet-5-5", models.UsageFundingSourceBYOK))
		})
	}
}
