package llm

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestAnthropicModelsPreserveEffortCapabilitiesAcrossPages(t *testing.T) {
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(http.StatusOK, `{"data":[
			{"id":" claude-future ","capabilities":{"effort":{"supported":true,"low":{"supported":true},"medium":{"supported":false},"high":{"supported":true},"xhigh":{"supported":true},"max":{"supported":false}}}},
			{"id":"claude-no-effort","capabilities":{"effort":{"supported":false,"high":{"supported":true}}}}
		],"has_more":true,"last_id":"claude-no-effort"}`),
		jsonResponse(http.StatusOK, `{"data":[{"id":"legacy-model"},{"id":" "}],"has_more":false}`),
	}}
	client, err := New(httpCtx, ProviderAnthropic, Credentials{APIKey: "sk-test"})
	require.NoError(t, err)

	listed, err := client.ListModels(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []Model{
		{ID: "claude-future", ThinkingLevels: []string{"low", "high", "xhigh"}},
		{ID: "claude-no-effort", ThinkingLevels: []string{}},
		{ID: "legacy-model"},
	}, listed)
	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, "claude-no-effort", httpCtx.Requests[1].URL.Query().Get("after_id"))
	assert.Equal(t, "sk-test", httpCtx.Requests[1].Header.Get("x-api-key"))
}

func TestAnthropicModelsRejectIncompleteCatalogs(t *testing.T) {
	for _, tc := range []struct {
		name      string
		responses []*http.Response
	}{
		{name: "later page fails", responses: []*http.Response{
			jsonResponse(http.StatusOK, `{"data":[{"id":"first"}],"has_more":true,"last_id":"first"}`),
			jsonResponse(http.StatusServiceUnavailable, `{"error":{"message":"unavailable"}}`),
		}},
		{name: "missing cursor", responses: []*http.Response{
			jsonResponse(http.StatusOK, `{"data":[{"id":"first"}],"has_more":true}`),
		}},
		{name: "repeated cursor", responses: []*http.Response{
			jsonResponse(http.StatusOK, `{"data":[{"id":"first"}],"has_more":true,"last_id":"first"}`),
			jsonResponse(http.StatusOK, `{"data":[{"id":"first"}],"has_more":true,"last_id":"first"}`),
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, err := New(&contexts.HTTPContext{Responses: tc.responses}, ProviderAnthropic, Credentials{APIKey: "sk-test"})
			require.NoError(t, err)
			listed, err := client.ListModels(t.Context())
			require.Error(t, err)
			assert.Nil(t, listed)
		})
	}
}
