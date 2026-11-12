package llmcatalog_test

import (
	"fmt"
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
	"gorm.io/gorm"
)

func TestRefreshReplacesCapabilitiesAndPreservesCacheAfterFailures(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		catalogResponse(200, `{"data":[{"id":"openrouter/DeepSeek/MODEL","reasoning":{"supported_efforts":["high"],"default_effort":"high","mandatory":true}},{"id":"legacy/model"}]}`),
		catalogResponse(502, `{"error":"unavailable"}`),
		catalogResponse(200, `{"data":[]}`),
		catalogResponse(200, `{"data":[{"id":"duplicate/model"},{"id":"duplicate/model"}]}`),
		catalogResponse(200, `{"data":[{"id":"deepseek/model","reasoning":{"supported_efforts":["high","xhigh"],"default_effort":"xhigh"}}]}`),
	}}
	svc := llmcatalog.New(crypto.NewNoOpEncryptor(), httpCtx)
	provider := models.HostedLLMProvider{Provider: models.UsageProviderOpenRouter}
	started := time.Now()
	require.NoError(t, svc.Refresh(t.Context(), db, provider))
	before, err := models.ListLLMModelCapabilities(db, provider.Provider)
	require.NoError(t, err)
	require.Len(t, before, 2)
	stored := capability(t, db, provider.Provider, llm.DefaultOpenRouterBaseURL, "deepseek/model")
	assert.Equal(t, datatypes.JSONSlice[string]{"high"}, stored.ThinkingLevels)
	assert.Equal(t, "high", stored.DefaultThinkingLevel)
	assert.True(t, stored.ReasoningMandatory)
	assert.WithinRange(t, stored.RefreshedAt, started.Add(-time.Second), time.Now().Add(time.Second))
	assert.Nil(t, capability(t, db, provider.Provider, llm.DefaultOpenRouterBaseURL, "legacy/model").ThinkingLevels)
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
	assert.Equal(t, datatypes.JSONSlice[string]{"high", "xhigh"}, after[0].ThinkingLevels)
	assert.Equal(t, "xhigh", after[0].DefaultThinkingLevel)
}

func TestRefreshPreservesOmittedEffortsAndClearsExplicitEmptyEfforts(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		catalogResponse(200, `{"data":[{"id":"cached/model","reasoning":{"supported_efforts":["high"],"default_effort":"high","mandatory":true}}]}`),
		catalogResponse(200, `{"data":[{"id":"cached/model"}]}`),
		catalogResponse(200, `{"data":[{"id":"cached/model","reasoning":{"supported_efforts":[]}}]}`),
	}}
	svc := llmcatalog.New(nil, httpCtx)
	provider := models.HostedLLMProvider{Provider: models.UsageProviderOpenRouter}
	require.NoError(t, svc.Refresh(t.Context(), db, provider))
	require.NoError(t, svc.Refresh(t.Context(), db, provider))
	stored := capability(t, db, provider.Provider, llm.DefaultOpenRouterBaseURL, "cached/model")
	assert.Equal(t, datatypes.JSONSlice[string]{"high"}, stored.ThinkingLevels)
	assert.Equal(t, "high", stored.DefaultThinkingLevel)
	assert.True(t, stored.ReasoningMandatory)
	require.NoError(t, svc.Refresh(t.Context(), db, provider))
	stored = capability(t, db, provider.Provider, llm.DefaultOpenRouterBaseURL, "cached/model")
	assert.NotNil(t, stored.ThinkingLevels)
	assert.Empty(t, stored.ThinkingLevels)
	assert.Empty(t, stored.DefaultThinkingLevel)
	assert.False(t, stored.ReasoningMandatory)
}

func TestRefreshExpandsNullEfforts(t *testing.T) {
	for _, mandatory := range []bool{false, true} {
		t.Run(fmt.Sprint(mandatory), func(t *testing.T) {
			support.Setup(t)
			db := database.DB(t.Context())
			svc := llmcatalog.New(nil, &contexts.HTTPContext{Responses: []*http.Response{
				catalogResponse(200, fmt.Sprintf(`{"data":[{"id":"model","reasoning":{"supported_efforts":null,"mandatory":%t}}]}`, mandatory)),
			}})
			require.NoError(t, svc.Refresh(t.Context(), db, models.HostedLLMProvider{Provider: models.UsageProviderOpenRouter}))
			want := []string{"none", "minimal", "low", "medium", "high", "xhigh", "max"}
			if mandatory {
				want = want[1:]
			}
			assert.Equal(t, want, []string(capability(t, db, models.UsageProviderOpenRouter, llm.DefaultOpenRouterBaseURL, "model").ThinkingLevels))
		})
	}
}

func TestRefreshConfiguredLoadsPublicCatalogWithoutHostedSetup(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{catalogResponse(200, `{"data":[{"id":"public/model","reasoning":{"supported_efforts":["high"]}}]}`)}}
	require.NoError(t, llmcatalog.New(nil, httpCtx).RefreshConfigured(t.Context(), db))
	assert.Equal(t, datatypes.JSONSlice[string]{"high"}, capability(t, db, models.UsageProviderOpenRouter, llm.DefaultOpenRouterBaseURL, "public/model").ThinkingLevels)
	require.Len(t, httpCtx.Requests, 1)
	assert.Empty(t, httpCtx.Requests[0].Header.Get("Authorization"))
}

func TestRefreshConfiguredKeepsSourcesIndependentAfterFailureAndEndpointChange(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	provider, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{Provider: models.UsageProviderOpenRouter, APIKey: []byte("saved-key"), BaseURL: " https://proxy.example/v1/// "})
	require.NoError(t, err)
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		catalogResponse(200, `{"data":[{"id":"model","reasoning":{"supported_efforts":["high"]}}]}`),
		catalogResponse(401, `{"error":"expired"}`),
		catalogResponse(200, `{"data":[{"id":"model","reasoning":{"supported_efforts":["low"]}}]}`),
		catalogResponse(200, `{"data":[{"id":"model","reasoning":{"supported_efforts":[]}}]}`),
	}}
	svc := llmcatalog.New(crypto.NewNoOpEncryptor(), httpCtx)
	require.NoError(t, svc.Refresh(t.Context(), db, *provider))
	require.Error(t, svc.RefreshConfigured(t.Context(), db))
	assert.Equal(t, datatypes.JSONSlice[string]{"high"}, capability(t, db, provider.Provider, "https://proxy.example/v1", "model").ThinkingLevels)
	assert.Equal(t, datatypes.JSONSlice[string]{"low"}, capability(t, db, provider.Provider, llm.DefaultOpenRouterBaseURL, "model").ThinkingLevels)
	require.Len(t, httpCtx.Requests, 3)
	assert.Equal(t, "https://proxy.example/v1/models", httpCtx.Requests[1].URL.String())
	assert.Equal(t, "Bearer saved-key", httpCtx.Requests[1].Header.Get("Authorization"))
	assert.Equal(t, llm.DefaultOpenRouterBaseURL+"/models", httpCtx.Requests[2].URL.String())
	assert.Empty(t, httpCtx.Requests[2].Header.Get("Authorization"))
	provider.BaseURL = "https://other.example/v1"
	require.NoError(t, svc.Refresh(t.Context(), db, *provider))
	assert.Empty(t, capability(t, db, provider.Provider, provider.BaseURL, "model").ThinkingLevels)
	assert.Equal(t, datatypes.JSONSlice[string]{"high"}, capability(t, db, provider.Provider, "https://proxy.example/v1", "model").ThinkingLevels)
}

func capability(t *testing.T, db *gorm.DB, provider, endpoint, model string) models.LLMModelCapability {
	t.Helper()
	var row models.LLMModelCapability
	require.NoError(t, db.Where("provider = ? AND base_url = ? AND model = ?", provider, endpoint, model).First(&row).Error)
	return row
}

func catalogResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
