package mcpserver

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleJSONRPCEmptyDiscoveryLists(t *testing.T) {
	t.Parallel()

	cases := []struct {
		method string
		key    string
	}{
		{method: "resources/list", key: "resources"},
		{method: "prompts/list", key: "prompts"},
		{method: "resources/templates/list", key: "resourceTemplates"},
	}
	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			resp := HandleJSONRPC(t.Context(), &Runtime{}, &AccessClaims{}, &JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  tc.method,
			})
			require.NotNil(t, resp)
			assert.Nil(t, resp.Error)
			result, ok := resp.Result.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, []any{}, result[tc.key])
		})
	}
}

func TestParseJSONRPCRejectsInvalidVersion(t *testing.T) {
	_, err := ParseJSONRPC(strings.NewReader(`{"jsonrpc":"1.0","id":1,"method":"ping"}`))
	require.Error(t, err)
}

func TestInitializeIncludesServerIcons(t *testing.T) {
	t.Parallel()

	resp := HandleJSONRPC(t.Context(), &Runtime{}, &AccessClaims{}, &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  []byte(`{"protocolVersion":"2025-06-18"}`),
	})
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	result, ok := resp.Result.(map[string]any)
	require.True(t, ok)
	info, ok := result["serverInfo"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, ServerName, info["name"])
	assert.Equal(t, ServerName, info["title"])
	assert.Equal(t, ServerVersion, info["version"])
	assert.Equal(t, ServerWebsiteURL, info["websiteUrl"])
	icons, ok := info["icons"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, icons, 2)
	assert.Equal(t, "image/png", icons[0]["mimeType"])
	assert.Equal(t, []string{"96x96"}, icons[0]["sizes"])
	assert.Equal(t, "light", icons[0]["theme"])
	assert.Equal(t, "dark", icons[1]["theme"])
	src, ok := icons[0]["src"].(string)
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(src, "data:image/png;base64,"))
	assert.Greater(t, len(src), 64)
}
