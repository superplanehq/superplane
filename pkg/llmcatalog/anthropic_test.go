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
	"gorm.io/datatypes"
)

func TestAnthropicRefreshMergesRestrictedCredentialsAndPreservesOmittedMetadata(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		catalogResponse(200, `{"data":[{"id":"claude-model","capabilities":{"effort":{"supported":true,"high":{"supported":true}}}},{"id":"other-key-model","capabilities":{"effort":{"supported":true,"max":{"supported":true}}}},{"id":"legacy-model"}]}`),
		catalogResponse(200, `{"data":[{"id":"claude-model"}]}`),
		catalogResponse(200, `{"data":[{"id":"claude-model","capabilities":{"effort":{"supported":false}}}]}`),
	}}
	svc := llmcatalog.New(nil, httpCtx)
	for _, key := range []string{"key-one", "restricted-key"} {
		require.NoError(t, svc.RefreshAnthropicIntegration(t.Context(), db, &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": key}}))
	}
	rows, err := models.ListLLMModelCapabilities(db, models.UsageProviderAnthropic)
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, datatypes.JSONSlice[string]{"high"}, capability(t, db, models.UsageProviderAnthropic, llm.DefaultAnthropicBaseURL, "claude-model").ThinkingLevels)
	assert.Equal(t, datatypes.JSONSlice[string]{"max"}, capability(t, db, models.UsageProviderAnthropic, llm.DefaultAnthropicBaseURL, "other-key-model").ThinkingLevels)
	assert.Nil(t, capability(t, db, models.UsageProviderAnthropic, llm.DefaultAnthropicBaseURL, "legacy-model").ThinkingLevels)
	require.NoError(t, svc.RefreshAnthropicIntegration(t.Context(), db, &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "restricted-key"}}))
	stored := capability(t, db, models.UsageProviderAnthropic, llm.DefaultAnthropicBaseURL, "claude-model")
	assert.NotNil(t, stored.ThinkingLevels)
	assert.Empty(t, stored.ThinkingLevels)
	assert.Equal(t, "restricted-key", httpCtx.Requests[1].Header.Get("x-api-key"))
}

func TestAnthropicRefreshPreservesCacheAfterIncompleteFetch(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		catalogResponse(200, `{"data":[{"id":"cached","capabilities":{"effort":{"supported":true,"high":{"supported":true}}}}]}`),
		catalogResponse(503, `{"error":{"message":"unavailable"}}`),
		catalogResponse(200, `{"data":[]}`),
		catalogResponse(200, `{"data":[{"id":"new"}],"has_more":true,"last_id":"new"}`),
		catalogResponse(503, `{"error":{"message":"unavailable"}}`),
	}}
	svc := llmcatalog.New(nil, httpCtx)
	integration := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "key"}}
	require.NoError(t, svc.RefreshAnthropicIntegration(t.Context(), db, integration))
	before, err := models.ListLLMModelCapabilities(db, models.UsageProviderAnthropic)
	require.NoError(t, err)
	for _, failure := range []string{"provider error", "empty", "later page error"} {
		t.Run(failure, func(t *testing.T) {
			require.Error(t, svc.RefreshAnthropicIntegration(t.Context(), db, integration))
			after, err := models.ListLLMModelCapabilities(db, models.UsageProviderAnthropic)
			require.NoError(t, err)
			assert.ElementsMatch(t, before, after)
		})
	}
}

func TestConfiguredHostedAnthropicPopulatesFreshCacheAndAttemptsIndependentSources(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{Provider: models.UsageProviderAnthropic, APIKey: []byte("hosted-key")})
	require.NoError(t, err)
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		catalogResponse(503, `{"error":"unavailable"}`),
		catalogResponse(200, `{"data":[{"id":"hosted-model","capabilities":{"effort":{"supported":true,"high":{"supported":true}}}}]}`),
	}}
	require.Error(t, llmcatalog.New(crypto.NewNoOpEncryptor(), httpCtx).RefreshConfigured(t.Context(), db))
	assert.Equal(t, datatypes.JSONSlice[string]{"high"}, capability(t, db, models.UsageProviderAnthropic, llm.DefaultAnthropicBaseURL, "hosted-model").ThinkingLevels)
	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, "hosted-key", httpCtx.Requests[1].Header.Get("x-api-key"))
}

func TestAnthropicRefreshKeepsProxyAndPublicSourcesIndependent(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		catalogResponse(200, `{"data":[{"id":"model","capabilities":{"effort":{"supported":false}}}]}`),
		catalogResponse(200, `{"data":[{"id":"model","capabilities":{"effort":{"supported":true,"max":{"supported":true}}}}]}`),
	}}
	svc := llmcatalog.New(crypto.NewNoOpEncryptor(), httpCtx)
	require.NoError(t, svc.Refresh(t.Context(), db, models.HostedLLMProvider{Provider: models.UsageProviderAnthropic, APIKey: []byte("hosted-key"), BaseURL: " https://proxy.example/v1/// "}))
	require.NoError(t, svc.RefreshAnthropicIntegration(t.Context(), db, &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "byok-key"}}))
	proxy := capability(t, db, models.UsageProviderAnthropic, "https://proxy.example/v1", "model")
	assert.NotNil(t, proxy.ThinkingLevels)
	assert.Empty(t, proxy.ThinkingLevels)
	assert.Equal(t, datatypes.JSONSlice[string]{"max"}, capability(t, db, models.UsageProviderAnthropic, llm.DefaultAnthropicBaseURL, "model").ThinkingLevels)
	assert.Equal(t, "https://proxy.example/v1/models", httpCtx.Requests[0].URL.String())
	assert.Equal(t, llm.DefaultAnthropicBaseURL+"/models", httpCtx.Requests[1].URL.String())
}
