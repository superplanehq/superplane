package mcpserver

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
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
	require.Len(t, icons, 3)

	expected := []struct {
		size  int
		theme any
	}{
		{size: 96, theme: "light"},
		{size: 96, theme: "dark"},
		{size: 192, theme: nil},
	}
	for i, want := range expected {
		icon := icons[i]
		assert.Equal(t, "image/png", icon["mimeType"])
		assert.Equal(t, []string{fmt.Sprintf("%dx%d", want.size, want.size)}, icon["sizes"])
		assert.Equal(t, want.theme, icon["theme"])

		src, ok := icon["src"].(string)
		require.True(t, ok)
		require.True(t, strings.HasPrefix(src, dataPNGPrefix))
		raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(src, dataPNGPrefix))
		require.NoError(t, err)
		cfg, err := png.DecodeConfig(bytes.NewReader(raw))
		require.NoError(t, err)
		assert.Equal(t, want.size, cfg.Width)
		assert.Equal(t, want.size, cfg.Height)
	}
}
