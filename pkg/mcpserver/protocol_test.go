package mcpserver

import (
	"bytes"
	"encoding/base64"
	"image/color"
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
	require.Len(t, icons, 4)

	want := []struct {
		size  string
		theme string
		px    int
	}{
		{size: "96x96", theme: "light", px: 96},
		{size: "96x96", theme: "dark", px: 96},
		{size: "192x192", theme: "light", px: 192},
		{size: "192x192", theme: "dark", px: 192},
	}
	for i, tc := range want {
		icon := icons[i]
		assert.Equal(t, "image/png", icon["mimeType"])
		assert.Equal(t, []string{tc.size}, icon["sizes"])
		assert.Equal(t, tc.theme, icon["theme"])
		assertOrangeBrandTile(t, icon["src"], tc.px)
	}
	assert.Equal(t, icons[0]["src"], icons[1]["src"])
	assert.Equal(t, icons[2]["src"], icons[3]["src"])
}

func assertOrangeBrandTile(t *testing.T, src any, size int) {
	t.Helper()

	raw, ok := src.(string)
	require.True(t, ok)
	const prefix = "data:image/png;base64,"
	require.True(t, strings.HasPrefix(raw, prefix))
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(raw, prefix))
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(decoded))
	require.NoError(t, err)
	bounds := img.Bounds()
	assert.Equal(t, size, bounds.Dx())
	assert.Equal(t, size, bounds.Dy())
	assert.Equal(t, color.RGBA{R: 237, G: 142, B: 9, A: 255}, color.RGBAModel.Convert(img.At(bounds.Min.X, bounds.Min.Y)))
	center := img.At(bounds.Min.X+bounds.Dx()/2, bounds.Min.Y+bounds.Dy()/2)
	r, g, b, _ := center.RGBA()
	assert.Less(t, int(r>>8)+int(g>>8)+int(b>>8), 40)
}
