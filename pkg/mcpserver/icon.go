package mcpserver

import (
	_ "embed"
	"encoding/base64"
)

const ServerWebsiteURL = "https://superplane.com"

//go:embed icon-light.png
var iconLightPNG []byte

//go:embed icon-dark.png
var iconDarkPNG []byte

//go:embed icon-192.png
var icon192PNG []byte

func dataPNGURI(raw []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw)
}

func serverIcon(raw []byte, size, theme string) map[string]any {
	return map[string]any{
		"src":      dataPNGURI(raw),
		"mimeType": "image/png",
		"sizes":    []string{size},
		"theme":    theme,
	}
}

func ServerInfo() map[string]any {
	return map[string]any{
		"name":       ServerName,
		"title":      ServerName,
		"version":    ServerVersion,
		"websiteUrl": ServerWebsiteURL,
		"icons": []map[string]any{
			serverIcon(iconLightPNG, "96x96", "light"),
			serverIcon(iconDarkPNG, "96x96", "dark"),
			serverIcon(icon192PNG, "192x192", "light"),
			serverIcon(icon192PNG, "192x192", "dark"),
		},
	}
}
