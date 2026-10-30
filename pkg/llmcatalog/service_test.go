package llmcatalog_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/llmcatalog"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
)

func TestRefreshReplacesCapabilitiesAndPreservesCacheAfterFailures(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		catalogResponse(http.StatusOK, `{"data":[
			{"id":"openrouter/DeepSeek/DEEPSEEK-v4-flash","pricing":{"prompt":"0.000001"},"reasoning":{"supported_efforts":["none","high"],"default_effort":"high","mandatory":true}},
			{"id":"legacy/model"}
		]}`),
		catalogResponse(http.StatusBadGateway, `{"error":"unavailable"}`),
		catalogResponse(http.StatusOK, `{"data":[]}`),
		catalogResponse(http.StatusOK, `{"data":[{"id":"duplicate/model"},{"id":"duplicate/model"}]}`),
		catalogResponse(http.StatusOK, `{"data":[
			{"id":"deepseek/deepseek-v4-flash","pricing":{"prompt":"0.000001"},"reasoning":{"supported_efforts":["none","high","xhigh"],"default_effort":"xhigh"}}
		]}`),
	}}
	svc := llmcatalog.New(crypto.NewNoOpEncryptor(), httpCtx)
	provider := models.HostedLLMProvider{Provider: models.UsageProviderOpenRouter}
	started := time.Now()
	require.NoError(t, svc.Refresh(t.Context(), db, provider))
	before, err := models.ListLLMModelCapabilities(db, provider.Provider)
	require.NoError(t, err)
	require.Len(t, before, 2)
	byModel := make(map[string]models.LLMModelCapability, len(before))
	for _, capability := range before {
		byModel[capability.Model] = capability
		assert.Equal(t, llm.DefaultOpenRouterBaseURL, capability.BaseURL)
		assert.WithinRange(t, capability.RefreshedAt, started.Add(-time.Second), time.Now().Add(time.Second))
		assert.Equal(t, before[0].RefreshedAt, capability.RefreshedAt)
	}
	deepseek := byModel["deepseek/deepseek-v4-flash"]
	assert.Equal(t, datatypes.JSONSlice[string]{"none", "high"}, deepseek.ThinkingLevels)
	assert.Equal(t, "high", deepseek.DefaultThinkingLevel)
	assert.True(t, deepseek.ReasoningMandatory)
	assert.Equal(t, datatypes.JSONSlice[string]{}, byModel["legacy/model"].ThinkingLevels)
	catalog, err := llmcatalog.LoadThinkingCatalog(db)
	require.NoError(t, err)
	assert.Equal(t, []string{"none", "high"}, catalog.ThinkingLevels(provider.Provider, deepseek.Model, models.UsageFundingSourceBYOK))
	assert.Empty(t, catalog.ThinkingLevels(provider.Provider, "legacy/model", models.UsageFundingSourceBYOK))

	for _, failure := range []string{"provider error", "empty catalog", "store error"} {
		t.Run(failure, func(t *testing.T) {
			require.Error(t, svc.Refresh(t.Context(), db, provider))
			after, err := models.ListLLMModelCapabilities(db, provider.Provider)
			require.NoError(t, err)
			assert.ElementsMatch(t, before, after)
		})
	}

	require.NoError(t, svc.Refresh(t.Context(), db, provider))
	after, err := models.ListLLMModelCapabilities(db, provider.Provider)
	require.NoError(t, err)
	require.Len(t, after, 1)
	assert.Equal(t, datatypes.JSONSlice[string]{"none", "high", "xhigh"}, after[0].ThinkingLevels)
	assert.Equal(t, "xhigh", after[0].DefaultThinkingLevel)
	assert.False(t, after[0].ReasoningMandatory)
	catalog, err = llmcatalog.LoadThinkingCatalog(db)
	require.NoError(t, err)
	assert.Equal(t, []string{"none", "high", "xhigh"}, catalog.ThinkingLevels(provider.Provider, deepseek.Model, models.UsageFundingSourceBYOK))
}

func TestRefreshConfiguredLoadsPublicCatalogWithoutHostedSetup(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, db.Where("provider = ?", models.UsageProviderOpenRouter).Delete(&models.HostedLLMProvider{}).Error)
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		catalogResponse(http.StatusOK, `{"data":[{"id":"deepseek/deepseek-v4-flash","reasoning":{"supported_efforts":["high","xhigh"],"default_effort":"high"}}]}`),
	}}
	svc := llmcatalog.New(nil, httpCtx)
	require.NoError(t, svc.RefreshConfigured(t.Context(), db))
	stored, err := models.ListLLMModelCapabilities(db, models.UsageProviderOpenRouter)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, llm.DefaultOpenRouterBaseURL, stored[0].BaseURL)
	assert.Equal(t, datatypes.JSONSlice[string]{"high", "xhigh"}, stored[0].ThinkingLevels)
	catalog, err := llmcatalog.LoadThinkingCatalog(db)
	require.NoError(t, err)
	assert.Equal(t, []string{"high", "xhigh"}, catalog.ThinkingLevels(models.UsageProviderOpenRouter, "deepseek/deepseek-v4-flash", models.UsageFundingSourceBYOK))
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, llm.DefaultOpenRouterBaseURL+"/models", httpCtx.Requests[0].URL.String())
	assert.Empty(t, httpCtx.Requests[0].Header.Get("Authorization"))
}

func TestRefreshConfiguredUsesPublicCatalogWhenSavedCredentialsFail(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())
	_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderOpenRouter,
		APIKey:        []byte("sk-expired"),
		AllowedModels: datatypes.JSONSlice[string]{"deepseek/deepseek-v4-flash"},
	})
	require.NoError(t, err)
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		catalogResponse(http.StatusUnauthorized, `{"error":"expired API key"}`),
		catalogResponse(http.StatusOK, `{"data":[{"id":"deepseek/deepseek-v4-flash","reasoning":{"supported_efforts":["high","xhigh"]}}]}`),
	}}
	svc := llmcatalog.New(crypto.NewNoOpEncryptor(), httpCtx)
	require.ErrorContains(t, svc.RefreshConfigured(t.Context(), db), "401")
	catalog, err := llmcatalog.LoadThinkingCatalog(db)
	require.NoError(t, err)
	assert.Equal(t, []string{"high", "xhigh"}, catalog.ThinkingLevels(models.UsageProviderOpenRouter, "deepseek/deepseek-v4-flash", models.UsageFundingSourceBYOK))
	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, llm.DefaultOpenRouterBaseURL+"/models", httpCtx.Requests[0].URL.String())
	assert.Equal(t, "Bearer sk-expired", httpCtx.Requests[0].Header.Get("Authorization"))
	assert.Equal(t, llm.DefaultOpenRouterBaseURL+"/models", httpCtx.Requests[1].URL.String())
	assert.Empty(t, httpCtx.Requests[1].Header.Get("Authorization"))
}

func TestRefreshConfiguredKeepsProviderSourcesIndependent(t *testing.T) {
	_ = support.Setup(t)
	db := database.DB(t.Context())
	provider, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderOpenRouter,
		Enabled:       false,
		APIKey:        []byte("sk-saved"),
		BaseURL:       " https://proxy.example/v1/// ",
		AllowedModels: datatypes.JSONSlice[string]{"deepseek/deepseek-v4-flash"},
	})
	require.NoError(t, err)
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		catalogResponse(http.StatusOK, `{"data":[{"id":"deepseek/deepseek-v4-flash","reasoning":{"supported_efforts":["high"]}}]}`),
		catalogResponse(http.StatusBadGateway, `{"error":"proxy unavailable"}`),
		catalogResponse(http.StatusOK, `{"data":[{"id":"deepseek/deepseek-v4-flash","reasoning":{"supported_efforts":["high","xhigh"]}}]}`),
	}}
	svc := llmcatalog.New(crypto.NewNoOpEncryptor(), httpCtx)
	require.NoError(t, svc.Refresh(t.Context(), db, *provider))
	require.Error(t, svc.RefreshConfigured(t.Context(), db))
	stored, err := models.ListLLMModelCapabilities(db, provider.Provider)
	require.NoError(t, err)
	require.Len(t, stored, 2)
	bySource := make(map[string][]string, len(stored))
	for _, capability := range stored {
		bySource[capability.BaseURL] = capability.ThinkingLevels
	}
	assert.Equal(t, []string{"high"}, bySource["https://proxy.example/v1"])
	assert.Equal(t, []string{"high", "xhigh"}, bySource[llm.DefaultOpenRouterBaseURL])
	catalog, err := llmcatalog.LoadThinkingCatalog(db)
	require.NoError(t, err)
	assert.Equal(t, []string{"high"}, catalog.ThinkingLevels(provider.Provider, "deepseek/deepseek-v4-flash", models.UsageFundingSourceHosted))
	assert.Equal(t, []string{"high", "xhigh"}, catalog.ThinkingLevels(provider.Provider, "deepseek/deepseek-v4-flash", models.UsageFundingSourceBYOK))
	require.Len(t, httpCtx.Requests, 3)
	assert.Equal(t, "https://proxy.example/v1/models", httpCtx.Requests[1].URL.String())
	assert.Equal(t, "Bearer sk-saved", httpCtx.Requests[1].Header.Get("Authorization"))
	assert.Equal(t, llm.DefaultOpenRouterBaseURL+"/models", httpCtx.Requests[2].URL.String())
	assert.Empty(t, httpCtx.Requests[2].Header.Get("Authorization"))
}

func catalogResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
