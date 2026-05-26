package cloudflare

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OriginRule__BuildExpression(t *testing.T) {
	t.Run("all incoming requests", func(t *testing.T) {
		expression, err := buildOriginExpression(originRuleMatchAll, nil, "")
		require.NoError(t, err)
		assert.Equal(t, "true", expression)
	})

	t.Run("custom predicates", func(t *testing.T) {
		expression, err := buildOriginExpression(originRuleMatchCustom, []OriginRuleMatchRule{
			{Field: "fullUri", Operator: "wildcard", Value: "/*", Conjunction: "and"},
			{Field: "uriPath", Operator: "startsWith", Value: "/api/"},
		}, "")
		require.NoError(t, err)
		assert.Equal(t, `(http.request.full_uri wildcard r"/*" and starts_with(http.request.uri.path, "/api/"))`, expression)
	})

	t.Run("raw expression remains supported", func(t *testing.T) {
		expression, err := buildOriginExpression("", nil, `http.host eq "example.com"`)
		require.NoError(t, err)
		assert.Equal(t, `http.host eq "example.com"`, expression)
	})
}

func originRuleResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func requestJSON(t *testing.T, request *http.Request) map[string]any {
	t.Helper()

	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	return payload
}

func cloudflareIntegrationContext() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "token123",
		},
		Metadata: Metadata{
			Zones: []Zone{{ID: "zone123", Name: "example.com", Status: "active"}},
		},
	}
}
